<p>Packages:</p>
<ul>
<li>
<a href="#config.coreos.os.extensions.gardener.cloud%2fv1alpha1">config.coreos.os.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="config.coreos.os.extensions.gardener.cloud/v1alpha1">config.coreos.os.extensions.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the API for configuring the os-coreos extension.</p>
</p>
Resource Types:
<ul></ul>
<h3 id="config.coreos.os.extensions.gardener.cloud/v1alpha1.Daemon">Daemon
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#config.coreos.os.extensions.gardener.cloud/v1alpha1.NTPConfig">NTPConfig</a>)
</p>
<p>
</p>
<h3 id="config.coreos.os.extensions.gardener.cloud/v1alpha1.ExtensionConfig">ExtensionConfig
</h3>
<p>
<p>ExtensionConfig is the configuration for the os-coreos extension.</p>
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
<code>ntp</code></br>
<em>
<a href="#config.coreos.os.extensions.gardener.cloud/v1alpha1.NTPConfig">
NTPConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NTP to configure either systemd-timesyncd or ntpd</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.coreos.os.extensions.gardener.cloud/v1alpha1.NTPConfig">NTPConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#config.coreos.os.extensions.gardener.cloud/v1alpha1.ExtensionConfig">ExtensionConfig</a>)
</p>
<p>
<p>NTPConfig General NTP Config for either systemd-timesyncd or ntpd</p>
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
<code>daemon</code></br>
<em>
<a href="#config.coreos.os.extensions.gardener.cloud/v1alpha1.Daemon">
Daemon
</a>
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
<a href="#config.coreos.os.extensions.gardener.cloud/v1alpha1.NTPDConfig">
NTPDConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NTPD to configure the ntpd client</p>
</td>
</tr>
</tbody>
</table>
<h3 id="config.coreos.os.extensions.gardener.cloud/v1alpha1.NTPDConfig">NTPDConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#config.coreos.os.extensions.gardener.cloud/v1alpha1.NTPConfig">NTPConfig</a>)
</p>
<p>
<p>NTPDConfig is the struct used in the ntp-config.conf.tpl template file</p>
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
[]string
</em>
</td>
<td>
<p>Servers List of ntp servers</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
