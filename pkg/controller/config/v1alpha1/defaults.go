package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func SetDefaults_ExtensionConfig(obj *ExtensionConfig) {
	if obj.NTP == nil {
		obj.NTP = &NTPConfig{}
	}
}

func SetDefaults_NTPConfig(obj *NTPConfig) {
	if obj.Daemon == "" {
		obj.Daemon = SystemdTimesyncd
	}
}
