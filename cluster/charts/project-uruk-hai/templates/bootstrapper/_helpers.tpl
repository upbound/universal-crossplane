{{/* vim: set filetype=mustache: */}}

{{- define "bootstrapper-name" -}}
{{- "upbound-bootstrapper" -}}
{{- end -}}

{{/*
Labels - bootstrapper
*/}}
{{- define "labelsBootstrapper" -}}
{{ include "labels" . }}
app.kubernetes.io/component: bootstrapper
{{- end }}

{{/*
Selector labels - gateway
*/}}
{{- define "selectorLabelsBootstrapper" -}}
{{ include "selectorLabels" . }}
app.kubernetes.io/component: bootstrapper
{{- end }}