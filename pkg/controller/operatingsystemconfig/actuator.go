// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	ignv3_3 "github.com/coreos/ignition/v2/config/v3_3"
	igntypes "github.com/coreos/ignition/v2/config/v3_3/types"
	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	runtimeutils "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	configv1alpha1 "github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1"
)

//go:embed templates/configure-cgroupsv2.sh.tpl
var cgroupsv2TemplateContent string

//go:embed templates/ntp-config.conf.tpl
var ntpConfigTemplateContent string

//go:embed templates/11-exec_config.conf
var customContainerdServiceOverride string

var ntpConfigTemplate *template.Template
var decoder runtime.Decoder

type actuator struct {
	client          client.Client
	extensionConfig Config
}

// Config contains configuration for the extension service.
type Config struct {
	// Embed the entire Extension config here for direct access in the controller.
	*configv1alpha1.ExtensionConfig
}

// NewActuator creates a new Actuator that updates the status of the handled OperatingSystemConfigs.
func NewActuator(mgr manager.Manager, extensionConfig Config) operatingsystemconfig.Actuator {
	return &actuator{
		client:          mgr.GetClient(),
		extensionConfig: extensionConfig,
	}
}

func init() {
	var err error
	scheme := runtime.NewScheme()
	runtimeutils.Must(configv1alpha1.AddToScheme(scheme))
	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	ntpConfigTemplate, err = template.New("ntp-config").Funcs(sprig.TxtFuncMap()).Parse(ntpConfigTemplateContent)
	if err != nil {
		panic(fmt.Errorf("failed to parse NTP config template: %w", err))
	}
}

func (a *actuator) GetAndMergeProviderConfiguration(osc *extensionsv1alpha1.OperatingSystemConfig) (*configv1alpha1.ExtensionConfig, error) {
	shootExtensionConfig := &configv1alpha1.ExtensionConfig{}
	if _, _, err := decoder.Decode(osc.Spec.ProviderConfig.Raw, nil, shootExtensionConfig); err != nil {
		return nil, fmt.Errorf("failed to decode provider config: %+v", err)
	}

	config := a.extensionConfig.DeepCopy()
	if shootExtensionConfig.NTP != nil {
		config.NTP = shootExtensionConfig.NTP
	}

	return config, nil
}

func (a *actuator) Reconcile(ctx context.Context, _ logr.Logger, osc *extensionsv1alpha1.OperatingSystemConfig) ([]byte, []extensionsv1alpha1.Unit, []extensionsv1alpha1.File, *extensionsv1alpha1.InPlaceUpdatesStatus, error) {
	var config *configv1alpha1.ExtensionConfig
	var err error

	// Check if the shoot provider configuration is provided. If yes, merge it with the default configuration from the extension.
	if osc.Spec.ProviderConfig != nil {
		config, err = a.GetAndMergeProviderConfiguration(osc)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	} else {
		// If no shoot provider configuration is provided, use the default configuration from the extension.
		config = a.extensionConfig.ExtensionConfig
	}

	switch purpose := osc.Spec.Purpose; purpose {
	case extensionsv1alpha1.OperatingSystemConfigPurposeProvision:
		userData, err := a.handleProvisionOSC(ctx, osc)
		return []byte(userData), nil, nil, nil, err

	case extensionsv1alpha1.OperatingSystemConfigPurposeReconcile:
		extensionUnits, extensionFiles, err := a.handleReconcileOSC(config, osc)
		return nil, extensionUnits, extensionFiles, nil, err

	default:
		return nil, nil, nil, nil, fmt.Errorf("unknown purpose: %s", purpose)
	}
}

func (a *actuator) Delete(_ context.Context, _ logr.Logger, _ *extensionsv1alpha1.OperatingSystemConfig) error {
	return nil
}

func (a *actuator) Migrate(ctx context.Context, log logr.Logger, osc *extensionsv1alpha1.OperatingSystemConfig) error {
	return a.Delete(ctx, log, osc)
}

func (a *actuator) ForceDelete(ctx context.Context, log logr.Logger, osc *extensionsv1alpha1.OperatingSystemConfig) error {
	return a.Delete(ctx, log, osc)
}

func (a *actuator) Restore(ctx context.Context, logger logr.Logger, osc *extensionsv1alpha1.OperatingSystemConfig) ([]byte, []extensionsv1alpha1.Unit, []extensionsv1alpha1.File, *extensionsv1alpha1.InPlaceUpdatesStatus, error) {
	return a.Reconcile(ctx, logger, osc)
}

//go:embed templates/containerd/run-command.sh.tpl
var containerdTemplateContent string

func (a *actuator) handleProvisionOSC(ctx context.Context, osc *extensionsv1alpha1.OperatingSystemConfig) (string, error) {
	cfg := igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: "3.3.0",
		},
	}

	// Write the containerd setup script. It initialises the containerd config and
	// patches it for cgroups v2 if necessary. A systemd oneshot unit runs it once
	// before containerd starts.
	cfg.Storage.Files = append(cfg.Storage.Files, newIgnitionFile(
		"/opt/bin/containerd-setup.sh",
		containerdTemplateContent,
		ptr.To(0o755),
	))

	// Convert files from the OSC spec.
	for _, file := range osc.Spec.Files {
		source, err := fileContentToDataURI(ctx, a.client, osc.Namespace, file)
		if err != nil {
			return "", fmt.Errorf("failed to get content for file %s: %w", file.Path, err)
		}
		ignFile := igntypes.File{
			Node: igntypes.Node{
				Path: file.Path,
			},
			FileEmbedded1: igntypes.FileEmbedded1{
				Contents: igntypes.Resource{
					Source: ptr.To(source),
				},
			},
		}
		if file.Permissions != nil {
			mode := int(*file.Permissions)
			ignFile.Mode = &mode
		}
		cfg.Storage.Files = append(cfg.Storage.Files, ignFile)
	}

	// Systemd oneshot unit that runs the containerd setup script exactly once before
	// containerd starts. The marker file prevents re-execution on subsequent boots.
	containerdSetupUnitContent := `[Unit]
Description=Setup containerd configuration
Before=containerd.service
ConditionPathExists=!/var/lib/osc/containerd-setup-done

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/opt/bin/containerd-setup.sh
ExecStartPost=/bin/sh -c 'mkdir -p /var/lib/osc && touch /var/lib/osc/containerd-setup-done'

[Install]
WantedBy=multi-user.target`
	cfg.Systemd.Units = append(cfg.Systemd.Units, igntypes.Unit{
		Name:     "containerd-setup.service",
		Contents: ptr.To(containerdSetupUnitContent),
		Enabled:  ptr.To(true),
	})

	// Enable containerd with the custom ExecStart drop-in.
	cfg.Systemd.Units = append(cfg.Systemd.Units, igntypes.Unit{
		Name:    "containerd.service",
		Enabled: ptr.To(true),
		Dropins: []igntypes.Dropin{{
			Name:     "11-exec_config.conf",
			Contents: &customContainerdServiceOverride,
		}},
	})

	// Enable docker (present on Flatcar alongside containerd).
	cfg.Systemd.Units = append(cfg.Systemd.Units, igntypes.Unit{
		Name:    "docker.service",
		Enabled: ptr.To(true),
	})

	// Convert units from the OSC spec.
	for _, unit := range osc.Spec.Units {
		ignUnit := igntypes.Unit{
			Name:    unit.Name,
			Enabled: unit.Enable,
		}
		if unit.Content != nil {
			ignUnit.Contents = unit.Content
		}
		for _, dropin := range unit.DropIns {
			content := dropin.Content
			ignUnit.Dropins = append(ignUnit.Dropins, igntypes.Dropin{
				Name:     dropin.Name,
				Contents: &content,
			})
		}
		cfg.Systemd.Units = append(cfg.Systemd.Units, ignUnit)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ignition config: %w", err)
	}

	// Validate the generated config against the Ignition v3.3 schema.
	if _, rpt, err := ignv3_3.Parse(data); err != nil {
		return "", fmt.Errorf("ignition config validation failed: %w (report: %s)", err, rpt)
	}

	return string(data), nil
}

// newIgnitionFile creates an igntypes.File with the given content encoded as a base64 data URI.
func newIgnitionFile(path, content string, mode *int) igntypes.File {
	return igntypes.File{
		Node: igntypes.Node{
			Path: path,
		},
		FileEmbedded1: igntypes.FileEmbedded1{
			Contents: igntypes.Resource{
				Source: ptr.To("data:;base64," + base64.StdEncoding.EncodeToString([]byte(content))),
			},
			Mode: mode,
		},
	}
}

// fileContentToDataURI resolves an OSC file's content (inline or from a k8s Secret) and
// returns it as a data URI suitable for an Ignition storage file source.
//
// For plain-encoded inline content (encoding: "") we use a non-base64 data URI
// (data:,<url-encoded>) so that the machine-controller-manager can find and replace
// placeholder strings such as <<BOOTSTRAP_TOKEN>> and <<MACHINE_NAME>>.
// The MCM explicitly looks for url.QueryEscape(placeholder) when processing Ignition
// user-data. Using base64 would hide these placeholders from the MCM, causing the node
// to receive the literal placeholder string as the bootstrap token.
//
// For base64-encoded inline content (encoding: "b64") and for Secret references we use
// the standard base64 data URI (data:;base64,<b64>) because the content does not
// contain MCM placeholders.
func fileContentToDataURI(ctx context.Context, cl client.Client, namespace string, file extensionsv1alpha1.File) (string, error) {
	if file.Content.Inline != nil {
		if file.Content.Inline.Encoding == string(extensionsv1alpha1.B64FileCodecID) {
			// Data is already base64-encoded; embed it directly in a base64 data URI.
			return "data:;base64," + file.Content.Inline.Data, nil
		}
		// Plain text: use a percent-encoded data URI so MCM placeholder strings
		// (<<BOOTSTRAP_TOKEN>>, <<MACHINE_NAME>>) remain visible in the Ignition JSON
		// and can be substituted by the machine-controller-manager before the VM boots.
		return "data:," + url.QueryEscape(file.Content.Inline.Data), nil
	}
	if file.Content.SecretRef != nil {
		secret := &corev1.Secret{}
		if err := cl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: file.Content.SecretRef.Name}, secret); err != nil {
			return "", fmt.Errorf("failed to get secret %q: %w", file.Content.SecretRef.Name, err)
		}
		data, ok := secret.Data[file.Content.SecretRef.DataKey]
		if !ok {
			return "", fmt.Errorf("key %q not found in secret %q", file.Content.SecretRef.DataKey, file.Content.SecretRef.Name)
		}
		return "data:;base64," + base64.StdEncoding.EncodeToString(data), nil
	}
	return "", fmt.Errorf("file %q has neither inline nor secret content", file.Path)
}

func (a *actuator) generateNTPConfig(config *configv1alpha1.ExtensionConfig) (string, error) {
	templateData := config.NTP.NTPD
	var templateOutput strings.Builder

	err := ntpConfigTemplate.Execute(&templateOutput, templateData)
	if err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return templateOutput.String(), nil
}

func (a *actuator) handleReconcileOSC(config *configv1alpha1.ExtensionConfig, _ *extensionsv1alpha1.OperatingSystemConfig) ([]extensionsv1alpha1.Unit, []extensionsv1alpha1.File, error) {
	var (
		extensionUnits []extensionsv1alpha1.Unit
		extensionFiles []extensionsv1alpha1.File
		err            error
	)

	// disable automatic updates
	extensionUnits = append(extensionUnits,
		extensionsv1alpha1.Unit{Name: "update-engine.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
		extensionsv1alpha1.Unit{Name: "locksmithd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
	)

	if ptr.Deref(config.NTP.Enabled, true) {
		if extensionUnits, extensionFiles, err = a.configureNTPDaemon(config, extensionUnits, extensionFiles); err != nil {
			return nil, nil, fmt.Errorf("error configuring NTP Daemon: %v", err)
		}
	}

	// blacklist sctp kernel module
	extensionFiles = append(extensionFiles, extensionsv1alpha1.File{
		Path:        filepath.Join("/", "etc", "modprobe.d", "sctp.conf"),
		Content:     extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: "install sctp /bin/true"}},
		Permissions: ptr.To[uint32](0644),
	})

	// add scripts and dropins for kubelet cgroup driver configuration
	filePathKubeletCGroupDriverScript := filepath.Join("/", "opt", "bin", "kubelet_cgroup_driver.sh")
	extensionFiles = append(extensionFiles, extensionsv1alpha1.File{
		Path:        filePathKubeletCGroupDriverScript,
		Content:     extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: cgroupsv2TemplateContent}},
		Permissions: ptr.To[uint32](0755),
	})
	extensionUnits = append(extensionUnits, extensionsv1alpha1.Unit{
		Name: "kubelet.service",
		DropIns: []extensionsv1alpha1.DropIn{{
			Name: "10-configure-cgroup-driver.conf",
			Content: `[Service]
ExecStartPre=` + filePathKubeletCGroupDriverScript + `
`,
		}},
		FilePaths: []string{filePathKubeletCGroupDriverScript},
	})
	extensionUnits = append(extensionUnits, extensionsv1alpha1.Unit{
		Name: "containerd.service",
		DropIns: []extensionsv1alpha1.DropIn{
			{
				Name:    "11-exec_config.conf",
				Content: customContainerdServiceOverride,
			},
		},
	})

	return extensionUnits, extensionFiles, nil
}

// configureNTPDaemon configures the VM either with systemd-timesyncd or ntpd as the time syncing client
func (a *actuator) configureNTPDaemon(config *configv1alpha1.ExtensionConfig, extensionUnits []extensionsv1alpha1.Unit, extensionFiles []extensionsv1alpha1.File) ([]extensionsv1alpha1.Unit, []extensionsv1alpha1.File, error) {
	switch config.NTP.Daemon {
	case configv1alpha1.SystemdTimesyncd:
		extensionUnits = append(extensionUnits,
			extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)},
			extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
		)
	case configv1alpha1.NTPD:
		extensionUnits = append(extensionUnits,
			extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
			extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true), FilePaths: []string{filepath.Join(string(filepath.Separator), "etc", "ntp.conf")}},
		)
		templateData, err := a.generateNTPConfig(config)
		if err != nil {
			return nil, nil, fmt.Errorf("error generating NTP config: %v", err)
		}
		extensionFiles = append(extensionFiles, extensionsv1alpha1.File{
			Path:        filepath.Join(string(filepath.Separator), "etc", "ntp.conf"),
			Content:     extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: templateData}},
			Permissions: ptr.To[uint32](0644),
		})
	default:
		return nil, nil, fmt.Errorf("unsupported NTP daemon: %s", config.NTP.Daemon)
	}

	return extensionUnits, extensionFiles, nil
}
