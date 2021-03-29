{{/* vim: set filetype=mustache: */}}

{{- define "agent-name" -}}
{{- "upbound-agent" -}}
{{- end -}}

{{/*
Labels - agent
*/}}
{{- define "labelsAgent" -}}
{{ include "labels" . }}
app.kubernetes.io/component: agent
{{- end }}

{{/*
Selector labels - agent
*/}}
{{- define "selectorLabelsAgent" -}}
{{ include "selectorLabels" . }}
app.kubernetes.io/component: agent
{{- end }}

