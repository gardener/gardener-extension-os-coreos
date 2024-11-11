package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExtensionConfig is the configuration for the os-coreos extension.
type ExtensionConfig struct {
	metav1.TypeMeta `json:",inline"`
	// UseNTP whether to toggle NTP or not
	UseNTP *bool `json:"useNTP"`
	// NTPConfig ntpConfig
	NTPConfig *NTPConfig `json:"ntpConfig"`
}

// NTPConfig is the struct used in the ntp-config template file
type NTPConfig struct {
	// NTPServers List of ntp servers
	NTPServers []string `json:"ntpServers"`
}
