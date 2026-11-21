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

{{/* The operator's image, as the Trenova chart's trenova.image renders one: from
the release manifest a published chart carries, <repository>:<the version it
was built under>@<digest>, so a release that did not rebuild the operator
leaves its pod alone; otherwise <repository>:<tag, or appVersion>, with the
digest image-digests.yaml gives it. */}}
{{- define "operator.image" -}}
{{- $tag := toString (.Values.image.tag | default "") -}}
{{- $pinned := "" -}}
{{- $raw := .Files.Get "release-manifest.json" -}}
{{- if and $raw (not (contains "@" $tag)) -}}
{{- $manifest := fromJson $raw -}}
{{- if ne (toString (get $manifest "schemaVersion")) "1" -}}
{{- fail (printf "release-manifest.json has schemaVersion %v; this chart reads 1" (get $manifest "schemaVersion")) -}}
{{- end -}}
{{- range $name, $e := get $manifest "images" -}}
{{- if and (not $pinned) (eq $e.repository $.Values.image.repository) (or (not $tag) (eq $tag $e.version)) -}}
{{- $pinned = printf "%s:%s@%s" $e.repository $e.version $e.digest -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- if $pinned -}}
{{- $pinned -}}
{{- else -}}
{{- $ref := printf "%s:%s" .Values.image.repository ($tag | default .Chart.AppVersion) -}}
{{- $digest := get (.Files.Get "image-digests.yaml" | fromYaml) $ref -}}
{{- if $digest -}}
{{- printf "%s@%s" $ref $digest -}}
{{- else -}}
{{- $ref -}}
{{- end -}}
{{- end -}}
{{- end -}}
