# Operator

The Trenova Cloud operator (`services/operator`) runs one customer instance. It lives on the
instance's own single-node cluster, next to the Trenova Helm chart
([`deploy/helm/trenova`](../../deploy/helm/trenova)), and only ever calls out: it polls the control
plane for the instance's desired spec, runs the chart at the version and size the spec asks for,
and reports what is running. Nothing calls into it. The control plane's side is its decision
record `0008-instances` and the contract `api/instance.yaml`; this doc is the instance's side.

```
control plane (instance API, mutual TLS)
   ▲ GET /v1/spec (ETag)       │ PUT /v1/status every minute      │ POST /v1/certificate
   │                           ▼                                  ▼
operator ── values.Render(spec, base values) ──▶ Helm SDK: upgrade --install trenova@<version>
   │                                                    │
   └── health.Check: Deployments, StatefulSets, CNPG Cluster, the revision's release Job
```

## Packages

| Package | What |
|---|---|
| `cmd/operator` | configuration (environment), wiring, `/healthz` and `/livez` on :8081 |
| `internal/instanceapi` | the client generated from `api/instance.yaml`, the operator's copy of the control plane's contract |
| `internal/controlplane` | that client with problem details turned into errors: `ErrUnauthenticated` (never retry with that certificate), `RateLimitedError` (`Retry-After`), `ProblemError` |
| `internal/credential` | the enrolment bundle, the operator's own key and identity, the Secret store, the transport that swaps certificates, enrolment and renewal (`Manager`) |
| `internal/values` | the spec turned into chart values: hostname, lifecycle, components, sizing |
| `internal/release` | the chart from OCI at the pinned version, `upgrade --install`, recovery of an interrupted release |
| `internal/health` | each component's readiness, read from the cluster |
| `internal/reconcile` | the loop |
| `internal/fakecp`, `internal/controlplanetest`, `cmd/controlplane-stub` | a control plane for tests: in process (`httptest`) and in a cluster (CI) |

The module is not in `go.work`: Helm and client-go would raise the versions of dependencies the
TMS shares with it in workspace builds. Build and test it with `GOWORK=off`.

## Identity

Staff issue an enrolment bundle in the admin (instance ID, API URL, a 7-day certificate and its
key, the CA bundle) and install it as the Secret `trenova-operator-enrolment`
(`enrolment.json`). On first start the operator:

1. uses the bundle's certificate once, to renew onto an ECDSA P-256 key it generated itself;
2. writes its own key, certificate and CA bundle, and the fingerprint of the bundle it enrolled
   from, to the Secret `trenova-operator-identity`;
3. from then on uses only its own certificate. The control plane retires the bundle's on that
   certificate's first use.

It renews when a third of the certificate's life is left (30-day certificates renew after 20
days), writing the new key and certificate to the Secret before switching to them. Losing the
response, or failing to write the Secret, leaves the old certificate working; the next tick
renews again. The transport is swapped in place and its idle connections closed, so the next
request presents the new certificate and retires the old one.

**Re-enrolment.** Staff re-enrol an operator by issuing a new bundle, which revokes every
certificate the instance holds. When the control plane answers 401, the operator reads the
enrolment Secret: a bundle it has not enrolled from yet (by fingerprint) is used at once; the
same bundle means it needs a new one, so it says so, marks itself unhealthy and retries every
five minutes. A start that finds a newer bundle than the one its identity came from enrols from
it, rather than waiting for a 401.

## What it trusts

The operator holds admin rights over the instance's namespace, Secrets included, so what it
applies is checked twice before Helm sees it.

- **The spec.** It must name the instance the operator's certificate names; any other is
  reported and never applied.
- **The chart.** The release workflow signs each chart with Cosign, keyless: the signature names
  `release.yml` in this repository at a version tag, recorded in Sigstore's log. The bundle travels
  in the chart's own OCI provenance layer (`helm push` takes `<archive>.prov`), so it cannot be
  separated from the tag. `internal/release/provenance.go` refuses a chart with no signature, a
  signature from any other identity, or an archive other than the one signed, then refuses a signed
  chart whose name or version is not the one asked for, which stops a tag moved to an older signed
  release. The signer pattern lives there and in `release.yml`, which verifies with it before
  pushing. Trust comes from Sigstore's public root over TUF (`tuf-repo-cdn.sigstore.dev`), so the
  operator needs that egress when it pulls a version; a failed check leaves the deployed release
  running and reports the reason. `helm --verify` does not apply: the layer holds a Sigstore bundle,
  not a PGP signature. `chart.verify: false` turns the check off for a chart built in the cluster.
- **The images.** At publish, the workflow resolves every image the chart names, third-party ones
  included, into `image-digests.yaml` inside the chart and so inside the signature. The chart's image
  helpers add the digest for a `repository:tag` listed there; an override to another repository or
  tag finds no entry and is used as given.

The namespace the chart creates enforces the `baseline` Pod Security level, so even a workload the
operator was tricked into creating cannot reach the node. Label a namespace you create yourself
(`target.createNamespace: false`) the same way.

## The loop

Every 15 seconds (`TRENOVA_OPERATOR_POLL_INTERVAL`):

1. renew the certificate if it is due;
2. `GET /v1/spec` with the last `ETag`, so an unchanged spec costs a 304;
3. render the chart values; pull the chart at `release.version` (cached, two versions);
4. compare with the deployed release: apply only when the chart version or the values differ;
5. read readiness; report when something changed, and at least every minute
   (`TRENOVA_OPERATOR_REPORT_INTERVAL`, at most 1m).

The applied revision is the desired revision whose rendered state is the deployed one. It needs
no storage of its own: after a restart the operator renders the spec again and finds the release
already matches. A failed render or apply keeps reporting the revision it last applied, with the
reason as the message. A 429, or a 503 while the control plane has no version for the spec yet,
pauses the loop until `Retry-After`. An unknown `schemaVersion` is reported and never applied.

**Recovery.** A release left `pending-*` by a crash mid-upgrade is rolled back to its last
deployed revision before the next apply; one that never deployed is uninstalled and installed
again. A failed release with a deployed one before it is upgraded over.

## From spec to values

The base values (the operator chart's `trenovaValues`: the ingress, ClamAV, image overrides) go
first, and what the spec decides is merged over them:

| Spec | Values |
|---|---|
| `hostname` | `domain`; `storageDomain` is `<first label>-files.<rest>` |
| `instanceId` | `tms.instanceId` |
| `components` contains `docling` | `docling.enabled` |
| `lifecycle: active` | `stopped: false`, `postgres.hibernate: false` |
| `lifecycle: suspended` | `stopped: true`, `postgres.hibernate: true`: nothing runs, every volume is kept |
| `lifecycle: decommissioned` | `stopped: true`, `postgres.hibernate: false`: PostgreSQL stays up for the final backup |
| `release.version` | the chart version (and so every Trenova image tag) |
| `plan.vcpu`, `memoryGiB`, `diskGiB` | every component's requests and limits, Redis's `maxMemory`, PostgreSQL's memory settings and volume |

**Sizing** (`internal/values/size.go`). 2 GiB and 0.5 vCPU are left to k3s, CloudNativePG and the
operator; the rest is shared by weight among the components that run (PostgreSQL 28, API 12,
worker 12, imaging 10, ClamAV 10, docling 15, the rest 1 to 5), each with a floor (ClamAV
1.5 GiB, docling 2 GiB, PostgreSQL 1 GiB). Requests are the shares; memory limits are 1.5 times
them (PostgreSQL 1.25). The imaging worker gets a CPU limit of half the plan's vCPUs, at least
one, because its sandbox pool sizes itself by it. PostgreSQL's volume is 35% of the disk;
`shared_buffers` is a quarter of its memory limit and `effective_cache_size` three quarters. A
plan smaller than 2 vCPU, 8 GiB and 40 GiB is refused. The values test renders every catalog
plan and lifecycle through the real chart and its schema, and checks the requests fit.

Only PostgreSQL's volume follows the plan: a StatefulSet's volume claims cannot change after it
is created, so Redis, Meilisearch, MinIO and ClamAV keep the chart's sizes.

## Readiness

A component is ready when its Deployment or StatefulSet has every replica updated and available
(stopped: none wanted and none left), the CloudNativePG cluster is healthy with every instance
ready (hibernated: the hibernation condition is true), and the release Job of the deployed Helm
revision completed. The report is ready when the desired revision is applied and every component
is ready; otherwise its message names the first one that is not.

## Installing it

[`deploy/helm/trenova-operator`](../../deploy/helm/trenova-operator), after the CloudNativePG
operator, in its own namespace (`trenova-system`):

```bash
kubectl -n trenova-system create secret generic trenova-operator-enrolment \
  --from-file=enrolment.json=enrolment-<instance id>.json
helm install trenova-operator oci://ghcr.io/emoss08/charts/trenova-operator \
  -n trenova-system --create-namespace --version <version>
```

Its permissions are two namespaced Roles. In its own namespace it may read the enrolment Secret
and create and update its identity Secret. In the target namespace (`trenova`) it may manage
exactly the kinds the Trenova chart renders (Secrets, ConfigMaps, Services, Deployments,
StatefulSets, Jobs, NetworkPolicies, Ingresses, CloudNativePG Clusters and ImageCatalogs) and
read Pods, PVCs and Events. It has no cluster-wide permission: the namespace, which it keeps on
uninstall, is created by its chart.

## Tests

- `go test ./...` in `services/operator`, against the in-process fake control plane, a fake
  Kubernetes, and Helm's memory storage with the real chart: enrolment, renewal, re-enrolment,
  revocation, rate limits, every plan and lifecycle through the chart's schema, install and
  no-op detection, recovery, readiness, and the loop end to end.
- `.github/workflows/operator.yml` installs the operator on k3d against
  `cmd/controlplane-stub`, with every image built from the commit and the Trenova chart pushed
  to a registry in the cluster at the version the stub pins. It waits for a ready report on
  revision 1, suspends (revision 2: workloads at zero, PostgreSQL hibernated) and resumes
  (revision 3).
- The contract: the operator's `api/instance.yaml` is a copy of the control plane's, and the
  control plane's CI compares the two, so a contract change lands in both.

## Not in the operator yet

- **Usage.** Hourly meter readings (`POST /v1/usage`) and stopping metered actions need the TMS's
  managed mode, which counts them.
- **The final backup.** A decommissioned instance keeps PostgreSQL up, and staff take the final
  backup by hand, as `0008-instances` allows, until the operator takes backups.
- **Secrets from the control plane.** The Trenova chart generates the instance's secrets in the
  cluster; escrowing the encryption key with the control plane changes that.
