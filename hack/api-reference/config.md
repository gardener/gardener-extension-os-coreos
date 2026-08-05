<p>Packages:</p>
<ul>
<li>
<a href="#config.coreos.os.extensions.gardener.cloud%2fv1alpha1">config.coreos.os.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="config.coreos.os.extensions.gardener.cloud/v1alpha1">config.coreos.os.extensions.gardener.cloud/v1alpha1</h2>
<p>

</p>

<h3 id="dhcpconfig">DHCPConfig
</h3>


<p>
(<em>Appears on:</em><a href="#interfaceconfig">InterfaceConfig</a>)
</p>

<p>

</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>enabled</code></br>
<em>
<a href="#dhcpenabled">DHCPEnabled</a>
</em>
</td>
<td>
<p>Enabled defines whether to enable DHCP<br />See https://www.freedesktop.org/software/systemd/man/latest/systemd.network.html#DHCP=</p>
</td>
</tr>
<tr>
<td>
<code>ipv4</code></br>
<em>
<a href="#dhcpipv4config">DHCPIPv4Config</a>
</em>
</td>
<td>
<p>IPv4 contains IPv4 DHCP options</p>
</td>
</tr>
<tr>
<td>
<code>ipv6</code></br>
<em>
<a href="#dhcpipv6config">DHCPIPv6Config</a>
</em>
</td>
<td>
<p>IPv4 contains IPv6 DHCP options</p>
</td>
</tr>

</tbody>
</table>


<h3 id="dhcpenabled">DHCPEnabled
</h3>
<p><em>Underlying type: string</em></p>


<p>
(<em>Appears on:</em><a href="#dhcpconfig">DHCPConfig</a>)
</p>

<p>

</p>


<h3 id="dhcpipv4config">DHCPIPv4Config
</h3>


<p>
(<em>Appears on:</em><a href="#dhcpconfig">DHCPConfig</a>)
</p>

<p>

</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>useGateway</code></br>
<em>
boolean
</em>
</td>
<td>
<p>UseGateway when set to true, and the DHCP server provides a Router option, the default gateway based on the router address will be configured<br />See https://www.freedesktop.org/software/systemd/man/latest/systemd.network.html#UseGateway= for more information.</p>
</td>
</tr>
<tr>
<td>
<code>useRoutes</code></br>
<em>
boolean
</em>
</td>
<td>
<p>UseRoutes defines whether static routes from DHCP will be added to the routing table<br />See https://www.freedesktop.org/software/systemd/man/latest/systemd.network.html#UseRoutes= for more information.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="dhcpipv6config">DHCPIPv6Config
</h3>


<p>
(<em>Appears on:</em><a href="#dhcpconfig">DHCPConfig</a>)
</p>

<p>

</p>


<h3 id="daemon">Daemon
</h3>
<p><em>Underlying type: string</em></p>


<p>
(<em>Appears on:</em><a href="#ntpconfig">NTPConfig</a>)
</p>

<p>

</p>


<h3 id="extensionconfig">ExtensionConfig
</h3>


<p>
ExtensionConfig is the configuration for the os-coreos extension.
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>enableDocker</code></br>
<em>
boolean
</em>
</td>
<td>
<em>(Optional)</em>
<p>EnableDocker specifies if docker should be available on the nodes.<br />Defaults to false, as docker is only need for special use-cases.</p>
</td>
</tr>
<tr>
<td>
<code>ntp</code></br>
<em>
<a href="#ntpconfig">NTPConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NTP to configure either systemd-timesyncd or ntpd</p>
</td>
</tr>
<tr>
<td>
<code>networkd</code></br>
<em>
<a href="#networkdconfig">NetworkdConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Networkd to configure systemd-networkd via .network files</p>
</td>
</tr>

</tbody>
</table>


<h3 id="interfaceconfig">InterfaceConfig
</h3>


<p>
(<em>Appears on:</em><a href="#networkdconfig">NetworkdConfig</a>)
</p>

<p>

</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the interface to configure. Supports glob patterns.<br />See https://www.freedesktop.org/software/systemd/man/latest/systemd.network.html#Name= for more information.</p>
</td>
</tr>
<tr>
<td>
<code>dhcp</code></br>
<em>
<a href="#dhcpconfig">DHCPConfig</a>
</em>
</td>
<td>
<p>DHCP contains DHCP configuration for the interface</p>
</td>
</tr>

</tbody>
</table>


<h3 id="ntpconfig">NTPConfig
</h3>


<p>
(<em>Appears on:</em><a href="#extensionconfig">ExtensionConfig</a>)
</p>

<p>
NTPConfig General NTP Config for either systemd-timesyncd or ntpd
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>enabled</code></br>
<em>
boolean
</em>
</td>
<td>
<p>Enabled Optionally disable or enable the extension to configure a timesync service for the machine</p>
</td>
</tr>
<tr>
<td>
<code>daemon</code></br>
<em>
<a href="#daemon">Daemon</a>
</em>
</td>
<td>
<p>Daemon One of either systemd-timesyncd or ntp</p>
</td>
</tr>
<tr>
<td>
<code>ntpd</code></br>
<em>
<a href="#ntpdconfig">NTPDConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NTPD to configure the ntpd client</p>
</td>
</tr>

</tbody>
</table>


<h3 id="ntpdconfig">NTPDConfig
</h3>


<p>
(<em>Appears on:</em><a href="#ntpconfig">NTPConfig</a>)
</p>

<p>
NTPDConfig is the struct used in the ntp-config.conf.tpl template file
</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>servers</code></br>
<em>
string array
</em>
</td>
<td>
<p>Servers List of ntp servers</p>
</td>
</tr>
<tr>
<td>
<code>interfaces</code></br>
<em>
string array
</em>
</td>
<td>
<p>Interfaces for ntpd to bind to. Can be more than one.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="networkdconfig">NetworkdConfig
</h3>


<p>
(<em>Appears on:</em><a href="#extensionconfig">ExtensionConfig</a>)
</p>

<p>

</p>

<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>

<tr>
<td>
<code>interfaces</code></br>
<em>
<a href="#interfaceconfig">InterfaceConfig</a> array
</em>
</td>
<td>
<p>Interfaces holds network configuration based for interfaces</p>
</td>
</tr>

</tbody>
</table>


