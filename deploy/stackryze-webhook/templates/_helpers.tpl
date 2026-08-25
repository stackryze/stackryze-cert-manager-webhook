{{- define "stackryze-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "stackryze-webhook.fullname" -}}
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

{{- define "stackryze-webhook.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "stackryze-webhook.fullname" .) }}
{{- end -}}

{{- define "stackryze-webhook.rootCAIssuer" -}}
{{ printf "%s-ca" (include "stackryze-webhook.fullname" .) }}
{{- end -}}

{{- define "stackryze-webhook.rootCACertificate" -}}
{{ printf "%s-ca" (include "stackryze-webhook.fullname" .) }}
{{- end -}}

{{- define "stackryze-webhook.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "stackryze-webhook.fullname" .) }}
{{- end -}}

{{- define "stackryze-webhook.labels" -}}
app: {{ include "stackryze-webhook.name" . }}
release: {{ .Release.Name }}
{{- end -}}
