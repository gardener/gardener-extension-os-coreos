// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package operatingsystemconfig_test

import (
	"bytes"
	"context"
	stdjson "encoding/json"
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

// ignitionTestConfig mirrors the Ignition v3 JSON structure for test assertions only.
type ignitionTestConfig struct {
	Ignition struct {
		Version string `json:"version"`
	} `json:"ignition"`
	Storage struct {
		Files []struct {
			Path     string `json:"path"`
			Contents struct {
				Source string `json:"source"`
			} `json:"contents"`
			Mode *int `json:"mode"`
		} `json:"files"`
	} `json:"storage"`
	Systemd struct {
		Units []struct {
			Name     string  `json:"name"`
			Contents *string `json:"contents"`
			Enabled  *bool   `json:"enabled"`
			Dropins  []struct {
				Name     string  `json:"name"`
				Contents *string `json:"contents"`
			} `json:"dropins"`
		} `json:"units"`
	} `json:"systemd"`
}

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
				Units:   []extensionsv1alpha1.Unit{{Name: "some-unit.service", Content: ptr.To("[Unit]\nDescription=Some Unit\n[Install]\nWantedBy=multi-user.target")}},
				Files:   []extensionsv1alpha1.File{{Path: "/some/file", Content: extensionsv1alpha1.FileContent{Inline: &extensionsv1alpha1.FileContentInline{Data: "bar"}}}},
			},
		}
	})

	When("purpose is 'provision'", func() {
		Describe("#Reconcile", func() {
			It("should return a valid Ignition v3 config JSON", func() {
				userData, extensionUnits, extensionFiles, inplaceUpdateStatus, err := actuator.Reconcile(ctx, log, osc)
				Expect(err).NotTo(HaveOccurred())
				Expect(extensionUnits).To(BeEmpty())
				Expect(extensionFiles).To(BeEmpty())
				Expect(inplaceUpdateStatus).To(BeNil())

				var ign ignitionTestConfig
				Expect(stdjson.Unmarshal(userData, &ign)).To(Succeed())

				By("having Ignition spec version 3.3.0")
				Expect(ign.Ignition.Version).To(Equal("3.3.0"))

				By("including the containerd setup script as a file")
				filePaths := make([]string, 0, len(ign.Storage.Files))
				for _, f := range ign.Storage.Files {
					filePaths = append(filePaths, f.Path)
				}
				Expect(filePaths).To(ContainElement("/opt/bin/containerd-setup.sh"))

				By("writing the containerd setup script as executable")
				for _, f := range ign.Storage.Files {
					if f.Path == "/opt/bin/containerd-setup.sh" {
						Expect(f.Mode).NotTo(BeNil())
						Expect(*f.Mode).To(Equal(0o755))
					}
				}

				By("including OSC files")
				Expect(filePaths).To(ContainElement("/some/file"))
				for _, f := range ign.Storage.Files {
					if f.Path == "/some/file" {
						// Plain-encoded content uses a percent-encoded data URI so MCM
						// placeholder strings remain visible for substitution.
						Expect(f.Contents.Source).To(Equal("data:,bar"))
					}
				}

				By("including a containerd-setup systemd unit that runs before containerd")
				unitNames := make([]string, 0, len(ign.Systemd.Units))
				for _, u := range ign.Systemd.Units {
					unitNames = append(unitNames, u.Name)
				}
				Expect(unitNames).To(ContainElements(
					"containerd-setup.service",
					"containerd.service",
					"docker.service",
					"some-unit.service",
				))

				By("enabling the containerd-setup unit")
				for _, u := range ign.Systemd.Units {
					if u.Name == "containerd-setup.service" {
						Expect(u.Enabled).To(Equal(ptr.To(true)))
						Expect(u.Contents).NotTo(BeNil())
						Expect(*u.Contents).To(ContainSubstring("Before=containerd.service"))
						Expect(*u.Contents).To(ContainSubstring("ExecStart=/opt/bin/containerd-setup.sh"))
					}
				}

				By("adding the containerd exec drop-in via the systemd unit")
				for _, u := range ign.Systemd.Units {
					if u.Name == "containerd.service" {
						Expect(u.Enabled).To(Equal(ptr.To(true)))
						Expect(u.Dropins).To(HaveLen(1))
						Expect(u.Dropins[0].Name).To(Equal("11-exec_config.conf"))
						Expect(u.Dropins[0].Contents).NotTo(BeNil())
					}
				}

				By("including OSC units with their content")
				for _, u := range ign.Systemd.Units {
					if u.Name == "some-unit.service" {
						Expect(u.Contents).NotTo(BeNil())
						Expect(*u.Contents).To(ContainSubstring("[Unit]"))
					}
				}
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
				Expect(extensionUnits).To(ContainElement(extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true), FilePaths: []string{"/etc/ntp.conf"}}))
			})
			It("should override global default with nothing", func() {
				providerConfigData := configv1alpha1.ExtensionConfig{
					NTP: &configv1alpha1.NTPConfig{
						Enabled: ptr.To(false),
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
				Expect(extensionUnits).To(Not(ContainElement(extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)})))
				Expect(extensionUnits).To(Not(ContainElement(extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStart), Enable: ptr.To(true)})))
				Expect(extensionUnits).To(Not(ContainElement(extensionsv1alpha1.Unit{Name: "systemd-timesyncd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)})))
				Expect(extensionUnits).To(Not(ContainElement(extensionsv1alpha1.Unit{Name: "ntpd.service", Command: ptr.To(extensionsv1alpha1.CommandStop), Enable: ptr.To(false)})))
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
								Servers:    []string{"foo.bar", "bar.foo"},
								Interfaces: []string{"dev1", "dev2"},
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
restrict [::1]

interface ignore wildcard
interface listen 127.0.0.1
interface listen dev1
interface listen dev2
`,
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
ExecStart=/usr/bin/containerd --config /etc/containerd/config.toml`,
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
