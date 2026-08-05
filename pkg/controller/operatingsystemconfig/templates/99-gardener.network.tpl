[Match]
Name={{ .Name }}

[Network]
DHCP={{ .DHCP }}

{{- with .DHCPV4 }}
[DHCPv4]
UseGateway={{ .UseGateway }}
UseRoutes={{ .UseRoutes }}
# default from flatcars zz-default.network
RoutesToDNS=false
{{- end }}

{{- with .DHCPV6 }}
[DHCPv6]
{{- end }}

# defaults from flatcars zz-default.network
[DHCP]
UseMTU=true
UseDomains=true
