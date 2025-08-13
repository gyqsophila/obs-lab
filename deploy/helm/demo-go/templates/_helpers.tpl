
{{- define "demo-go.labels" -}}
app.kubernetes.io/name: {{ include "demo-go.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "demo-go.selectorLabels" -}}
app.kubernetes.io/name: {{ include "demo-go.name" . }}
{{- end }}

{{- define "demo-go.name" -}}
{{ .Chart.Name }}
{{- end }}

{{- define "demo-go.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end }}
