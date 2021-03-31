{{/* vim: set filetype=mustache: */}}

{{- define "graphql-name" -}}
{{- "crossplane-graphql" -}}
{{- end -}}

{{/*
Labels - graphql
*/}}
{{- define "labelsGraphql" -}}
{{ include "labels" . }}
app.kubernetes.io/component: graphql
{{- end }}

{{/*
Selector labels - graphql
*/}}
{{- define "selectorLabelsGraphql" -}}
{{ include "selectorLabels" . }}
app.kubernetes.io/component: graphql
{{- end }}

