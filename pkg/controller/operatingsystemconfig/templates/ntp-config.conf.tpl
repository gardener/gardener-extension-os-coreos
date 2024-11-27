{{- range .Servers }}
server {{ . }} iburst
{{- end }}

driftfile /var/lib/ntp/ntp.drift
restrict default nomodify nopeer noquery notrap limited kod
restrict 127.0.0.1
restrict [::1]