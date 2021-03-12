{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

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

