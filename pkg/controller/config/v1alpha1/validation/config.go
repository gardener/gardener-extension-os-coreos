package validation

import (
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

	return allErrs
}

func validateNTPDConfig(config *configv1alpha1.NTPDConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(config.Servers) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("ntpServers"), "a list of NTP servers is required"))
	}
	return allErrs
}
