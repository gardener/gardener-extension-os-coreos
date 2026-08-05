package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Daemon string

const (
	SystemdTimesyncd Daemon = "systemd-timesyncd"
	NTPD             Daemon = "ntpd"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExtensionConfig is the configuration for the os-coreos extension.
type ExtensionConfig struct {
	metav1.TypeMeta `json:",inline"`

	// EnableDocker specifies if docker should be available on the nodes.
	// Defaults to false, as docker is only need for special use-cases.
	// +optional
	EnableDocker *bool `json:"enableDocker,omitempty"`
	// NTP to configure either systemd-timesyncd or ntpd
	// +optional
	NTP *NTPConfig `json:"ntp,omitempty"`

	// Networkd to configure systemd-networkd via .network files
	// +optional
	Networkd *NetworkdConfig `json:"networkd,omitempty"`
}

// NTPConfig General NTP Config for either systemd-timesyncd or ntpd
type NTPConfig struct {
	// Enabled Optionally disable or enable the extension to configure a timesync service for the machine
	Enabled *bool `json:"enabled,omitempty"`
	// Daemon One of either systemd-timesyncd or ntp
	Daemon Daemon `json:"daemon"`
	// NTPD to configure the ntpd client
	// +optional
	NTPD *NTPDConfig `json:"ntpd,omitempty"`
}

// NTPDConfig is the struct used in the ntp-config.conf.tpl template file
type NTPDConfig struct {
	// Servers List of ntp servers
	Servers []string `json:"servers"`
	// Interfaces for ntpd to bind to. Can be more than one.
	Interfaces []string `json:"interfaces,omitempty"`
}

type NetworkdConfig struct {
	// Interfaces holds network configuration based for interfaces
	Interfaces []InterfaceConfig `json:"interfaces,omitempty"`
}

type InterfaceConfig struct {
	// Name is the name of the interface to configure. Supports glob patterns.
	// See https://www.freedesktop.org/software/systemd/man/latest/systemd.network.html#Name= for more information.
	Name string `json:"name"`

	// DHCP contains DHCP configuration for the interface
	DHCP *DHCPConfig `json:"dhcp,omitempty"`
}

type DHCPEnabled string

const (
	DHCPEnabledYes  DHCPEnabled = "yes"
	DHCPEnabledNo   DHCPEnabled = "no"
	DHCPEnabledIPv4 DHCPEnabled = "ipv4"
	DHCPEnabledIPv6 DHCPEnabled = "ipv6"
)

type DHCPConfig struct {
	// Enabled defines whether to enable DHCP
	// See https://www.freedesktop.org/software/systemd/man/latest/systemd.network.html#DHCP=
	Enabled *DHCPEnabled `json:"enabled,omitempty"`
	// IPv4 contains IPv4 DHCP options
	IPv4 *DHCPIPv4Config `json:"ipv4,omitempty"`
	// IPv4 contains IPv6 DHCP options
	IPv6 *DHCPIPv6Config `json:"ipv6,omitempty"`
}

type DHCPIPv4Config struct {
	// UseGateway when set to true, and the DHCP server provides a Router option, the default gateway based on the router address will be configured
	// See https://www.freedesktop.org/software/systemd/man/latest/systemd.network.html#UseGateway= for more information.
	UseGateway *bool `json:"useGateway"`
	// UseRoutes defines whether static routes from DHCP will be added to the routing table
	// See https://www.freedesktop.org/software/systemd/man/latest/systemd.network.html#UseRoutes= for more information.
	UseRoutes *bool `json:"useRoutes"`
}

type DHCPIPv6Config struct {
}
