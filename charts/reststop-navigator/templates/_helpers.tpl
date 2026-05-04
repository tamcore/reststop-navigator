{{/*
Expand the name of the chart.
*/}}
{{- define "reststop-navigator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully-qualified app name.
*/}}
{{- define "reststop-navigator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "reststop-navigator.redis.fullname" -}}
{{- printf "%s-redis" (include "reststop-navigator.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "reststop-navigator.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/name: {{ include "reststop-navigator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "reststop-navigator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "reststop-navigator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "reststop-navigator.redisURL" -}}
{{- if .Values.redis.enabled -}}
redis://{{ include "reststop-navigator.redis.fullname" . }}:6379/0
{{- else -}}
{{ required "redis.enabled=false requires externalRedisURL" .Values.externalRedisURL }}
{{- end -}}
{{- end -}}

{{- define "reststop-navigator.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- if .Values.image.digest -}}
{{ .Values.image.repository }}@{{ .Values.image.digest }}
{{- else -}}
{{ .Values.image.repository }}:{{ $tag }}
{{- end -}}
{{- end -}}
