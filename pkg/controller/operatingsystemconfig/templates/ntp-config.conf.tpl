{{- range .Servers }}
server {{ . }} iburst
{{- end }}

driftfile /var/lib/ntp/ntp.drift
restrict default nomodify nopeer noquery notrap limited kod
restrict 127.0.0.1
restrict [::1]

{{ if .Interfaces -}}
interface ignore wildcard
interface listen 127.0.0.1
interface listen {{ .Interfaces | join " " }}
{{- end }}