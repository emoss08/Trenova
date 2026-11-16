{{/* Log keeping (logs.enabled): Loki's configuration and Alloy's pipeline.
Both are rendered from here so templates/logs.yaml can checksum them. */}}

{{- define "trenova.lokiPort" -}}
3100
{{- end -}}

{{- define "trenova.lokiURL" -}}
{{- printf "http://%s:%s" (include "trenova.name" (dict "root" . "component" "loki")) (include "trenova.lokiPort" .) -}}
{{- end -}}

{{/* Loki, one process with every module (-target=all), its chunks, TSDB index
and write-ahead log on its own volume at /var/loki. The compactor deletes
whatever is older than logs.retentionHours. The limits keep one noisy
component from filling a small node's memory or disk, and still allow a
query over the whole retention, which the operator reads in pages of
max_entries_limit_per_query lines. */}}
{{- define "trenova.lokiConfig" -}}
{{- $retention := .Values.logs.retentionHours | int -}}
auth_enabled: false
server:
  http_listen_port: {{ include "trenova.lokiPort" . }}
  grpc_listen_port: 9095
  log_level: warn
  http_server_write_timeout: 70s
  grpc_server_max_recv_msg_size: 16777216
  grpc_server_max_send_msg_size: 16777216
common:
  instance_addr: 127.0.0.1
  path_prefix: /var/loki
  replication_factor: 1
  ring:
    kvstore:
      store: inmemory
  storage:
    filesystem:
      chunks_directory: /var/loki/chunks
      rules_directory: /var/loki/rules
schema_config:
  configs:
    - from: "2026-01-01"
      store: tsdb
      object_store: filesystem
      schema: v13
      index:
        prefix: index_
        period: 24h
compactor:
  working_directory: /var/loki/compactor
  compaction_interval: 10m
  retention_enabled: true
  retention_delete_delay: 2h
  retention_delete_worker_count: 10
  delete_request_store: filesystem
limits_config:
  retention_period: {{ printf "%dh" $retention }}
  reject_old_samples: true
  reject_old_samples_max_age: {{ printf "%dh" $retention }}
  ingestion_rate_mb: 4
  ingestion_burst_size_mb: 8
  per_stream_rate_limit: 2MB
  per_stream_rate_limit_burst: 8MB
  max_global_streams_per_user: 1000
  max_line_size: 256KB
  max_line_size_truncate: true
  max_query_length: {{ printf "%dh" (add $retention 1) }}
  max_entries_limit_per_query: 5000
  max_query_series: 200
  max_query_parallelism: 4
  tsdb_max_query_parallelism: 8
  split_queries_by_interval: 6h
  query_timeout: 1m
  discover_service_name: []
  discover_log_levels: false
querier:
  max_concurrent: 2
query_range:
  results_cache:
    cache:
      embedded_cache:
        enabled: true
        max_size_mb: 32
analytics:
  reporting_enabled: false
{{- end -}}

{{/* Alloy's pipeline: the release's pods in its own namespace, discovered
and tailed through the Kubernetes API (init containers included: the release
Job's steps are init containers), each line labelled with the pod's
component, the pod and the container and nothing else, pushed to Loki. A
pod without a component label is not read. */}}
{{- define "trenova.alloyConfig" -}}
discovery.kubernetes "pods" {
  role = "pod"

  namespaces {
    names = [{{ .Release.Namespace | quote }}]
  }

  selectors {
    role  = "pod"
    label = {{ printf "app.kubernetes.io/name=%s,app.kubernetes.io/instance=%s" .Chart.Name .Release.Name | quote }}
  }
}

discovery.relabel "pods" {
  targets = discovery.kubernetes.pods.targets

  rule {
    source_labels = ["__meta_kubernetes_pod_label_app_kubernetes_io_component"]
    target_label  = "component"
  }

  rule {
    source_labels = ["__meta_kubernetes_pod_name"]
    target_label  = "pod"
  }

  rule {
    source_labels = ["__meta_kubernetes_pod_container_name"]
    target_label  = "container"
  }

  rule {
    source_labels = ["component"]
    regex         = ""
    action        = "drop"
  }
}

loki.source.kubernetes "pods" {
  targets    = discovery.relabel.pods.output
  forward_to = [loki.relabel.streams.receiver]
}

loki.relabel "streams" {
  forward_to = [loki.write.local.receiver]

  rule {
    action = "labelkeep"
    regex  = "component|pod|container"
  }
}

loki.write "local" {
  endpoint {
    url = {{ printf "%s/loki/api/v1/push" (include "trenova.lokiURL" .) | quote }}
  }
}
{{- end -}}
