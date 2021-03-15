{{/* vim: set filetype=mustache: */}}

{{- define "agent-name" -}}
{{- "upbound-agent" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "labels" -}}
helm.sh/chart: {{ include "chart" . }}
{{ include "selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Labels - gateway
*/}}
{{- define "labelsGateway" -}}
{{ include "labels" . }}
app.kubernetes.io/component: gateway
{{- end }}

{{/*
Labels - graphql
*/}}
{{- define "labelsGraphql" -}}
{{ include "labels" . }}
app.kubernetes.io/component: graphql
{{- end }}

{{/*
Selector labels
*/}}
{{- define "selectorLabels" -}}
app.kubernetes.io/name: {{ include "name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Selector labels - gateway
*/}}
{{- define "selectorLabelsGateway" -}}
{{ include "selectorLabels" . }}
app.kubernetes.io/component: gateway
{{- end }}

{{/*
Selector labels - graphql
*/}}
{{- define "selectorLabelsGraphql" -}}
{{ include "selectorLabels" . }}
app.kubernetes.io/component: graphql
{{- end }}

