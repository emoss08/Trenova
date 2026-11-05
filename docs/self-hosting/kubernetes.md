# Trenova on Kubernetes

The Helm chart in [`deploy/helm/trenova`](../../deploy/helm/trenova) runs Trenova on a
Kubernetes cluster: the API, the worker, the imaging worker, the client, and everything they
need, with PostgreSQL run by [CloudNativePG](https://cloudnative-pg.io). It is the same
software as the [Docker Compose stack](README.md), and Trenova Cloud runs each customer's
instance from it, on a single-node k3s cluster per instance.

Compose is still the simpler choice for one host you look after by hand. Choose the chart when
you already run Kubernetes, or want the database run by an operator.

## Contents

- [What runs](#what-runs)
- [Requirements](#requirements)
- [Install](#install)
- [Values](#values)
- [The release Job](#the-release-job)
- [Secrets](#secrets)
- [PostgreSQL](#postgresql)
- [Network policies](#network-policies)
- [Upgrades](#upgrades)
- [Backups](#backups)
- [Troubleshooting](#troubleshooting)
- [Uninstall](#uninstall)

## What runs

| Component | Kind | Notes |
|---|---|---|
| `api`, `worker` | Deployment | The TMS. Probes on the monitoring port (9090): `/live`, `/ready` |
| `release` | Job, one per Helm revision | Prepares PostgreSQL, migrates, provisions roles, publishes for GTC, bootstraps |
| `postgres` | CloudNativePG `Cluster` | The Trenova image (PostGIS, pg_cron, pgvector), TLS only |
| `imaging` | Deployment | Reached over mutual TLS with certificates the chart issues |
| `temporal` | Deployment | auto-setup, in the same PostgreSQL cluster |
| `redis`, `meilisearch`, `minio` | StatefulSet | MinIO is optional: point `storage.external` at S3 instead |
| `gotenberg` | Deployment | No network, read-only root |
| `gtc` | Deployment | Change data capture into Meilisearch and Redis |
| `client` | Deployment | Caddy serving the web app and Dash, behind your Ingress |
| `docling`, `clamav` | Deployment, StatefulSet | Off by default; see [Values](#values) |

Every container runs as a non-root user with no capabilities, under the runtime's default
seccomp profile, and with a read-only root filesystem, except ClamAV, Temporal and GTC,
whose images need to write to theirs.

## Requirements

- Kubernetes 1.29 or newer, with a default StorageClass and a CNI that enforces
  NetworkPolicy (k3s, Calico, Cilium). k3s on one node works.
- The CloudNativePG operator, the version `Chart.yaml` names under
  `annotations.trenova.app/cloudnative-pg` (1.30.1).
- An Ingress controller, and a way to get certificates for it (cert-manager, or your
  controller's own), unless something else fronts the API.
- At least 4 vCPU and 8 GiB of memory for the release, more with document extraction or
  malware scanning on. The defaults request about 3.3 vCPU and 6 GiB.

## Install

Install the operator once per cluster:

```bash
kubectl apply --server-side -f https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.30/releases/cnpg-1.30.1.yaml
```

Write your values, at minimum the two hostnames, the Ingress and the first administrator:

```yaml
# trenova-values.yaml
domain: trenova.example.com
storageDomain: storage.trenova.example.com
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt
bootstrap:
  orgName: Example Freight
  adminName: Ada Admin
  adminEmail: ada@example.com
  timezone: America/Chicago
  state: TX
  city: Austin
  postalCode: "78701"
  addressLine1: 1 Main St
  scac: EXFR
  dotNumber: "1234567"
```

Then install a release, pinned to its version:

```bash
helm install trenova oci://ghcr.io/emoss08/charts/trenova --version 0.11.0 \
  -n trenova --create-namespace -f trenova-values.yaml --wait --wait-for-jobs --timeout 30m
```

The first install pulls the images, starts PostgreSQL and applies every migration, which
takes several minutes. When it returns, point DNS for both hostnames at the Ingress and sign
in as the administrator; the install's notes print the command that reads the generated
password.

`helm test trenova -n trenova` checks that the API answers, directly and through the client.

## Values

[`values.yaml`](../../deploy/helm/trenova/values.yaml) documents every value, and
`values.schema.json` refuses a misspelt key or a missing hostname before anything is applied.
The ones you are most likely to change:

| Value | What it does |
|---|---|
| `domain`, `storageDomain` | The hostnames. `storageDomain` serves presigned uploads and downloads from the bundled MinIO |
| `ingress.*` | The Ingress for both hostnames, to the client |
| `bootstrap.*` | The first organization and administrator; skipped while `adminEmail` is empty |
| `minio.enabled`, `storage.external.*` | Bundled MinIO, or an S3-compatible service and a Secret with `accessKey` and `secretKey` |
| `docling.enabled` | Document extraction (docling-serve, 2 to 4 GiB of memory) |
| `clamav.enabled` | Malware scanning; files are refused while clamd is down |
| `tms.config` | Merged over the TMS configuration the chart renders; see `services/tms/config/config.example.yaml` |
| `tms.extraEnv` | Integration credentials and other `TRENOVA_*` settings, as env entries (`valueFrom` works) |
| `*.resources`, `postgres.parameters` | Sizing; raise `shared_buffers` with the PostgreSQL memory limit |
| `secrets.existingSecret`, `postgres.existingSecret` | Secrets you manage; see [Secrets](#secrets) |
| `stopped`, `postgres.hibernate` | Stop every workload but PostgreSQL, and PostgreSQL too, keeping every volume; set both back to `false` to start again |

The TMS decodes its configuration strictly, so a key in `tms.config` that the release does
not know stops the API, the worker and the release Job; their logs name the key.

## The release Job

Each install, upgrade and rollback runs one Job, `trenova-release-<revision>`, whose steps run
in order:

1. **prepare**, as the CloudNativePG superuser: makes the owner a superuser (migrations create
   event triggers and extensions), and creates Temporal's and GTC's roles and the
   `temporal` and `temporal_visibility` databases. Every role's password is set again from its
   Secret, so a restored cluster takes the release's passwords.
2. **migrate**: `trenova-db up`, as the owner: the migrations and every reference seed that
   changed.
3. **rls-roles**: `trenova db rls provision-roles` creates the application and system roles
   (skipped with `postgres.rlsMode: off`).
4. **publication**: scopes GTC's publication to the tables
   [`deploy/gtc/sinks.yaml`](../../deploy/gtc/sinks.yaml) routes, and lets GTC's role read them.
5. **bootstrap**: `trenova db bootstrap`, when `bootstrap.adminEmail` is set. Later runs are
   no-ops; different values, or a database that already has users, fail the step.

Every step is idempotent. The API and worker start alongside the Job and wait for their role,
then for their schema version, so `--wait --wait-for-jobs` waits for all of it. Follow it with:

```bash
kubectl -n trenova logs -f job/trenova-release-1 --all-containers
```

## Secrets

Unless you supply them, the chart generates the application's secrets
(`trenova-secrets`), the database roles' passwords (`trenova-db`) and the imaging worker's
certificates once, and keeps them on every upgrade by reading the live Secrets back. The first
two are kept when the release is uninstalled, so a reinstall over the same volumes can still
read its data.

**Back up `trenova-secrets` apart from the cluster.** Its `encryptionKey` is the only way to
read encrypted data, and losing `mfaRecoveryCodePepper` invalidates every recovery code.

Supply your own Secrets (`secrets.existingSecret`, `postgres.existingSecret`,
`imaging.tls.existingServerSecret` and `existingClientSecret`) when:

- the chart is rendered without a cluster (`helm template`, Argo CD, Flux): the chart reads
  nothing back there and would generate new values on every render;
- the secrets are held elsewhere: a vault, External Secrets, or Trenova Cloud's control plane.

`values.yaml` lists the keys each Secret must hold.

## PostgreSQL

CloudNativePG runs the cluster `trenova-postgres` from the Trenova image through an
`ImageCatalog` (its tag is Trenova's version, not PostgreSQL's), with data checksums, the
`builtin` `C.UTF-8` locale, and `pg_hba` refusing every connection that is not TLS.

| Role | Login | Used by |
|---|---|---|
| `trenova` (owner) | yes, superuser | the release Job only |
| `trenova_app` | yes | the API and worker, bound by row-level security |
| `trenova_system` | yes | the API and worker's `dbscope.WithSystem` code paths |
| `temporal` | yes | Temporal, owner of its two databases |
| `trenova_gtc` | yes, replication | GTC: logical replication and its backfill reads |

`postgres.rlsMode` defaults to `enforce`: PostgreSQL itself refuses another organization's
rows ([row-level security](../engineering/row-level-security.md)). `off` connects the API and
worker as the owner, as the Compose stack does.

## Network policies

With `networkPolicy.enabled` (the default), every pod in the release refuses connections
unless a policy admits them: each backing service admits only the components that use it,
PostgreSQL admits those and the operator's namespace (`networkPolicy.cnpgNamespace`), and
Redis, Meilisearch, MinIO, Gotenberg, docling-serve and the imaging worker cannot open
connections at all. Temporal and GTC reach only what they use. The API, worker, release Job
and ClamAV keep outbound access for AI providers, email, integrations and signature updates.

`networkPolicy.apiFrom` narrows who may reach the API (by default, any pod in the cluster),
and `networkPolicy.metricsFrom` admits a Prometheus to port 9090 on the API and worker.

`managed.enabled` is for Trenova Cloud's operator, which turns it on: the API then serves a
usage API on port 8090 behind a generated token, reachable only from the peers in
`managed.from`. Leave it off on a self-hosted install.

## Upgrades

Read the release notes, then:

```bash
helm upgrade trenova oci://ghcr.io/emoss08/charts/trenova --version <new> \
  -n trenova -f trenova-values.yaml --wait --wait-for-jobs --timeout 30m
```

The new revision's Job migrates while the old API and worker keep serving; the new pods
become ready once the schema reaches their version, and the old ones are retired then.
`helm rollback` runs the Job of the revision it returns to, which never undoes a migration:
restore a backup to go back past one.

## Backups

The chart does not configure backups yet. What has to be kept:

- **PostgreSQL**: configure CloudNativePG's
  [Barman Cloud plugin](https://cloudnative-pg.io/plugin-barman-cloud/) for continuous WAL
  archiving and base backups to object storage, or take `pg_dump`s of `trenova` from the
  cluster's `-rw` service.
- **Object storage**: the bucket, from MinIO's volume or your S3 service's own replication.
- **`trenova-secrets`**, apart from both.

Meilisearch, Redis and Temporal's state rebuild from PostgreSQL: GTC backfills the indexes,
and in-flight AI runs are retried.

## Troubleshooting

| Symptom | Look at |
|---|---|
| The install times out | `kubectl -n trenova get pods,jobs,clusters`; the release Job's logs |
| The Job waits on PostgreSQL | `kubectl -n trenova describe cluster trenova-postgres`; the operator's logs in `cnpg-system` |
| `api` or `worker` stays in `wait-for-postgres` | The Job has not provisioned the application role yet, or failed: its `rls-roles` step |
| `failed to unmarshal config` | A `tms.config` key this release does not know |
| GTC waits for `gtc_publication` | The Job's `publication` step |
| Pods can't reach each other | A CNI that does not apply NetworkPolicy the way the chart expects; try `networkPolicy.enabled: false` to confirm |

## Uninstall

```bash
helm uninstall trenova -n trenova
```

This keeps the PostgreSQL cluster running (CloudNativePG deletes a cluster's volumes with it),
the persistent volumes of Redis, Meilisearch, MinIO and ClamAV, and the two generated Secrets,
so installing the release again picks up where it left off. To remove the data too, delete
the cluster and then the namespace:

```bash
kubectl -n trenova delete cluster trenova-postgres
kubectl delete namespace trenova
```
