{{/*
A CloudNativePG Cluster for one generation of the database. The live one is
made by initdb, or, once a restore has replaced it, by recovery from the
backups; a candidate is a database restored beside the live one, which
nothing uses until it is switched to. Both render from here, so a candidate
that becomes the live database renders the same.

  root       the chart's root context
  name       the cluster's name
  recovery   nil, or the generation it is restored from and the time:
             (dict "sourceGeneration" … "targetTime" …)
  candidate  whether it is a restore not switched to yet
*/}}
{{- define "trenova.postgresClusterResource" -}}
{{- $root := .root -}}
{{- $v := $root.Values -}}
{{- $catalog := include "trenova.postgresClusterOf" (dict "root" $root "generation" "") -}}
{{- $plugin := "barman-cloud.cloudnative-pg.io" -}}
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: {{ .name }}
  labels:
    {{- include "trenova.labels" (dict "root" $root "component" "postgres") | nindent 4 }}
  annotations:
    {{- if not .candidate }}
    # The operator deletes a cluster's volumes with it, so uninstalling the
    # release, or a restore switching to another generation, leaves the
    # cluster, and the data, in place.
    helm.sh/resource-policy: keep
    {{- end }}
    # Hibernation stops every instance and keeps the volumes.
    cnpg.io/hibernation: {{ ternary "on" "off" (and $v.postgres.hibernate (not .candidate)) | quote }}
spec:
  instances: {{ ternary 1 $v.postgres.instances .candidate }}
  imageCatalogRef:
    apiGroup: postgresql.cnpg.io
    kind: ImageCatalog
    name: {{ $catalog }}
    major: {{ $v.postgres.majorVersion }}
  postgresUID: {{ $v.postgres.uid }}
  postgresGID: {{ $v.postgres.gid }}
  # Only the release Job's first and last steps sign in as the superuser
  # (files/db-prepare.sh, files/db-publication.sh).
  enableSuperuserAccess: true
  primaryUpdateStrategy: unsupervised
  # A short wait for clients, then a fast (clean) shutdown; values.yaml has why.
  smartShutdownTimeout: {{ $v.postgres.smartShutdownTimeout | int }}
  stopDelay: {{ $v.postgres.stopDelay | int }}
  {{- with $v.imagePullSecrets }}
  imagePullSecrets:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  bootstrap:
    {{- if .recovery }}
    {{- $source := include "trenova.postgresClusterOf" (dict "root" $root "generation" .recovery.sourceGeneration) }}
    # Restored from the backups the source generation archived, replaying its
    # WAL up to the target time.
    recovery:
      source: {{ $source }}
      recoveryTarget:
        targetTime: {{ .recovery.targetTime | quote }}
    {{- else }}
    initdb:
      database: {{ $v.postgres.database }}
      owner: {{ $v.postgres.owner }}
      secret:
        name: {{ include "trenova.dbSecret" $root }}
      dataChecksums: true
      encoding: UTF8
      localeProvider: builtin
      builtinLocale: C.UTF-8
    {{- end }}
  {{- if .recovery }}
  externalClusters:
    - name: {{ include "trenova.postgresClusterOf" (dict "root" $root "generation" .recovery.sourceGeneration) }}
      plugin:
        name: {{ $plugin }}
        parameters:
          barmanObjectName: {{ include "trenova.backupStore" $root }}
          serverName: {{ include "trenova.postgresClusterOf" (dict "root" $root "generation" .recovery.sourceGeneration) }}
  {{- end }}
  {{- if $v.backup.enabled }}
  # WAL is archived to the backups bucket as it is written, under this
  # generation's own name, so a restored generation never writes over the
  # backups it came from.
  plugins:
    - name: {{ $plugin }}
      isWALArchiver: true
      parameters:
        barmanObjectName: {{ include "trenova.backupStore" $root }}
        serverName: {{ .name }}
  {{- end }}
  postgresql:
    shared_preload_libraries:
      - pg_cron
      - pg_prewarm
    parameters:
      {{- range $k, $val := $v.postgres.parameters }}
      {{ $k }}: {{ $val | toString | quote }}
      {{- end }}
      cron.database_name: {{ $v.postgres.database | quote }}
    # TLS only: every client verifies the server against the cluster's CA. The
    # operator's own replication and status connections use certificates, above
    # these lines.
    pg_hba:
      - hostssl all all all scram-sha-256
      - host all all all reject
  storage:
    size: {{ $v.postgres.storage.size }}
    {{- include "trenova.storageClass" $v.postgres.storage | nindent 4 }}
  resources:
    {{- if .candidate }}
    # A candidate only replays WAL until it is switched to: it asks for
    # little of the node beside the live database, with the same ceiling.
    requests: { cpu: 250m, memory: 512Mi }
    {{- with $v.postgres.resources.limits }}
    limits:
      {{- toYaml . | nindent 6 }}
    {{- end }}
    {{- else }}
    {{- toYaml $v.postgres.resources | nindent 4 }}
    {{- end }}
  inheritedMetadata:
    labels:
      {{- include "trenova.selectorLabels" (dict "root" $root "component" "postgres") | nindent 6 }}
  {{- with $v.affinity }}
  affinity:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end -}}
