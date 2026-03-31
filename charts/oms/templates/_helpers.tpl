{{/*
Expand the name of the chart.
*/}}
{{- define "oms.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "oms.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "oms.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "oms.labels" -}}
helm.sh/chart: {{ include "oms.chart" . }}
{{ include "oms.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "oms.selectorLabels" -}}
app.kubernetes.io/name: {{ include "oms.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "oms.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "oms.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Database connection string
*/}}
{{- define "oms.databaseUrl" -}}
{{- $host := .Values.database.host }}
{{- $port := .Values.database.port }}
{{- $name := .Values.database.name }}
{{- $user := .Values.database.user }}
{{- $sslMode := .Values.database.sslMode }}
{{- printf "postgres://%s:$(DATABASE_PASSWORD)@%s:%d/%s?sslmode=%s" $user $host (int $port) $name $sslMode }}
{{- end }}

{{/*
Kafka brokers
*/}}
{{- define "oms.kafkaBrokers" -}}
{{- .Values.kafka.brokers }}
{{- end }}

{{/*
Database secret name
*/}}
{{- define "oms.databaseSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- include "oms.fullname" . }}-db
{{- end }}
{{- end }}

{{/*
Shopify webhook secret name
*/}}
{{- define "oms.shopifySecretName" -}}
{{- if .Values.shopify.existingSecret }}
{{- .Values.shopify.existingSecret }}
{{- else }}
{{- include "oms.fullname" . }}-shopify
{{- end }}
{{- end }}
