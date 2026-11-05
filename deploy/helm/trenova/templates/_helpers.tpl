{{/* Names */}}

{{- define "trenova.fullname" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | trunc 40 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 40 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "trenova.name" -}}
{{- printf "%s-%s" (include "trenova.fullname" .root) .component -}}
{{- end -}}

{{- define "trenova.postgresCluster" -}}
{{- printf "%s-postgres" (include "trenova.fullname" .) -}}
{{- end -}}

{{- define "trenova.postgresHost" -}}
{{- printf "%s-rw" (include "trenova.postgresCluster" .) -}}
{{- end -}}

{{/* The CA CloudNativePG issues the cluster's server certificate from, which
every client verifies PostgreSQL against. The Secret also holds the CA's key,
so a pod mounts the certificate alone. */}}
{{- define "trenova.postgresCaVolume" -}}
- name: postgres-ca
  secret:
    secretName: {{ printf "%s-ca" (include "trenova.postgresCluster" .) }}
    items:
      - key: ca.crt
        path: ca.crt
{{- end -}}

{{- define "trenova.postgresCaMount" -}}
- name: postgres-ca
  mountPath: /certs/postgres
  readOnly: true
{{- end -}}

{{/* libpq's settings for a verified connection. */}}
{{- define "trenova.postgresTlsEnv" -}}
- name: PGSSLMODE
  value: verify-full
- name: PGSSLROOTCERT
  value: /certs/postgres/ca.crt
{{- end -}}

{{/* The name the TMS and Gotenberg agree on; the password is in the app Secret. */}}
{{- define "trenova.gotenbergUsername" -}}
trenova
{{- end -}}

{{- define "trenova.appSecret" -}}
{{- .Values.secrets.existingSecret | default (printf "%s-secrets" (include "trenova.fullname" .)) -}}
{{- end -}}

{{- define "trenova.dbSecret" -}}
{{- .Values.postgres.existingSecret | default (printf "%s-db" (include "trenova.fullname" .)) -}}
{{- end -}}

{{- define "trenova.imagingServerSecret" -}}
{{- .Values.imaging.tls.existingServerSecret | default (printf "%s-imaging-server-tls" (include "trenova.fullname" .)) -}}
{{- end -}}

{{- define "trenova.imagingClientSecret" -}}
{{- .Values.imaging.tls.existingClientSecret | default (printf "%s-imaging-client-tls" (include "trenova.fullname" .)) -}}
{{- end -}}

{{/* Labels. Call with (dict "root" $ "component" "api"). */}}

{{- define "trenova.selectorLabels" -}}
app.kubernetes.io/name: {{ .root.Chart.Name }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{- define "trenova.labels" -}}
{{ include "trenova.selectorLabels" . }}
app.kubernetes.io/version: {{ include "trenova.appVersion" .root | quote }}
app.kubernetes.io/part-of: trenova
app.kubernetes.io/managed-by: {{ .root.Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .root.Chart.Name .root.Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "trenova.podLabels" -}}
{{ include "trenova.selectorLabels" . }}
{{- with .root.Values.podLabels }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "trenova.appVersion" -}}
{{- .Chart.AppVersion -}}
{{- end -}}

{{/* An image reference. Call with (dict "root" $ "image" .Values.x.image). The
tag defaults to the chart's appVersion; "<tag>@sha256:<digest>" pins a digest.
A published chart carries image-digests.yaml, written by the release workflow:
the digest of every image at the tag it names, inside the chart's signature. An
override to another repository or tag finds no entry and is used as given. */}}
{{- define "trenova.image" -}}
{{- $ref := printf "%s:%s" .image.repository (.image.tag | default .root.Chart.AppVersion) -}}
{{- $digest := get (.root.Files.Get "image-digests.yaml" | fromYaml) $ref -}}
{{- if $digest -}}
{{- printf "%s@%s" $ref $digest -}}
{{- else -}}
{{- $ref -}}
{{- end -}}
{{- end -}}

{{- define "trenova.pullPolicy" -}}
{{- .pullPolicy | default "IfNotPresent" -}}
{{- end -}}

{{/* Pod-level settings every pod shares. Call with the root context. */}}
{{- define "trenova.podCommon" -}}
automountServiceAccountToken: false
enableServiceLinks: false
{{- with .Values.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.nodeSelector }}
nodeSelector:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.tolerations }}
tolerations:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- with .Values.affinity }}
affinity:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end -}}

{{- define "trenova.podAnnotations" -}}
{{- $a := merge (dict) (.extra | default dict) .root.Values.podAnnotations -}}
{{- with $a }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{/* A pod security context: non-root as uid, with fsGroup for volumes. */}}
{{- define "trenova.podSecurity" -}}
securityContext:
  runAsNonRoot: true
  runAsUser: {{ .uid }}
  runAsGroup: {{ .uid }}
  fsGroup: {{ .uid }}
  fsGroupChangePolicy: OnRootMismatch
  seccompProfile:
    type: RuntimeDefault
{{- end -}}

{{/* A hardened container: no privilege escalation, no capabilities; pass
readOnly to make the root filesystem read-only. */}}
{{- define "trenova.containerSecurity" -}}
securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: {{ .readOnly | default false }}
  capabilities:
    drop: [ALL]
{{- end -}}

{{- define "trenova.storageClass" -}}
{{- if .storageClass }}
storageClassName: {{ .storageClass | quote }}
{{- end }}
{{- end -}}

{{/* URLs and origins */}}

{{- define "trenova.origin" -}}
{{- printf "https://%s" .Values.domain -}}
{{- end -}}

{{- define "trenova.storagePublicEndpoint" -}}
{{- if .Values.minio.enabled -}}
{{- printf "https://%s" .Values.storageDomain -}}
{{- else -}}
{{- .Values.storage.external.publicEndpoint -}}
{{- end -}}
{{- end -}}

{{/* A secret key as an env var. Call with (dict "name" "X" "secret" s "key" k). */}}
{{- define "trenova.secretEnv" -}}
- name: {{ .name }}
  valueFrom:
    secretKeyRef:
      name: {{ .secret }}
      key: {{ .key }}
      {{- if .optional }}
      optional: true
      {{- end }}
{{- end -}}

{{/* The environment of every TMS process: the API, the worker and the
release Job's steps. Secrets only; everything else is in the rendered config. */}}
{{- define "trenova.tmsEnv" -}}
{{- $app := include "trenova.appSecret" . -}}
{{- $db := include "trenova.dbSecret" . -}}
- name: APP_ENV
  value: production
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_DATABASE_PASSWORD" "secret" $db "key" "appPassword") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_DATABASE_SYSTEM_PASSWORD" "secret" $db "key" "systemPassword") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_DATABASE_RLS_SCOPEKEYID" "secret" $app "key" "rlsScopeKeyId") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_DATABASE_RLS_SCOPEKEY" "secret" $app "key" "rlsScopeKey") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_CACHE_PASSWORD" "secret" $app "key" "redisPassword") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_SEARCH_MEILISEARCH_APIKEY" "secret" $app "key" "meiliMasterKey") }}
{{- if .Values.minio.enabled }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_STORAGE_ACCESSKEY" "secret" $app "key" "storageAccessKey") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_STORAGE_SECRETKEY" "secret" $app "key" "storageSecretKey") }}
{{- else }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_STORAGE_ACCESSKEY" "secret" .Values.storage.external.existingSecret "key" "accessKey") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_STORAGE_SECRETKEY" "secret" .Values.storage.external.existingSecret "key" "secretKey") }}
{{- end }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_SECURITY_SESSION_SECRET" "secret" $app "key" "sessionSecret") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_SECURITY_ENCRYPTION_KEY" "secret" $app "key" "encryptionKey") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_SECURITY_ENCRYPTION_RETIREDKEYS" "secret" $app "key" "encryptionRetiredKeys" "optional" true) }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_SECURITY_MFA_RECOVERYCODEPEPPER" "secret" $app "key" "mfaRecoveryCodePepper") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_SYSTEM_SYSTEMUSERPASSWORD" "secret" $app "key" "systemUserPassword") }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_RENDERER_GOTENBERGPASSWORD" "secret" $app "key" "gotenbergPassword") }}
{{- if .Values.docling.enabled }}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_DOCUMENTCONVERSION_APIKEY" "secret" $app "key" "doclingApiKey") }}
{{- end }}
{{- with .Values.tms.extraEnv }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{/* The owner's password, for the release Job's steps alone: migrations and
role provisioning run as it (database.migrator), and nothing else needs it. */}}
{{- define "trenova.migratorEnv" -}}
{{ include "trenova.secretEnv" (dict "name" "TRENOVA_DATABASE_MIGRATOR_PASSWORD" "secret" (include "trenova.dbSecret" .) "key" "password") }}
{{- end -}}

{{/* What a TMS container mounts: the rendered config and the imaging client
certificate. */}}
{{- define "trenova.tmsVolumeMounts" -}}
- name: config
  mountPath: /app/config
  readOnly: true
- name: imaging-client-tls
  mountPath: /certs/imaging
  readOnly: true
{{ include "trenova.postgresCaMount" . }}
- name: tmp
  mountPath: /tmp
{{- end -}}

{{- define "trenova.tmsVolumes" -}}
- name: config
  configMap:
    name: {{ include "trenova.name" (dict "root" . "component" "config") }}
- name: imaging-client-tls
  secret:
    secretName: {{ include "trenova.imagingClientSecret" . }}
    defaultMode: 0440
{{ include "trenova.postgresCaVolume" . }}
- name: tmp
  emptyDir:
    sizeLimit: 1Gi
{{- end -}}

{{- define "trenova.tmsChecksums" -}}
checksum/config: {{ include "trenova.config" . | sha256sum }}
{{- end -}}

{{/* An init container that waits until a role can sign in to PostgreSQL and a
query returns a row. Call with (dict "root" $ "user" u "secret" s "key" k
"database" d "query" q "why" "text"). */}}
{{- define "trenova.waitForPostgres" -}}
- name: wait-for-postgres
  image: {{ include "trenova.image" (dict "root" .root "image" .root.Values.postgres.image) }}
  command:
    - sh
    - -c
    - |
      until psql -qtAc "$WAIT_QUERY" 2>/dev/null | grep -q .; do
        echo "Waiting: $WAIT_REASON"
        sleep 5
      done
  env:
    - name: PGHOST
      value: {{ include "trenova.postgresHost" .root }}
    - name: PGDATABASE
      value: {{ .database }}
    - name: PGUSER
      value: {{ .user }}
    {{- include "trenova.secretEnv" (dict "name" "PGPASSWORD" "secret" .secret "key" .key) | nindent 4 }}
    {{- include "trenova.postgresTlsEnv" . | nindent 4 }}
    - name: PGCONNECT_TIMEOUT
      value: "5"
    - name: WAIT_QUERY
      value: {{ .query | quote }}
    - name: WAIT_REASON
      value: {{ .why | quote }}
  {{- include "trenova.containerSecurity" (dict "readOnly" true) | nindent 2 }}
  resources:
    requests: { cpu: 10m, memory: 16Mi }
    limits: { memory: 64Mi }
  volumeMounts:
    {{- include "trenova.postgresCaMount" . | nindent 4 }}
{{- end -}}

{{/* The TMS's own role, signed in: the release Job has made it. The processes
then wait for the schema themselves. */}}
{{- define "trenova.waitForTmsRole" -}}
{{ include "trenova.waitForPostgres" (dict "root" . "user" .Values.postgres.appRole "secret" (include "trenova.dbSecret" .) "key" "appPassword" "database" .Values.postgres.database "query" "SELECT 1" "why" "the application role (the release Job provisions it)") }}
{{- end -}}

{{- define "trenova.dnsEgress" -}}
- to:
    - namespaceSelector: {}
  ports:
    - port: 53
      protocol: UDP
    - port: 53
      protocol: TCP
{{- end -}}

{{/* A workload's replica count: none while the release is stopped. Call with
(dict "root" $ "replicas" n). */}}
{{- define "trenova.replicas" -}}
{{- if .root.Values.stopped }}0{{ else }}{{ .replicas }}{{ end -}}
{{- end -}}
