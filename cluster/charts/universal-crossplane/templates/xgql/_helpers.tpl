{{/* vim: set filetype=mustache: */}}

{{- define "xgql-name" -}}
{{- "xgql" -}}
{{- end -}}

{{/*
Labels - xgql
*/}}
{{- define "labelsXgql" -}}
{{ include "labels" . }}
app.kubernetes.io/component: xgql
{{- end }}

{{/*
Selector labels - xgql
*/}}
{{- define "selectorLabelsXgql" -}}
{{ include "selectorLabels" . }}
app.kubernetes.io/component: xgql
{{- end }}

