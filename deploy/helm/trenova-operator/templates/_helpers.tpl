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

{{/* The Loki of the target release, as the Trenova chart names it:
<fullname>-loki, the fullname being the release, or <release>-trenova when the
release's name does not contain the chart's, cut to 40 characters. */}}
{{- define "operator.lokiURL" -}}
{{- $release := .Values.target.release -}}
{{- $fullname := ternary $release (printf "%s-trenova" $release) (contains "trenova" $release) | trunc 40 | trimSuffix "-" -}}
{{- printf "http://%s-loki.%s.svc:3100" $fullname .Values.target.namespace -}}
{{- end -}}

{{/* The operator's image: the digest from image-digests.yaml when the chart is a
published one, as in the Trenova chart's trenova.image. */}}
{{- define "operator.image" -}}
{{- $ref := printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) -}}
{{- $digest := get (.Files.Get "image-digests.yaml" | fromYaml) $ref -}}
{{- if $digest -}}
{{- printf "%s@%s" $ref $digest -}}
{{- else -}}
{{- $ref -}}
{{- end -}}
{{- end -}}
