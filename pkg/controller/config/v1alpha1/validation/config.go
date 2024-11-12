package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	configv1alpha1 "github.com/gardener/gardener-extension-os-coreos/pkg/controller/config/v1alpha1"
)

func ValidateExtensionConfig(config *configv1alpha1.ExtensionConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	var rootPath *field.Path

	if config.UseNTP != nil {
		// Make sure NTPConfig is set when UseNTP is true
		if *config.UseNTP && config.NTPConfig == nil {
			allErrs = append(allErrs, field.Required(rootPath.Child("ntpConfig"), "ntpConfig must be set when useNTP is true"))
		}

		if config.NTPConfig != nil {
			allErrs = append(allErrs, validateNTPConfig(config.NTPConfig, rootPath.Child("ntpConfig"))...)
		}
	}

	return allErrs
}

func validateNTPConfig(config *configv1alpha1.NTPConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(config.NTPServers) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("ntpServers"), "a list of NTP servers are required"))
	}
	return allErrs
}
