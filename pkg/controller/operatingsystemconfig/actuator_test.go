// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/test"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1"
	. "github.com/gardener/gardener-extension-os-coreos/pkg/controller/operatingsystemconfig"
)

var _ = Describe("Actuator", func() {
	var (
		ctx        = context.TODO()
		log        = logr.Discard()
		fakeClient client.Client
		mgr        manager.Manager

		osc      *extensionsv1alpha1.OperatingSystemConfig
		actuator operatingsystemconfig.Actuator
	)

	BeforeEach(func() {
		fakeClient = fakeclient.NewClientBuilder().Build()
		mgr = test.FakeManager{Client: fakeClient}
		extensionConfig := Config{ExtensionConfig: &v1alpha1.ExtensionConfig{UseNTP: ptr.To(false)}}
		actuator = NewActuator(mgr, extensionConfig)

		osc = &extensionsv1alpha1.OperatingSystemConfig{
			Spec: extensionsv1alpha1.OperatingSystemConfigSpec{
				Purpose: extensionsv1alpha1.OperatingSystemConfigPurposeProvision,
				Units:   []extensionsv1alpha1.Unit{{Name: "some-unit", Content: ptr.To("foo")}},
				Files:   []extensionsv1alpha1.File{{Path: "/some/file", Content: extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: "bar"}}}},
			},
		}
	})

	When("purpose is 'provision'", func() {
		expectedUserData := `#!/bin/bash
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

mkdir -p "/some"

cat << EOF | base64 -d > "/some/file"
YmFy
EOF


cat << EOF | base64 -d > "/etc/systemd/system/some-unit"
Zm9v
EOF
#!/bin/bash

CONTAINERD_CONFIG=/etc/containerd/config.toml

ALTERNATE_LOGROTATE_PATH="/usr/bin/logrotate"

# prefer containerd from torcx
# TODO(MichaelEischer): remove this special case once all flatcar versions that use torcx,
# that is before 3815.2.0, have run out of support
CONTAINERD="/usr/bin/containerd"
if [ -x /run/torcx/unpack/docker/bin/containerd ]; then
    CONTAINERD="/run/torcx/unpack/docker/bin/containerd"
fi

# initialize default containerd config if does not exist
if [ ! -s "$CONTAINERD_CONFIG" ]; then
    mkdir -p "$(dirname "$CONTAINERD_CONFIG")"
    ${CONTAINERD} config default > "$CONTAINERD_CONFIG"
    chmod 0644 "$CONTAINERD_CONFIG"
fi

# if cgroups v2 are used, patch containerd configuration to use systemd cgroup driver
if [[ -e /sys/fs/cgroup/cgroup.controllers ]]; then
    sed -i "s/SystemdCgroup *= *false/SystemdCgroup = true/" "$CONTAINERD_CONFIG"
fi

# TODO(MichaelEischer): remove this block once all flatcar versions that use torcx,
# that is before 3815.2.0, have run out of support
# provide kubelet with access to the containerd binaries in /run/torcx/unpack/docker/bin
if [ ! -s /etc/systemd/system/kubelet.service.d/environment.conf ]; then
    mkdir -p /etc/systemd/system/kubelet.service.d/
    cat <<EOF | tee /etc/systemd/system/kubelet.service.d/environment.conf
[Service]
Environment="PATH=/run/torcx/unpack/docker/bin:$PATH"
EOF
    chmod 0644 /etc/systemd/system/kubelet.service.d/environment.conf
    systemctl daemon-reload
fi

# some flatcar versions have logrotate at /usr/bin instead of /usr/sbin
if [ -f "$ALTERNATE_LOGROTATE_PATH" ]; then
    sed -i "s;/usr/sbin/logrotate;$ALTERNATE_LOGROTATE_PATH;" /etc/systemd/system/containerd-logrotate.service
    systemctl daemon-reload
fi

systemctl daemon-reload
systemctl enable containerd && systemctl restart containerd
systemctl enable docker && systemctl restart docker
systemctl enable 'some-unit' && systemctl restart --no-block 'some-unit'
`

		Describe("#Reconcile", func() {
			It("should not return an error", func() {
				userData, extensionUnits, extensionFiles, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(userData)).To(Equal(expectedUserData))
				Expect(extensionUnits).To(BeEmpty())
				Expect(extensionFiles).To(BeEmpty())
			})
		})
	})

	When("purpose is 'reconcile'", func() {
		BeforeEach(func() {
			osc.Spec.Purpose = extensionsv1alpha1.OperatingSystemConfigPurposeReconcile
		})

		Describe("#Reconcile", func() {
			It("should enable ntpd service", func() {
				extensionConfig := Config{
					ExtensionConfig: &v1alpha1.ExtensionConfig{UseNTP: ptr.To(true),
						NTPConfig: &v1alpha1.NTPConfig{NTPServers: []string{"foo.bar"}}},
				}
				actuator = NewActuator(mgr, extensionConfig)
				userData, extensionUnits, _, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())
				Expect(userData).To(BeEmpty())
				Expect(extensionUnits).To(ContainElement(extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)}))
			})
			It("should not return an error", func() {
				userData, extensionUnits, extensionFiles, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())

				Expect(userData).To(BeEmpty())
				Expect(extensionUnits).To(ConsistOf(
					extensionsv1alpha1.Unit{Name: "update-engine.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
					extensionsv1alpha1.Unit{Name: "locksmithd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
					extensionsv1alpha1.Unit{
						Name: "kubelet.service",
						DropIns: []extensionsv1alpha1.DropIn{{
							Name: "10-configure-cgroup-driver.conf",
							Content: `[Service]
ExecStartPre=/opt/bin/kubelet_cgroup_driver.sh
`,
						}},
						FilePaths: []string{"/opt/bin/kubelet_cgroup_driver.sh"},
					},
				))
				Expect(extensionFiles).To(ConsistOf(
					extensionsv1alpha1.File{
						Path:        "/etc/modprobe.d/sctp.conf",
						Permissions: ptr.To[uint32](0644),
						Content:     extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: "install sctp /bin/true"}},
					},
					extensionsv1alpha1.File{
						Path:        "/opt/bin/kubelet_cgroup_driver.sh",
						Permissions: ptr.To[uint32](0755),
						Content: extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: `#!/bin/bash

KUBELET_CONFIG=/var/lib/kubelet/config/kubelet

if [[ -e /sys/fs/cgroup/cgroup.controllers ]]; then
        echo "CGroups V2 are used!"
        echo "=> Patch kubelet to use systemd as cgroup driver"
        sed -i "s/cgroupDriver: cgroupfs/cgroupDriver: systemd/" "$KUBELET_CONFIG"
else
        echo "No CGroups V2 used by system"
fi
`}},
					},
				))
			})
		})
	})
})
