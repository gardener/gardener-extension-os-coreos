// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
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
	ntpConfigTemplate, err = template.New("ntp-config").Parse(ntpConfigTemplateContent)
	if err != nil {
		panic(fmt.Errorf("failed to parse NTP config template: %w", err))
	}
}

func (a *actuator) Reconcile(ctx context.Context, _ logr.Logger, osc *extensionsv1alpha1.OperatingSystemConfig) ([]byte, []extensionsv1alpha1.Unit, []extensionsv1alpha1.File, *extensionsv1alpha1.InPlaceUpdatesStatus, error) {
	switch purpose := osc.Spec.Purpose; purpose {
	case extensionsv1alpha1.OperatingSystemConfigPurposeProvision:
		userData, err := a.handleProvisionOSC(ctx, osc)
		return []byte(userData), nil, nil, nil, err

	case extensionsv1alpha1.OperatingSystemConfigPurposeReconcile:
		extensionUnits, extensionFiles, err := a.handleReconcileOSC(osc)
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
	writeFilesToDiskScript, err := operatingsystemconfig.FilesToDiskScript(ctx, a.client, osc.Namespace, osc.Spec.Files)
	if err != nil {
		return "", err
	}
	writeUnitsToDiskScript := operatingsystemconfig.UnitsToDiskScript(osc.Spec.Units)

	script := `#!/bin/bash
if [ ! -s /etc/containerd/config.toml ]; then
  mkdir -p /etc/containerd/
  containerd config default > /etc/containerd/config.toml
  chmod 0644 /etc/containerd/config.toml
fi

mkdir -p /etc/systemd/system/containerd.service.d
cat <<EOF > /etc/systemd/system/containerd.service.d/11-exec_config.conf
` + customContainerdServiceOverride + `
EOF
chmod 0644 /etc/systemd/system/containerd.service.d/11-exec_config.conf
` + writeFilesToDiskScript + `
` + writeUnitsToDiskScript + `
` + containerdTemplateContent + `
systemctl daemon-reload
systemctl enable containerd && systemctl restart containerd
systemctl enable docker && systemctl restart docker
`

	for _, unit := range osc.Spec.Units {
		script += fmt.Sprintf(`systemctl enable '%s' && systemctl restart --no-block '%s'
`, unit.Name, unit.Name)
	}

	// The provisioning script must run only once.
	script = operatingsystemconfig.WrapProvisionOSCIntoOneshotScript(script)

	return script, nil
}

func (a *actuator) generateNTPConfig() (string, error) {
	templateData := a.extensionConfig.NTP.NTPD
	var templateOutput strings.Builder

	err := ntpConfigTemplate.Execute(&templateOutput, templateData)
	if err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return templateOutput.String(), nil
}

func (a *actuator) handleReconcileOSC(_ *extensionsv1alpha1.OperatingSystemConfig) ([]extensionsv1alpha1.Unit, []extensionsv1alpha1.File, error) {
	var (
		extensionUnits []extensionsv1alpha1.Unit
		extensionFiles []extensionsv1alpha1.File
	)

	// disable automatic updates
	extensionUnits = append(extensionUnits,
		extensionsv1alpha1.Unit{Name: "update-engine.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
		extensionsv1alpha1.Unit{Name: "locksmithd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
	)

	var err error
	if extensionUnits, extensionFiles, err = a.configureNTPDaemon(extensionUnits, extensionFiles); err != nil {
		return nil, nil, fmt.Errorf("error configuring NTP Daemon: %v", err)
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
func (a *actuator) configureNTPDaemon(extensionUnits []extensionsv1alpha1.Unit, extensionFiles []extensionsv1alpha1.File) ([]extensionsv1alpha1.Unit, []extensionsv1alpha1.File, error) {
	switch a.extensionConfig.NTP.Daemon {
	case configv1alpha1.SystemdTimesyncd:
		extensionUnits = append(extensionUnits,
			extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)},
			extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
		)
	case configv1alpha1.NTPD:
		extensionUnits = append(extensionUnits,
			extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
			extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)},
		)
		templateData, err := a.generateNTPConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("error generating NTP config: %v", err)
		}
		extensionFiles = append(extensionFiles, extensionsv1alpha1.File{
			Path:        filepath.Join(string(filepath.Separator), "etc", "ntp.conf"),
			Content:     extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: templateData}},
			Permissions: ptr.To[uint32](0644),
		})
	default:
		return nil, nil, fmt.Errorf("unsupported NTP daemon: %s", a.extensionConfig.NTP.Daemon)
	}

	return extensionUnits, extensionFiles, nil
}
