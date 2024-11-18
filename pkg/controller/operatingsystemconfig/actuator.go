// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type actuator struct {
	client client.Client
}

// NewActuator creates a new Actuator that updates the status of the handled OperatingSystemConfigs.
func NewActuator(mgr manager.Manager) operatingsystemconfig.Actuator {
	return &actuator{
		client: mgr.GetClient(),
	}
}

func (a *actuator) Reconcile(ctx context.Context, _ logr.Logger, osc *extensionsv1alpha1.OperatingSystemConfig) ([]byte, []extensionsv1alpha1.Unit, []extensionsv1alpha1.File, error) {
	switch purpose := osc.Spec.Purpose; purpose {
	case extensionsv1alpha1.OperatingSystemConfigPurposeProvision:
		userData, err := a.handleProvisionOSC(ctx, osc)
		return []byte(userData), nil, nil, err

	case extensionsv1alpha1.OperatingSystemConfigPurposeReconcile:
		extensionUnits, extensionFiles, err := a.handleReconcileOSC(osc)
		return nil, extensionUnits, extensionFiles, err

	default:
		return nil, nil, nil, fmt.Errorf("unknown purpose: %s", purpose)
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

func (a *actuator) Restore(ctx context.Context, logger logr.Logger, osc *extensionsv1alpha1.OperatingSystemConfig) ([]byte, []extensionsv1alpha1.Unit, []extensionsv1alpha1.File, error) {
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
# TODO(MichaelEischer): remove this file once all flatcar versions that use torcx,
# that is before 3815.2.0, have run out of support
[Service]
ExecStart=
# try to use containerd provided via torcx, but also falls back to /usr/bin/containerd provided via systemd-sysext
ExecStart=/bin/bash -c 'PATH="/run/torcx/unpack/docker/bin:$PATH" containerd --config /etc/containerd/config.toml'
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

	return script, nil
}

//go:embed templates/configure-cgroupsv2.sh.tpl
var cgroupsv2TemplateContent string

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

	return extensionUnits, extensionFiles, nil
}
