package validation

import (
	"fmt"

	"github.com/gobwas/glob"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	configv1alpha1 "github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1"
)

func ValidateExtensionConfig(config *configv1alpha1.ExtensionConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	var rootPath *field.Path

	validDaemonNames := sets.New(configv1alpha1.SystemdTimesyncd, configv1alpha1.NTPD)

	if config.NTP != nil {
		// Make sure daemon name is valid
		if !validDaemonNames.Has(config.NTP.Daemon) {
			allErrs = append(allErrs, field.NotSupported(rootPath.Child("daemon"), config.NTP.Daemon, validDaemonNames.UnsortedList()))
		}

		// Check if user configured systemd-timesyncd daemon with ntpd config
		if config.NTP.Daemon != configv1alpha1.NTPD && config.NTP.NTPD != nil {
			allErrs = append(allErrs, field.Forbidden(rootPath.Child("ntpd"), "NTP daemon not allowed in systemd config"))
		}

		if config.NTP.NTPD != nil {
			allErrs = append(allErrs, validateNTPDConfig(config.NTP.NTPD, rootPath.Child("ntpd"))...)
		}
	}

	if config.Networkd != nil {
		allErrs = append(allErrs, validateNetworkdConfig(config.Networkd, rootPath.Child("networkd"))...)
	}

	return allErrs
}

func validateNTPDConfig(config *configv1alpha1.NTPDConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(config.Servers) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("servers"), "a list of NTP servers is required"))
	}
	return allErrs
}

var allNetworkdDHCPEnabled = sets.New(
	configv1alpha1.DHCPEnabledYes,
	configv1alpha1.DHCPEnabledNo,
	configv1alpha1.DHCPEnabledIPv4,
	configv1alpha1.DHCPEnabledIPv6,
)

func validateNetworkdConfig(config *configv1alpha1.NetworkdConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	ifaceNameSet := sets.New[string]()
	for i, iface := range config.Interfaces {
		fldPath := fldPath.Child(fmt.Sprintf("interface[%d]", i))
		if iface.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("name"), "interface name is required"))
		}
		_, err := glob.Compile(iface.Name)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), iface.Name, err.Error()))
		}

		if ifaceNameSet.Has(iface.Name) {
			allErrs = append(allErrs, field.Duplicate(fldPath.Child("name"), iface.Name))
		}

		if iface.DHCP != nil {
			if iface.DHCP.Enabled != nil {
				if !allNetworkdDHCPEnabled.Has(*iface.DHCP.Enabled) {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("dhcp.enabled"), iface.DHCP.Enabled, fmt.Sprintf("unsupported value for enabled, supported values are %q", allNetworkdDHCPEnabled.UnsortedList())))
				}
				if iface.DHCP.IPv4 != nil && *iface.DHCP.Enabled == configv1alpha1.DHCPEnabledIPv6 {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("dhcp.ipv4"), *iface.DHCP.IPv4, "ipv4 DHCP config cannot be specified with only ipv6 DHCP"))
				}
				if iface.DHCP.IPv6 != nil && *iface.DHCP.Enabled == configv1alpha1.DHCPEnabledIPv4 {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("dhcp.ipv6"), *iface.DHCP.IPv6, "ipv6 DHCP config cannot be specified with only ipv4 DHCP"))
				}
			}
		}

		ifaceNameSet = ifaceNameSet.Insert(iface.Name)
	}
	return allErrs
}
