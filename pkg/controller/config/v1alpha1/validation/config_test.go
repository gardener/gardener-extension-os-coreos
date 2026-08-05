package validation

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"

	configv1alpha1 "github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1"
)

var _ = Describe("ExtensionConfig validation", func() {
	var (
		config *configv1alpha1.ExtensionConfig
	)

	BeforeEach(func() {
		config = &configv1alpha1.ExtensionConfig{
			NTP: &configv1alpha1.NTPConfig{
				Daemon: configv1alpha1.SystemdTimesyncd,
			},
		}
	})

	It("should allow valid config", func() {
		Expect(ValidateExtensionConfig(config)).To(BeEmpty())
	})

	It("should fail with incorrect daemon name", func() {
		config.NTP.Daemon = "foo"
		errs := ValidateExtensionConfig(config)
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].Type).To(Equal(field.ErrorTypeNotSupported))
		Expect(errs[0].Field).To(Equal("daemon"))
	})

	It("should fail with daemon systemd-timesyncd and ntpd config set", func() {
		config.NTP.Daemon = configv1alpha1.SystemdTimesyncd
		config.NTP.NTPD = &configv1alpha1.NTPDConfig{Servers: []string{"foo.bar"}}
		errs := ValidateExtensionConfig(config)
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
		Expect(errs[0].Field).To(Equal("ntpd"))
	})
})

var _ = Describe("#validateNetworkdConfig", func() {
	It("should succeed for a valid config", func() {
		config := &configv1alpha1.NetworkdConfig{
			Interfaces: []configv1alpha1.InterfaceConfig{
				{
					Name: "eth0",
				},
			},
		}
		Expect(validateNetworkdConfig(config, field.NewPath("networkd")).ToAggregate()).To(Succeed())
	})

	It("should fail if interface name is missing", func() {
		config := &configv1alpha1.NetworkdConfig{
			Interfaces: []configv1alpha1.InterfaceConfig{
				{},
			},
		}
		errs := validateNetworkdConfig(config, field.NewPath("networkd"))
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
		Expect(errs[0].Field).To(Equal("networkd.interface[0].name"))
	})

	It("should fail if interface name is specified twice", func() {
		config := &configv1alpha1.NetworkdConfig{
			Interfaces: []configv1alpha1.InterfaceConfig{
				{
					Name: "eth0",
				},
				{
					Name: "eth0",
				},
			},
		}
		errs := validateNetworkdConfig(config, field.NewPath("networkd"))
		Expect(errs).To(HaveLen(1))
		Expect(errs[0].Type).To(Equal(field.ErrorTypeDuplicate))
		Expect(errs[0].Field).To(Equal("networkd.interface[1].name"))
	})
	It("should fail if dhcp options are set for other ip version", func() {
		config := &configv1alpha1.NetworkdConfig{
			Interfaces: []configv1alpha1.InterfaceConfig{
				{
					Name: "ipv6",
					DHCP: &configv1alpha1.DHCPConfig{
						Enabled: new(configv1alpha1.DHCPEnabledIPv6),
						IPv4:    &configv1alpha1.DHCPIPv4Config{},
					},
				},
				{
					Name: "ipv4",
					DHCP: &configv1alpha1.DHCPConfig{
						Enabled: new(configv1alpha1.DHCPEnabledIPv4),
						IPv6:    &configv1alpha1.DHCPIPv6Config{},
					},
				},
			},
		}
		errs := validateNetworkdConfig(config, field.NewPath("networkd"))
		Expect(errs).To(HaveLen(2))
		Expect(errs[0].Type).To(Equal(field.ErrorTypeInvalid))
		Expect(errs[0].Field).To(Equal("networkd.interface[0].dhcp.ipv4"))
		Expect(errs[1].Type).To(Equal(field.ErrorTypeInvalid))
		Expect(errs[1].Field).To(Equal("networkd.interface[1].dhcp.ipv6"))
	})
})
