// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"bytes"
	"context"
	"path/filepath"

	"github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/test"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	runtimeutils "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	configv1alpha1 "github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1"
	. "github.com/gardener/gardener-extension-os-coreos/pkg/controller/operatingsystemconfig"
)

var _ = Describe("Actuator", func() {
	var (
		ctx        = context.TODO()
		log        = logr.Discard()
		fakeClient client.Client
		mgr        manager.Manager

		osc                   *extensionsv1alpha1.OperatingSystemConfig
		actuator              operatingsystemconfig.Actuator
		globalExtensionConfig *configv1alpha1.ExtensionConfig
		scheme                = runtime.NewScheme()
		encoder               runtime.Encoder
	)

	BeforeEach(func() {
		fakeClient = fakeclient.NewClientBuilder().Build()
		runtimeutils.Must(configv1alpha1.AddToScheme(scheme))
		encoder = serializer.NewCodecFactory(scheme).EncoderForVersion(&json.Serializer{}, configv1alpha1.SchemeGroupVersion)
		mgr = test.FakeManager{Client: fakeClient}
		extensionConfig := Config{
			ExtensionConfig: &configv1alpha1.ExtensionConfig{
				NTP: &configv1alpha1.NTPConfig{
					Enabled: ptr.To(true),
					Daemon:  configv1alpha1.SystemdTimesyncd,
				},
			},
		}
		globalExtensionConfig = extensionConfig.ExtensionConfig
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
if [ -f "/var/lib/osc/provision-osc-applied" ]; then
  echo "Provision OSC already applied, exiting..."
  exit 0
fi

if [ ! -s /etc/containerd/config.toml ]; then
  mkdir -p /etc/containerd/
  containerd config default > /etc/containerd/config.toml
  chmod 0644 /etc/containerd/config.toml
fi

mkdir -p /etc/systemd/system/containerd.service.d
cat <<EOF > /etc/systemd/system/containerd.service.d/11-exec_config.conf
[Service]
ExecStart=
# try to use containerd provided via torcx, but also falls back to /usr/bin/containerd provided via systemd-sysext
# TODO: Remove torxc once flatcar LTS support has run out.
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


mkdir -p /var/lib/osc
touch /var/lib/osc/provision-osc-applied
`

		Describe("#Reconcile", func() {
			It("should not return an error", func() {
				userData, extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(userData)).To(Equal(expectedUserData))
				Expect(extensionUnits).To(BeEmpty())
				Expect(extensionFiles).To(BeEmpty())
				Expect(inplaceUpdateStatus).To(BeNil())
			})
		})
	})

	When("purpose is 'reconcile'", func() {
		BeforeEach(func() {
			osc.Spec.Purpose = extensionsv1alpha1.OperatingSystemConfigPurposeReconcile
		})

		Describe("#Reconcile", func() {
			It("should override global defaults on a shoot by shoot basis and enable ntpd for that shoot", func() {
				// Shoot specific override via providerConfig
				providerConfigData := configv1alpha1.ExtensionConfig{
					NTP: &configv1alpha1.NTPConfig{
						Daemon:  configv1alpha1.NTPD,
						Enabled: ptr.To(true),
						NTPD: &configv1alpha1.NTPDConfig{
							Servers: []string{"foo.bar", "bar.foo"},
						},
					},
				}
				providerConfigBuffer := new(bytes.Buffer)
				err := encoder.Encode(&providerConfigData, providerConfigBuffer)
				osc.Spec.ProviderConfig = &runtime.RawExtension{Raw: providerConfigBuffer.Bytes()}
				defer DeferCleanup(func() {
					osc.Spec.ProviderConfig = nil
				})
				Expect(err).NotTo(HaveOccurred())
				userData, extensionUnits, _, _, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())
				Expect(userData).To(BeEmpty())
				Expect(providerConfigData).To(Not(Equal(*globalExtensionConfig)))
				Expect(extensionUnits).To(ContainElement(extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)}))
			})
			It("should not enable any timesync service when Daemon is None", func() {
				extensionConfig := Config{
					ExtensionConfig: &configv1alpha1.ExtensionConfig{
						NTP: &configv1alpha1.NTPConfig{
							Enabled: ptr.To(false),
						},
					},
				}
				actuator = NewActuator(mgr, extensionConfig)
				userData, extensionUnits, _, _, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())
				Expect(userData).To(BeEmpty())
				Expect(extensionUnits).To(Not(ContainElement(extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)})))
				Expect(extensionUnits).To(Not(ContainElement(extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)})))
				Expect(extensionUnits).To(Not(ContainElement(extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)})))
				Expect(extensionUnits).To(Not(ContainElement(extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)})))
			})
			It("should enable ntpd service", func() {
				extensionConfig := Config{
					ExtensionConfig: &configv1alpha1.ExtensionConfig{
						NTP: &configv1alpha1.NTPConfig{
							Enabled: ptr.To(true),
							Daemon:  configv1alpha1.NTPD,
							NTPD: &configv1alpha1.NTPDConfig{
								Servers: []string{"foo.bar", "bar.foo"},
							},
						},
					},
				}
				actuator = NewActuator(mgr, extensionConfig)
				userData, extensionUnits, extensionFiles, _, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())
				Expect(userData).To(BeEmpty())
				Expect(extensionUnits).To(ContainElement(extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true), FilePaths: []string{filepath.Join(string(filepath.Separator), "etc", "ntp.conf")}}))
				Expect(extensionFiles).To(ContainElement(extensionsv1alpha1.File{
					Path:        filepath.Join(string(filepath.Separator), "etc", "ntp.conf"),
					Permissions: ptr.To[uint32](0644),
					Content: extensionsv1alpha1.FileContent{
						Inline: &extensionsv1alpha1.FileContentInline{
							Data: `
server foo.bar iburst
server bar.foo iburst

driftfile /var/lib/ntp/ntp.drift
restrict default nomodify nopeer noquery notrap limited kod
restrict 127.0.0.1
restrict [::1]`,
						},
					},
				}))
			})
			It("should not return an error", func() {
				userData, extensionUnits, extensionFiles, _, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())

				Expect(userData).To(BeEmpty())
				Expect(extensionUnits).To(ConsistOf(
					extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)},
					extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)},
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
					extensionsv1alpha1.Unit{
						Name: "containerd.service",
						DropIns: []extensionsv1alpha1.DropIn{
							{
								Name: "11-exec_config.conf",
								Content: `[Service]
ExecStart=
# try to use containerd provided via torcx, but also falls back to /usr/bin/containerd provided via systemd-sysext
# TODO: Remove torxc once flatcar LTS support has run out.
ExecStart=/bin/bash -c 'PATH="/run/torcx/unpack/docker/bin:$PATH" containerd --config /etc/containerd/config.toml'`,
							},
						},
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
