{{/*
The TMS configuration (/app/config/config.yaml): what the release itself
decides (hosts, origins, the roles, which components run), with tms.config
merged over it. Secrets are not here; trenova.tmsEnv passes them as
TRENOVA_<SECTION>_<KEY> variables, which override these keys.
*/}}
{{- define "trenova.config" -}}
{{- /* The loader applies a TRENOVA_* variable only to a key it knows, so every
key a Secret supplies is written here empty: the variable fills it, and an
empty one fails validation instead of running with a default. */ -}}
{{- $origin := include "trenova.origin" . -}}
{{- $database := dict
  "host" (include "trenova.postgresHost" .)
  "port" 5432
  "name" .Values.postgres.database
  "user" .Values.postgres.appRole
  "password" ""
  "sslMode" "verify-full"
  "sslRootCert" "/certs/postgres/ca.crt"
  "system" (dict "user" .Values.postgres.systemRole "password" "" "maxOpenConns" 8)
  "migrator" (dict "user" .Values.postgres.owner "password" "")
  "rls" (dict "mode" .Values.postgres.rlsMode "scopeKeyId" "" "scopeKey" "")
-}}
{{- $storage := dict
  "provider" "minio"
  "bucket" .Values.storage.bucket
  "region" .Values.storage.region
  "autoCreateBucket" .Values.storage.autoCreateBucket
  "accessKey" ""
  "secretKey" ""
  "publicEndpoint" (include "trenova.storagePublicEndpoint" .)
-}}
{{- if .Values.minio.enabled }}
{{- $_ := set $storage "endpoint" (printf "%s:9000" (include "trenova.name" (dict "root" . "component" "minio"))) -}}
{{- $_ := set $storage "useSSL" false -}}
{{- else }}
{{- $_ := set $storage "endpoint" .Values.storage.external.endpoint -}}
{{- $_ := set $storage "useSSL" .Values.storage.external.useSSL -}}
{{- end }}
{{- $documentConversion := dict "doclingUrl" "" -}}
{{- if .Values.docling.enabled }}
{{- $documentConversion = dict
  "doclingUrl" (printf "http://%s:5001" (include "trenova.name" (dict "root" . "component" "docling")))
  "apiKey" ""
  "maxConcurrent" .Values.docling.workers
  "maxFileBytes" .Values.docling.maxFileBytes
  "maxPages" .Values.docling.maxPages
-}}
{{- end }}
{{- $contentScan := dict "clamav" (dict "address" "") -}}
{{- if .Values.clamav.enabled }}
{{- $contentScan = dict "clamav" (dict "address" (printf "%s:3310" (include "trenova.name" (dict "root" . "component" "clamav")))) -}}
{{- end }}
{{- $imaging := include "trenova.name" (dict "root" . "component" "imaging") -}}
{{- $release := dict
  "app" (dict
    "version" (.Values.tms.image.tag | default .Chart.AppVersion)
    "webBaseUrl" $origin
    "problemTypeBaseUri" (printf "%s/problems/" $origin))
  "platform" (dict "mode" .Values.tms.platformMode "instanceId" .Values.tms.instanceId)
  "server" (dict "cors" (dict "allowedOrigins" (list $origin)))
  "database" $database
  "cache" (dict "host" (include "trenova.name" (dict "root" . "component" "redis")) "port" 6379 "password" "")
  "search" (dict "meilisearch" (dict "url" (printf "http://%s:7700" (include "trenova.name" (dict "root" . "component" "meilisearch"))) "apiKey" ""))
  "storage" $storage
  "security" (dict
    "session" (dict "secret" "")
    "encryption" (dict "key" "")
    "mfa" (dict "recoveryCodePepper" "")
    "passwordReset" (dict "baseUrl" $origin)
    "csrf" (dict "trustedOrigins" (list $origin)))
  "monitoring" (dict "metrics" (dict "enabled" true "host" "0.0.0.0" "port" 9090))
  "temporal" (dict "hostPort" (printf "%s:7233" (include "trenova.name" (dict "root" . "component" "temporal"))))
  "renderer" (dict "gotenbergUrl" (printf "http://%s:3000" (include "trenova.name" (dict "root" . "component" "gotenberg"))))
  "documentConversion" $documentConversion
  "contentScan" $contentScan
  "portal" (dict "baseUrl" (printf "%s/dash" $origin))
  "reporting" (dict "deliveryLinkBaseUrl" $origin)
  "system" (dict "systemUserPassword" "")
  "imaging" (dict
    "endpoint" (printf "dns:///%s:50051" $imaging)
    "serverName" $imaging
    "tls" (dict
      "enabled" true
      "caFile" "/certs/imaging/ca.crt"
      "certFile" "/certs/imaging/tls.crt"
      "keyFile" "/certs/imaging/tls.key"))
-}}
{{- if .Values.managed.enabled }}
{{- $_ := set $release "managed" (dict "enabled" true "host" "0.0.0.0" "port" 8090 "token" "") -}}
{{- end }}
{{- mustMergeOverwrite $release (deepCopy .Values.tms.config) | toYaml -}}
{{- end -}}
