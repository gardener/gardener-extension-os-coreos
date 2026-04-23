<p>Packages:</p>
<ul>
<li>
<a href="#config.coreos.os.extensions.gardener.cloud%2fv1alpha1">config.coreos.os.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="config.coreos.os.extensions.gardener.cloud/v1alpha1">config.coreos.os.extensions.gardener.cloud/v1alpha1</h2>
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


