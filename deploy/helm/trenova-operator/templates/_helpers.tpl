{{- define "operator.name" -}}
{{- printf "%s" .Release.Name | trunc 50 | trimSuffix "-" -}}
{{- end -}}

{{- define "operator.labels" -}}
app.kubernetes.io/name: trenova-operator
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "operator.selector" -}}
app.kubernetes.io/name: trenova-operator
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
