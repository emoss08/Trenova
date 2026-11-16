# Operator

The Trenova Cloud operator (`services/operator`) runs one customer instance. It lives on the
instance's own single-node cluster, next to the Trenova Helm chart
([`deploy/helm/trenova`](../../deploy/helm/trenova)), and only ever calls out: it polls the control
plane for the instance's desired spec, runs the chart at the version and size the spec asks for,
and reports what is running. Nothing calls into it. The control plane's side is its decision
record `0008-instances` and the contract `api/instance.yaml`; this doc is the instance's side.

```
control plane (instance API, mutual TLS)
   ▲ GET /v1/spec (ETag)    │ PUT /v1/status every minute   │ POST /v1/certificate   │ POST /v1/usage
   │ GET /v1/commands       │ GET /v1/email-events, POST /v1/email-events/ack (while emailEvents)
   │ POST /v1/telemetry every minute, beside the loop
   │                        ▼                               ▼                        ▼ every 15 minutes
operator ── values.Render(spec, base values) ──▶ Helm SDK: upgrade --install trenova@<version>
   │                                                    │
   ├── escrow.Ensure: the Secret trenova-escrow, from GET /v1/secrets when missing or incomplete
   ├── health.Check: Deployments, StatefulSets, CNPG Cluster, the revision's release Job
   ├── TMS managed API (:8090, bearer token): GET /managed/v1/readings, PUT /managed/v1/state,
   │                                           PUT /managed/v1/first-administrator/invite,
   │                                           PUT /managed/v1/features,
   │                                           GET|PUT|DELETE /managed/v1/system-email,
   │                                           POST /managed/v1/system-email/events,
   │                                           GET|PUT /managed/v1/log-requests,
   │                                           POST /managed/v1/support/log-pulls,
   │                                           GET /managed/v1/data-usage (telemetry, every 5 minutes)
   ├── the kubelet (:10250, ServiceAccount token, cluster CA): GET /stats/summary (telemetry)
   ├── Loki (:3100, no auth, the release's): label values and query_range, read only
   └── the documents bucket: PUT and HEAD support/logs/<bundleId>.zip
```

## Packages

| Package | What |
|---|---|
| `cmd/operator` | configuration (environment), wiring, `/healthz` and `/livez` on :8081 |
| `internal/instanceapi` | the client generated from `api/instance.yaml`, the operator's copy of the control plane's contract |
| `internal/controlplane` | that client with problem details turned into errors: `ErrUnauthenticated` (never retry with that certificate), `RateLimitedError` (`Retry-After`), `ProblemError` |
| `internal/credential` | the enrolment bundle, the operator's own key and identity, the Secret store, the transport that swaps certificates, enrolment and renewal (`Manager`) |
| `internal/values` | the spec turned into chart values: hostname, lifecycle, components, sizing, managed mode, the escrow Secret; the chart's names (`Fullname`, `AppSecret`, `APIService`, `EscrowSecret`) |
| `internal/escrow` | the escrowed secrets: the Secret in the target namespace, fetched from the control plane when it is missing or incomplete, and the shapes the TMS accepts |
| `internal/release` | the chart from OCI at the pinned version, `upgrade --install`, recovery of an interrupted release |
| `internal/health` | each component's readiness, read from the cluster |
| `internal/tms` | the client of the TMS's managed API, and the token read from the release's app Secret |
| `internal/usage` | the usage relay: readings from the TMS to the control plane, its answer back |
| `internal/telemetry` | what the instance uses, sampled every minute (the sampler) and sent beside the loop (the relay) |
| `internal/storage` | the instance's buckets: the documents and backups Secrets, fetched from `GET /v1/storage` when missing or incomplete, and the documents bucket's CORS rules for the instance's origin |
| `internal/backup` | backups read from and asked of CloudNativePG: health for the status report, the base backup before an update, the final backup, and a restore's progress |
| `internal/systememail` | the system email sender: the TMS brought to the spec's fingerprint, the credential fetched from `GET /v1/secrets` and kept only in memory |
| `internal/emailevents` | the system email events: bounces and complaints the control plane keeps for the instance, relayed to the TMS and acknowledged once it handled them |
| `internal/supportkeys` | the support access key set: the TMS brought to the spec's `support` when its instance id, staff domain or key ids differ |
| `internal/commands` | what the control plane asks besides the spec: restarting a component, collecting its log lines, collecting a log bundle; the support access grant checked first for staff |
| `internal/redact` | every value of every Secret in the release namespace, and anything shaped like a credential, removed from a log line |
| `internal/logs` | log keeping: Loki read a page at a time, the last lines, log bundles (a zip in scratch space, uploaded to the documents bucket) on one background worker, and the log requests made in Trenova |
| `internal/selfupgrade` | the operator's own version: its HelmChart moved to the spec's release, the move watched and undone when it fails |
| `internal/reconcile` | the loop |
| `internal/fakecp`, `internal/controlplanetest`, `cmd/controlplane-stub` | a control plane for tests: in process (`httptest`) and in a cluster (CI) |
| `internal/tmstest` | the TMS's managed API for tests |

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
3. bring the operator itself to `release.version` ([Its own version](#its-own-version)); while it
   is not there, apply nothing and report why;
4. render the chart values; make sure the escrow Secret is there (below), and apply nothing until
   it is; pull the chart at `release.version` (cached, two versions);
5. compare with the deployed release: apply only when the chart version or the values differ;
6. read readiness; while the instance is active and ready and the spec has an invite, install it
   in the TMS (below), put the spec's features to the TMS (below), bring the TMS's system
   email sender and support access keys to the spec, and read the features and live support
   grants the TMS holds into the report (below);
7. report when something changed, and at least every minute
   (`TRENOVA_OPERATOR_REPORT_INTERVAL`, at most 1m);
8. relay usage when it is due (below), then, while the instance is active and ready, take up the
   log requests made in Trenova ([Logs](#logs)), within 30 seconds;
9. carry out the control plane's commands ([Commands](#commands)). This runs on every tick whose
   spec read succeeded, whatever happened in between: a failed apply or report is exactly when
   staff need a component's logs or a restart. It waits only while the control plane asked the
   operator to wait (a 429 or 503), and says so once;
10. when the commands answer said `emailEvents` and the instance is active and ready, relay the
    system email events ([System email](#system-email)).

The loop's dependencies are all required: `reconcile.New` refuses a missing one by name, and the
commands runner refuses one without its control plane. Nothing is checked by type assertion at
run time. (Before this, the runner reached the control plane by asserting the loop's client had
`Commands`; the client `cmd/operator` built did not, so the operator never asked for a command
and said nothing: a staff request for logs waited 15 minutes and expired.)

The applied revision is the desired revision whose rendered state is the deployed one. It needs
no storage of its own: after a restart the operator renders the spec again and finds the release
already matches. A failed render or apply keeps reporting the revision it last applied, with the
reason as the message. A 429, or a 503 while the control plane has no version for the spec yet,
pauses the loop until `Retry-After`. An unknown `schemaVersion` is reported and never applied.

**Recovery.** A release left `pending-*` by a crash mid-upgrade is rolled back to its last
deployed revision before the next apply; one that never deployed is uninstalled and installed
again. A failed release with a deployed one before it is upgraded over.

A first install can fail partway (the database operator's webhook is not up yet) after some of
its pods have started, so removing it has to leave nothing for the next one to trip over. The
operator uninstalls it and waits until its resources are gone (foreground deletion), then deletes
the volume claims labelled for the release: the ones a StatefulSet made are not the chart's, so an
uninstall leaves them, half-written. Only then does it forget the release and install again.
Helm's record of the release is kept until that is done (`KeepHistory`), so a cleanup that timed
out or was interrupted is finished on the next tick and not skipped. Without the wait, the new
install adopted a claim that was still being deleted and its pod stayed pending for good; without
the volume step, Meilisearch refused a data directory it had not finished creating. This is only
ever done to a release that never deployed. One with a deployed revision in its history keeps its
volumes whatever state it is in, and one uninstalled by hand is not brought back.

## From spec to values

The base values (the operator chart's `trenovaValues`: the ingress, ClamAV, image overrides) go
first, and what the spec decides is merged over them:

| Spec | Values |
|---|---|
| `hostname` | `domain`; `storageDomain` is `<first label>-files.<rest>` |
| `instanceId` | `managed.instanceId`, which the chart passes to the TMS as `managed.instanceId`: the tag on the email Trenova's sender carries ([System email](#system-email)) |
| `components` contains `docling` | `docling.enabled`; its resources are rendered either way (see Sizing) |
| `components` contains `embedding` | `embedding.enabled` (Trenova AI's search service); its resources are rendered either way, and the chart passes its URL and model to the TMS as `managed.embedding` ([managed-mode.md](managed-mode.md#search-by-meaning)) |
| (always) | `managed.enabled: true`, `managed.from`: the operator's own namespace |
| (always) | `secrets.escrowSecret: trenova-escrow` |
| (always) | `logs.enabled: true`, whatever the base values say: log keeping ([Logs](#logs)) |
| (always) | `tms.config.ai.privateNetworkProviders: false`, `serverCloudCredentials: false`: a customer's administrator adds the instance's AI providers, so none may sit on a private address or sign in as the instance's own cloud identity, whatever the base values say |
| `firstAdministrator` | `bootstrap.orgName`, `adminName`, `adminEmail`, and `bootstrap.adminSetup: link` |
| `lifecycle: active` | `stopped: false`, `postgres.hibernate: false` |
| `lifecycle: suspended` | `stopped: true`, `postgres.hibernate: true`: nothing runs, every volume is kept |
| `lifecycle: decommissioned` | `stopped: true`, `postgres.hibernate: false`: PostgreSQL stays up for the final backup |
| (storage from `GET /v1/storage`) | `minio.enabled: false`; `storage.bucket`, `region`, `autoCreateBucket: false`, `storage.external` (`endpoint`, `useSSL`, `publicEndpoint`, `existingSecret: trenova-storage`); `backup.enabled`, `destinationPath`, `endpointURL`, `existingSecret: trenova-backup-storage`. Without storage (a `404`), none of these: MinIO and no backups |
| `database` | `postgres.generation`, `postgres.restoredFrom` (`sourceGeneration`, `targetTime`): the instance runs on the database a restore made |
| `restore` | `postgres.restore` (`generation`, `sourceGeneration`, `targetTime`): a database restored beside the live one; left out once `database` names it, refused without storage |
| `release.version` | the chart version (and so every Trenova image tag) |
| `plan.vcpu`, `memoryGiB`, `diskGiB` | every component's requests and limits, Redis's `maxMemory`, PostgreSQL's memory settings and volume |

**Sizing** (`internal/values/size.go`). 2 GiB and 0.6 vCPU are left to k3s, CloudNativePG, the
operator and the release Job. ClamAV's memory does not grow with the plan (it is the signature database: about
1.2 GiB, twice that while clamd reloads it each day), so it takes a fixed 3 GiB request and
4 GiB limit, ClamAV's documented minimum and preferred container size, off the top. The
embedding service is fixed too: it requests 320 MiB (what it holds with its model loaded) and
1 GiB is taken off the top for it (`reserveMiB`, its memory limit), so the room between its
request and its limit is never shared out to another workload; its CPU limit is 2, the threads
its model is built for, and it shares the CPU by a weight of 2. The rest
is shared by weight among every component the chart can run (PostgreSQL 28, API 12, worker 12,
imaging 10, docling 15, Loki 2, Alloy 0.5 of the memory and 1 of the CPU, the rest 1 to 5;
ClamAV 4 of the CPU), each with a floor (docling 2 GiB, PostgreSQL 1 GiB, Loki 256 MiB, Alloy
128 MiB). Requests are the shares; memory limits are 1.5 times them (PostgreSQL 1.25), and at
least 1 GiB for Loki (a week-long query) and 512 MiB for Alloy. The imaging worker gets a CPU limit of half the plan's vCPUs, at least
one, because its sandbox pool sizes itself by it. PostgreSQL's volume is 35% of the disk and
Loki's 10% (at least 4 GiB);
`shared_buffers` is a quarter of its memory limit and `effective_cache_size` three quarters. A
plan smaller than 2 vCPU, 10 GiB (13 GiB with ClamAV) and 40 GiB is refused: below that, every
component together and the room a rollout needs no longer fit. (Log keeping moved it from 9 and
12 GiB: at 9 GiB Loki's and Alloy's floors left the largest rollout pod about 90 MiB short of
the node's allocatable memory.) The values test renders every
catalog plan and lifecycle through the real chart and its schema, and checks the requests fit.

**Room for a rollout.** A Helm upgrade applies every changed Deployment at once, and a Deployment
that rolls starts its new pod before the old one stops: the API and the worker set
`maxUnavailable: 0, maxSurge: 1` because they must keep serving, and the client, the imaging
worker and Gotenberg keep Kubernetes's default (a 25% surge, one pod at one replica): the client
serves every page, and the API calls the other two on its request paths. Kubernetes creates a
second pod of all five at the same time, while the old ones still hold their requests. Nothing
else needs a second pod: Temporal, gtc, docling, the embedding service, Loki and Alloy are `Recreate` (searches fall back to keywords while the embedding service restarts; Loki owns one
`ReadWriteOnce` volume, and two Alloys would push every line twice; docling's only caller is the
worker's extraction activities, which retry and resubmit a lost task, so it stops before its
replacement loads its models), a StatefulSet replaces a pod after the old one is gone,
CloudNativePG restarts its one instance in place, and the release Job's pod is already counted
with the workloads. The sizing holds room for the largest one of those five (`surge` in `shares`:
the largest weight among them is added to the division once, for memory and for CPU), not for all
five at once. That is enough: a new pod that does not fit waits `Pending`; each rollout that
finishes stops its old pod and gives back exactly what its new pod took, so the smaller ones
finish first if they were scheduled first and the largest always finds the held room free in the
end. Holding room for all five together would cost PostgreSQL about 28% of its share at every
plan size; holding the largest costs about 10% and makes an upgrade a little slower on a machine
sized exactly to its plan. Before this, every share was requested and such a machine kept about
1 GiB free at Sandbox while one new API pod asked for 1,325 MiB: the pod stayed `Pending`, the
old pods kept serving, the rollout never finished and the
instance never reported ready. What is held depends only on the machine and the full set of
components, like every share, so turning docling on still changes nothing else.
`TestEveryMachineFitsItsWorkloadsAndARollout` renders the chart at every plan size, both minimums
and the local cap, with and without storage of ours and with ClamAV on and off, reads each
Deployment's strategy from the manifests, and checks that every request plus the largest surge
pod fits the node's allocatable memory and CPU; it fails too when the chart's rolling Deployments and the
`surge` flags disagree, so a workload that changes its strategy changes its flag in the same
change.

**A component's share never depends on which other components are enabled.** It depends only on
the machine (the plan, or the node when that is smaller) and on the installation's own
configuration (ClamAV, which the operator chart's `trenovaValues` turns on for every instance on
that cluster). Docling (the spec's `components: [docling]`, which Document Intelligence brings),
the embedding service (`components: [embedding]`, which Trenova AI brings) and MinIO (on only
when the instance has no storage of ours) are counted whether they run or not,
and their resources are rendered either way. Turning one on or off therefore changes
`docling.enabled` (or `minio.enabled`) and nothing else in the sizing: PostgreSQL's resources and
parameters, and every running workload's resources, stay as they are, and CloudNativePG has
nothing to restart. Before this, a purchase that turned docling on divided the machine again,
changed `shared_buffers`, and restarted PostgreSQL under a running instance. Only a different
machine (a plan change) divides it again. `TestTurningAComponentOnLeavesEveryOtherWorkloadAsItWas`
holds the rule for every plan, with and without storage, and under the local node cap; keep it
when a component is added: an optional component gets a share in `shares`, always counted.

The cost is the share an instance does not use. A plan without Document Intelligence leaves
docling's share idle, a plan without Trenova AI leaves the embedding service's 1 GiB idle, an
instance with storage of ours leaves MinIO's, and the room for a rollout is idle between upgrades
(ClamAV on at the plan sizes). The table below predates the embedding service's 1 GiB and its CPU
weight of 2; every figure in it is now somewhat lower and is to be read again from
`TestEveryMachineFitsItsWorkloadsAndARollout`'s rendering:

| Machine | PostgreSQL (request, limit, `shared_buffers`) | API and worker (each) | Docling's share (idle on the base plan) | MinIO's share (idle with storage of ours) | Held for a rollout |
|---|---|---|---|---|---|
| 2 vCPU, 10 GiB, no ClamAV (the local cap) | 1,968 MiB, 2,461 MiB, 615 MB | 843 MiB, 184m | 2,048 MiB, 98m | 256 MiB, 36m | 843 MiB, 184m |
| 4 vCPU, 16 GiB (Sandbox) | 2,707 MiB, 3,384 MiB, 846 MB | 1,160 MiB, 432m | 2,048 MiB, 230m | 290 MiB, 86m | 1,160 MiB, 432m |
| 8 vCPU, 32 GiB (Production) | 6,645 MiB, 8,306 MiB, 2,076 MB | 2,847 MiB, 940m | 3,559 MiB, 501m | 711 MiB, 188m | 2,847 MiB, 940m |
| 16 vCPU, 64 GiB (Production Plus) | 14,520 MiB, 18,150 MiB, 4,537 MB | 6,223 MiB, 1,957m | 7,778 MiB, 1,044m | 1,555 MiB, 391m | 6,223 MiB, 1,957m |

Log keeping costs, at the same sizes (request, memory limit, CPU request; Loki's volume):

| Machine | Loki | Alloy | Loki's volume |
|---|---|---|---|
| the local cap | 256 MiB, 1,024 MiB, 24m | 128 MiB, 512 MiB, 12m | 4 GiB |
| Sandbox | 256 MiB, 1,024 MiB, 57m | 128 MiB, 512 MiB, 28m | 5 GiB |
| Production | 474 MiB, 1,024 MiB, 125m | 128 MiB, 512 MiB, 62m | 20 GiB |
| Production Plus | 1,037 MiB, 1,555 MiB, 261m | 259 MiB, 512 MiB, 130m | 50 GiB |

Idle requests are not wasted memory on the node: nothing else is scheduled there, so the page
cache and every limit above its request can still use it. They are room the scheduler keeps free.

Only PostgreSQL's volume follows the plan: a StatefulSet's volume claims cannot change after it
is created, so Redis, Meilisearch, MinIO and ClamAV keep the chart's sizes. Loki's claim is its
own object and takes the plan's size when it is made, then keeps it (the chart reads the live
claim back): a claim can only grow, and k3s's `local-path` class cannot grow one, so following a
plan change would fail the upgrade.

**When PostgreSQL restarts** (a plan change, a new PostgreSQL image, a changed parameter,
hibernation), CloudNativePG first waits for clients to disconnect, then asks for a fast shutdown.
The API, the worker, Temporal and gtc hold pooled connections that never leave on their own, so
the chart sets `smartShutdownTimeout: 15` (CloudNativePG's default, 180, held a restart for three
minutes with the new API and worker pods waiting in `Init`) and `stopDelay: 300`, the pod's
termination grace period, which bounds the fast shutdown and its checkpoint. A fast shutdown is a
clean one: open sessions end, every committed transaction is already in the WAL and a shutdown
checkpoint is written; the clients reconnect when the database is back. `switchoverDelay` keeps
CloudNativePG's default: it applies only with replicas, and cutting it short could demote a
primary before its replicas have every WAL record. A base backup still running 15 seconds into a
stop fails and is reported (a pre-update backup as `failed`; the final backup is taken again);
the pre-update backup completes before the control plane writes the version it guards, so the two
do not overlap in the normal order. Nothing else in the chart or the operator waits on the long
default: the release Job and the new API and worker pods wait for PostgreSQL to come back, not
for it to stop.

**A machine smaller than the plan.** The operator chart's `node` values (`vcpu`, `memoryGiB`,
`diskGiB`; zero by default) say what the cluster really runs on. Set, the operator sizes the
workloads for the smaller of the plan and the node in each dimension (`values.Capacity`), and
the spec itself is untouched. The control plane's development-only local provider sets 2 vCPU,
10 GiB and 40 GiB, with ClamAV off (the smallest machine that fits, since log keeping), so every plan bought for "This computer" runs the same small
instance on a laptop while the plan, its features and its allowances stay the real ones.

## Escrowed secrets

Four of the TMS's secrets make its data readable: the encryption key
(`security.encryption.key`), the row-level security scope key and its ID
(`database.rls.scopeKey`, `scopeKeyId`) and the MFA recovery code pepper
(`security.mfa.recoveryCodePepper`). A backup restored without them is unreadable, so the control
plane keeps them in escrow and hands the same values out on every `GET /v1/secrets`. A Cloud
instance never lets the chart generate them: a reinstall, or a restore onto new infrastructure,
would get new keys and lose the data.

**Where.** The Secret `trenova-escrow` (`values.EscrowSecret`) in the target namespace, keys
`encryptionKey`, `rlsScopeKeyId`, `rlsScopeKey` and `mfaRecoveryCodePepper`, labelled
`app.kubernetes.io/managed-by: trenova-operator` and `app.kubernetes.io/component: escrow`. It is
not a Helm resource, so uninstalling the release leaves it. The operator renders
`secrets.escrowSecret: trenova-escrow`; the chart then reads those four env vars from it and leaves
them out of its app Secret.

**When.** Every tick, after rendering and before anything is applied, the operator reads the
Secret. A complete one whose values have the shapes below is used as it is and the control plane
is not asked: its values are fixed per instance, so a present Secret came from it. A missing one is
fetched and created; an incomplete or malformed one is fetched and its four keys rewritten. The
values are checked before they are written: the encryption key and the pepper 32 to 128
characters, the scope key ID 1 to 32 of `[A-Za-z0-9_-]`, the scope key standard base64 of at least
32 bytes. Values of any other shape are refused, and the report names the key and the rule, never
the value. Until the Secret is in place nothing is installed or upgraded and the report says why;
a 429 waits out `Retry-After` and a 401 goes to credential recovery, as for the spec. The values
are never logged. The same answer may carry the system email credential (`systemEmail`); the escrow
Secret never holds it ([System email](#system-email)).

## The first administrator

The spec's `firstAdministrator` names the person who ordered the instance and their
organization; the first install creates both with no usable password (`bootstrap.adminSetup:
link`, [managed-mode.md](managed-mode.md#the-first-administrator)). A spec that is missing a name,
has one over 100 characters or a control character, or a malformed email, is refused like any
other invalid spec.

Once the instance runs, the spec carries `firstAdministrator.invite`: the SHA-256 of a token the
control plane holds, and its expiry. While the instance is active and ready, the operator puts it
to the TMS (`PUT /managed/v1/first-administrator/invite`, the managed token read again on a 401)
and reports the answer as `administrator` on the status report: the hash the TMS holds, if it is
still live, and whether the administrator is set up. It asks again at most every minute, or at
once for a new invite, and reports at once when the answer changes, which is how the control plane
learns to email the link and later that the administrator chose a password. A failure is logged,
leaves `administrator` off the report and is retried on the next tick; it never holds converging
or the report back. The token itself never reaches the operator.

## The features an instance may use

The spec's `features` are the TMS feature keys the customer bought. While the instance is active
and ready, the operator puts them to the TMS (`PUT /managed/v1/features`) on every tick. The list
is the complete set and sending it replaces what the TMS holds, so a changed list, or a TMS that
was restarted or restored, is caught up on the next tick. The operator sends the list as the spec
has it: the TMS ignores a key it does not know. A failure is logged and retried on the next tick;
it never holds converging or the report back. What the TMS does with the list is in
[managed-mode.md](managed-mode.md).

The same request carries the spec's `addonEndings` as `addonEndings` (always, an empty list when
the spec has none, so the TMS's list is replaced on every tick) and the spec's `account.planUrl`
as `planUrl` (the control plane adds `account` to every spec it serves rather than storing it in
a revision, and the spec's `ETag` changes with it, so a console that moves is fetched like a new
revision; a spec without one sends none, and the TMS keeps what it holds)
(`reconcile.featureList`, `tms.Client.PutFeatures`). It also always states the plan's end as
`planEndsAt`: the spec's `account.endsAt` in UTC RFC 3339 while the plan is set to end, and `null`
when it is not, so a kept plan clears what the TMS holds on the next tick. The control plane
composes the end when it serves the spec and changes the `ETag` with it, so it arrives with no new
revision; the operator needs no comparison of its own, since the list goes out every tick and the
TMS stores and announces an end only when it differs from what it holds.

## System email

Trenova supplies each managed instance a sender for its system email (invitations, password
resets, notifications; [managed-mode.md](managed-mode.md#system-email) has what the TMS does
with it and the exact contract). The spec names it by fingerprint only, `systemEmail:
{"fingerprint": "<hex>"}`; the credential is escrowed with the other secrets and comes from
`GET /v1/secrets` (`systemEmail`). While the instance is active and ready, every tick
(`internal/systememail.Relay.Sync`):

1. `GET /managed/v1/system-email`;
2. with no `systemEmail` in the spec, `DELETE` it when the TMS holds one; otherwise stop when the
   TMS holds the spec's fingerprint;
3. take the credential from memory when it has the spec's fingerprint, otherwise from
   `GET /v1/secrets`, checking its fingerprint against the spec's; then
   `PUT /managed/v1/system-email`.

The credential is never written to a Secret, a log line or the status report; a TMS that was
restarted or restored is caught up from memory without asking the control plane again. A rotation
is a new fingerprint in the spec, so the next tick fetches and puts the new credential. A failure
is logged and retried on the next tick and never holds converging or the report back. Secrets
whose fingerprint does not match the spec (a rotation half published), or a 429, wait a minute
(or `Retry-After`) before the control plane is asked again; a 401 goes to credential recovery.

**Bounces and complaints.** Trenova's provider account has one webhook for every instance, so it
posts to the control plane, which keeps each bounce or spam complaint for the instance the
message's `trenova_instance` tag names (the TMS tags every message it sends through Trenova's
sender with its `managed.instanceId`). The commands answer carries `emailEvents`; when it is true
and the instance is active and ready, after the commands (`internal/emailevents.Relay.Run`, within
30 seconds):

1. `GET /v1/email-events`: at most 50, oldest first; the operator relays one batch per tick;
2. `POST /managed/v1/system-email/events` with each event's `id`, `provider` and `payload`;
3. `POST /v1/email-events/ack` with exactly the IDs the TMS answered as `handled`, and nothing
   when it handled none.

A TMS that fails acknowledges nothing, and an event the TMS left out stays queued; both are
relayed again on the next tick, which the TMS takes without applying anything twice. A failure is
logged with counts and event IDs, never a payload (it names the customer's recipients), and never
holds back the rest of the tick or the status report; a 429 waits out `Retry-After` and a 401
goes to credential recovery. A suspended, decommissioned or unready instance relays nothing; the
control plane deletes an event left unacknowledged for 7 days.

## Support access

Trenova staff enter an instance with an assertion the control plane signs, and only under a
grant one of its organizations made ([managed-mode.md](managed-mode.md#support-access)). The
operator carries two things for it.

**The keys, in.** The spec's `support` (`instanceId`, `staffDomain`, `keys`, one to four Ed25519
public JWKs) is composed by the control plane when it serves the spec, so a rotation changes the
`ETag` with no new revision. While the instance is active and ready and the spec carries
`support`, every tick (`internal/supportkeys.Relay.Sync`, within 30 seconds):

1. `GET /managed/v1/support-access`, which answers the instance id, domain and key ids it holds,
   never a key;
2. stop when all three match (the key ids as a set, in any order);
3. otherwise `PUT /managed/v1/support-access` with exactly `{instanceId, staffDomain, keys}`: the
   TMS refuses an unknown field and an empty set, and the set it gets replaces the one it held.

A spec without `support` sends nothing, and the TMS keeps what it holds. Logs and errors name the
key ids, never a key. A failure is logged, put in the report's `message` when nothing else is
there (`support access keys not delivered: …`) and retried on the next tick; it never holds
converging, the feature list, the report or the usage relay back.

**The grants, out.** While building each status report (every tick, while the instance is active
and ready, within 30 seconds, like the held features), the operator reads
`GET /managed/v1/support-access/grants` and puts the list in `supportGrants`: at most 100, each
grant once, organization names cut to 200 characters and staff addresses lower-cased. A grant the
contract cannot carry (an unknown scope, an organization id not shaped `org_` and 26 characters,
an empty name) is logged and left out, since one bad item would make the control plane refuse the
whole report. A changed list is reported at once. When the TMS is not up or the read
fails, the field is left out and the control plane keeps the last list it was sent. The list says
who each grant is for, in which organization, with what scope and until when; never who made it.

## Usage relay

The TMS counts metered actions (its app meters: `shipments.total`, `document_intelligence.pages`,
`documents.storage_bytes` and `organizations.total`, listed with their units and cap semantics in
[managed-mode.md](managed-mode.md#what-the-control-plane-must-send)) and the control plane bills
them against the subscription's allowances. The operator carries the readings one way and the
answer the other (`internal/usage`), through the TMS's managed API: a listener on the API pods'
port 8090 that the chart serves only with `managed.enabled`. The chart generates its bearer
token into the app Secret (`managedToken` in `<fullname>-secrets`, or `secrets.existingSecret`),
and its NetworkPolicy admits port 8090 from `managed.from` alone, which the operator sets to its
own namespace. The operator reads the token from that Secret in the target namespace, and reads
it again whenever the TMS answers 401, so a token the chart has just created or replaced is
picked up on the next call.

**When.** After the tick's converge and report, while the spec's lifecycle is `active` and that
converge reported ready (the desired revision deployed, every component ready): the first time
as soon as that holds, then every 15 minutes (`TRENOVA_OPERATOR_USAGE_INTERVAL`, at most 15m,
because the control plane bills a window 30 minutes after it ends). A suspended or
decommissioned instance, or one that is not ready, relays nothing; resuming relays at once.

**A run.**

1. `GET /managed/v1/readings?since=<current hour − 2h>`: every counted meter's hours from then
   to the current hour (zeros included) and storage's level for the current hour. Two hours
   cover the control plane's 30-minute close grace, and resending an hour replaces it, so a
   missed run is made good by the next.
2. Map each app meter to its catalog meter through the current spec's allowances (`appMeter`);
   a reading with no allowance is dropped. An allowance without `appMeter` is an instance
   counter the TMS does not count. The mapping is one to one; the first allowance wins.
3. `POST /v1/usage`, at most 1,000 readings per request (an empty report still gets an answer).
4. Map the last answer's meters back to app meters, dropping any without one, and
   `PUT /managed/v1/state` with the window (none when the subscription has no open window, which
   stops every metered action), whether overage is billed, and each meter's quantity, allowance,
   hard cap and `stopped`.

**Failures** never hold up converging or status reports: a run happens after both, within a
minute's deadline, and is only logged. Any failure retries after a minute; a 429 waits out its
`Retry-After` (at least the minute); a 401 from the control plane goes to the same credential
recovery as the rest of the loop. A missing token (the release not deployed with managed mode
yet), a TMS that refuses the token after it was read again, or an unreachable TMS are logged
and retried the same way. The TMS keeps the last state it was sent until the next.

## Telemetry

Once a minute the operator samples what the instance is using and sends it to the control plane
(`POST /v1/telemetry`, the contract's `TelemetrySample`), which the console and the staff admin
draw and attention rules read. Nothing in it bills or entitles. It is always on, with no setting.

**Where it runs.** `internal/telemetry.Relay.Run` runs beside the loop, started by `Loop.Run`
once the credentials are up (a `sync.WaitGroup` that `Loop.Run` waits on as it returns, so a lost
lease or a stop ends both). It never runs on the loop's tick, so a slow or failing control plane,
metrics API or kubelet never holds back the spec apply, the status report or a command. The loop's
only touch is `Observe`: each status report's health parameters (Helm revision, stopped,
hibernated, database generation), stored atomically, which the next sample reads. Before the loop
has observed a release, a sample carries an empty `parts` list.

**When.** It sleeps until the next minute starts (UTC), samples, stamps the sample with that
minute (zero seconds), then sends. The wait is computed from the clock each time, so samples never
share a minute; a minute the operator was busy for is skipped, not stamped twice.

**What is read, and from where.** Every read has 5 seconds (the stats summary 10); a read that
fails leaves its fields out (never zero) and logs one warning for that sample, so at most once a
minute per kind.

| Figure | Source | When the source is down |
|---|---|---|
| `parts[]`: name, `ready`, `replicas`, `readyReplicas` | `health.Checker.Parts`, the same reading the status report's `components` come from (Deployments, StatefulSets, the CloudNativePG `Cluster`, the revision's release Job), so the names are exactly the report's | `parts` is empty |
| `parts[].restarts` | the restart counts of every container (init containers included) of the part's pods, summed over its copies | left out (the pod list failed) |
| `parts[].cpuLimitMillicores`, `memoryLimitBytes` | the limits of the containers of the part's running pods, summed; left out when any container has none, or no pod runs | left out |
| `parts[].cpuMillicores`, `memoryBytes` | `metrics.k8s.io` `PodMetrics` in the target namespace, summed over the part's running pods; 0 when none runs | left out, also when one running pod has no metrics yet |
| `parts[].version` | the tag of the workload's first container image (the Cluster's `status.image`), without registry, name or digest | left out when the image has no tag |
| `machine.cpuCapacityMillicores`, `memoryCapacityBytes` | the Node's `status.capacity` | left out |
| `machine.cpuMillicores`, `memoryBytes` | `metrics.k8s.io` `NodeMetrics` (memory is the working set) | left out |
| `machine.diskBytes`, `diskCapacityBytes` | the kubelet's stats summary, `node.fs` (the kubelet's filesystem, which on a single-node k3s machine also holds `local-path` volumes), read from the kubelet itself (below) | left out: the node has no InternalIP, the certificate does not verify, the kubelet refuses or the answer is over 8 MiB |
| `data.databaseBytes`, `documentsBytes`, `searchBytes` | Trenova itself: `GET /managed/v1/data-usage` on the TMS's managed API (`tms.Client.DataUsage`, the managed token as for every TMS call), at most every 5 minutes with 10 seconds, never on the loop's tick; each figure is absent when Trenova could not measure it | the last good answer is used for up to 30 minutes, then the figures are left out; a release without the endpoint (404) leaves them out and says so at most once an hour |
| `data.logsBytes` | the stats summary's volume statistics: used bytes of the volume claims mounted by the pod labelled `loki` | left out when the summary fails or the volume reports no usage (k3s's `local-path` volumes report none) |

The node is the one most of the release's running pods are on, or the operator's own
(`TRENOVA_OPERATOR_NODE_NAME`, from the downward API) when the instance runs none (suspended).
The data sizes come from Trenova and not from volumes because the kubelet reports no usage for
k3s's `local-path` volumes, and the operator holds no database connection of its own.

**The kubelet.** The stats summary is read from the kubelet directly, never through the API
server's node proxy (`nodes/proxy` would reach the whole kubelet API, exec included):
`GET https://<node InternalIP>:<status.daemonEndpoints.kubeletEndpoint.Port, 10250 when unset>/stats/summary`
with the operator's ServiceAccount token (the projected file, read again on every call since it
rotates), the certificate verified against the cluster's CA bundle the pod mounts (k3s signs the
kubelet's serving certificate with it, the node's IP in its names). Verification is never
skipped: a certificate that does not verify sends nothing, the token included, and leaves disk
out. The kubelet authorizes the call as `nodes/stats: get`. The answer is read up to 8 MiB
within 10 seconds. The operator chart has no NetworkPolicy, so nothing limits the operator's way
to the kubelet's port; a cluster that adds one must allow it.

**Sending.** Unsent samples wait in memory in a ring of at most 120 (two hours); past that the
oldest is dropped. After each sample the relay sends up to 30, oldest first, within 15 seconds; a
204 drops them, a network failure or 5xx keeps them for the next minute, so an outage is caught up 30 a minute
and a long one loses its oldest minutes. A 400 can never succeed: that batch is dropped and logged
once at error level with the control plane's problem detail, and the next batch goes on. A sample for a minute already kept replaces it, one older
than the newest kept is dropped, and samples older than the control plane accepts (2 hours, less
a minute) are dropped before sending, so a batch never carries a minute twice and is never refused
for one stale sample. A 429 waits out `Retry-After`; a 401 is left to the loop's credential
recovery. A restart loses what was unsent.

**Permissions.** In the target namespace, `metrics.k8s.io` `pods: list` (the release Role).
Cluster-wide, the ClusterRole `<release>-telemetry`: `nodes: get`, `nodes/stats: get` and
`metrics.k8s.io` `nodes: get`, bound to the operator's ServiceAccount; the only cluster-wide grant
the operator holds. Never `nodes/proxy`: the chart test fails if any role grants it.

## Storage and backups

**The buckets.** A managed instance keeps documents and backups in two buckets of its own, made by
the control plane's provider (its `0021-backups` decision). The operator fetches them with `GET
/v1/storage` when the cluster holds no copy and keeps them in two Secrets in the target namespace,
labelled `app.kubernetes.io/managed-by: trenova-operator`, never Helm resources:

| Secret | Keys | Read by |
|---|---|---|
| `trenova-storage` | `accessKey`, `secretKey`, `endpoint`, `region`, `bucket` | the TMS (`storage.external.existingSecret`) |
| `trenova-backup-storage` | `ACCESS_KEY_ID`, `ACCESS_SECRET_KEY`, `ACCESS_REGION`, `destinationPath`, `endpoint` | the Barman Cloud plugin beside PostgreSQL, through the chart's `ObjectStore`; no application pod |

Each tick, before anything is applied, the operator reads both. Complete and well formed, they are
used as they are. Missing or incomplete, both are fetched and written; values that are not an
`https` endpoint, DNS bucket names, two different buckets and two different credentials are
refused and nothing is applied, as for the escrowed secrets. A `404` means the instance has no
storage of ours (its infrastructure was made by hand): the chart keeps MinIO and takes no backups,
and the operator asks again ten minutes later, or at once when a new spec revision arrives. The
control plane seals a new instance's storage when it records the provision, minutes after the
operator first looked, and the revision that follows is the operator's cue: before it looked
again only after ten minutes, and an instance whose storage arrived inside that window ran on
MinIO without backups until the operator restarted. The spec does not say whether the instance
has storage; the answer to `GET /v1/storage` is the only signal. The credentials are never logged.

**A private endpoint.** The operator chart's `storage.privateEndpoint` (empty by default) is
where the operator and the release's pods reach the buckets when that is not the address the
control plane gives, which browsers keep using: `storage.external.endpoint` and `useSSL` (from the
scheme, so `http://` is allowed here and only here), `backup.endpointURL` and the CORS call use
it; `publicEndpoint` stays the control plane's `https` address. The local provider sets it to its
MinIO on the Docker network (`http://minio.trenova.internal:9000`), while browsers fetch documents
through the local edge at `https://files.instances.localhost`.

**CORS.** The TMS hands the browser presigned links to the documents bucket, so the bucket must
admit the instance's origin. Once per origin, the operator puts the bucket's CORS rules (GET, HEAD
and PUT from `https://<hostname>`) with the documents credential, signed with AWS Signature V4,
and records the origin on the `trenova-storage` Secret (`trenova.app/cors-origin`). A failure is
logged and retried five minutes later; it never holds converging.

**Backups** are CloudNativePG's Barman Cloud plugin: the chart renders the `ObjectStore` (the
backups bucket, `retentionPolicy: 30d`, server-side encryption) and a daily `ScheduledBackup`
(suspended while PostgreSQL is hibernated), and the cluster archives WAL through the plugin under
its own name. The plugin and cert-manager are installed once per VM by cloud-init. The operator
creates two kinds of base backup itself, as CloudNativePG `Backup`s with `method: plugin`:

- **before an update:** the spec's `backup.id`, as `trenova-backup-<last 16 hex digits>`, once the
  revision carrying it is applied and the instance is active. It is reported on `backup.request`
  (`running`, `completed` with its ID and time, or `failed` with the reason); the control plane
  writes the new version only after `completed`.
- **the final backup:** once a `decommissioned` revision is applied, `trenova-final-r<revision>`.
  Completed, it is reported as `finalBackup` with that revision as `appliedRevision`; failed, it
  is deleted ten minutes after it stopped and taken again, and the health message says so.

**Health**, on every report while the instance has storage (`backup`): `walArchiving` from the
cluster's `ContinuousArchiving` condition (`healthy`, `failing`, `unknown`), `lastArchivedWalAt`
from the cluster's status, `lastBaseBackupAt` and `firstRecoverabilityPoint` from the object
store's recovery window for the cluster, `lastBaseBackupId` from the newest completed `Backup`,
and a `message` when archiving or the last backup failed. A change in any of them sends a report at
once.

**Restores.** A spec with `restore` renders a second cluster, `<release>-postgres-<generation>`,
bootstrapped by recovery from the source generation's backups to the target time, with small
requests and its own NetworkPolicy; the live cluster is unchanged. The operator reports it
`preparing`, `ready` once the new cluster is healthy (with `recoveredTo`), or `failed` if it is
not after six hours. A spec whose `database` names that generation switches to it: every
connection follows the cluster's name, and once the instance is ready on that revision the
operator hibernates every other cluster of the release (kept, never deleted) and reports
`switched`. Readiness checks the cluster `database` names.

## Logs

Every managed instance keeps its own logs for 7 days, separated by source (the chart's `logs`
component, which the operator always turns on; off by default for a self-hosted install):

- **Alloy** (`grafana/alloy`, one replica, `Recreate`) discovers the release's pods in the release
  namespace only (`discovery.kubernetes` with `namespaces.names` and a label selector on the
  release) and tails them through the Kubernetes API (`loki.source.kubernetes`), init containers
  included. Its ServiceAccount is bound to a namespaced Role with exactly `pods: get, list, watch`
  and `pods/log: get`: no hostPath, no DaemonSet, no ClusterRole. It is the one pod in the release
  that mounts a service account token. Each line keeps three labels and no others: `component`
  (the pod's `app.kubernetes.io/component`), `pod` and `container`. Every workload carries the
  component label, PostgreSQL's pods through the CloudNativePG cluster's `inheritedMetadata`, the
  release Job's pod through its template; a pod without one is not read.
- **Loki** (`grafana/loki`, single binary, one replica, `Recreate`) keeps the lines on its own
  claim (`<fullname>-loki`, 10% of the disk, at least 4 GiB, kept at its first size) in the TSDB
  and filesystem stores. Its compactor deletes anything older than 168 hours
  (`limits_config.retention_period`, `compactor.retention_enabled`), and lines older than that are
  refused. No authentication, analytics reporting off, a ClusterIP Service `<fullname>-loki:3100`
  in the release namespace. Its limits protect a small node: 4 MB/s ingestion (8 MB burst), 2 MB/s
  per stream, 1,000 streams, lines over 256 KB cut, at most 5,000 lines per query, four queries in
  parallel, a one-minute query timeout, and a 169-hour query range so the whole week can be read
  in pages.
- **Who reaches it.** Loki's NetworkPolicy admits Alloy and `managed.from` (the operator's
  namespace, as for the managed API) on 3100, and lets Loki open no connections. Alloy's admits
  nothing in and lets it reach Loki, DNS and the Kubernetes API (ports 443 and 6443; the chart
  cannot know the API server's address, so the port is the bound, which
  `logs.alloy.kubernetesAPI.to` can narrow).

The operator reads Loki at `TRENOVA_OPERATOR_LOKI_URL`, which its chart derives from
`target.release` and `target.namespace` (`http://<fullname>-loki.<namespace>.svc:3100`,
`values.LokiService`). What it does with the lines (the bundles and the quick view) is in the
design record's log contract and in `internal/commands`.

**Cost**, at the sizes above: 384 MiB of requests and 1.5 GiB of limits on the local cap and
Sandbox, about 0.6 GiB of requests at Production and 1.3 GiB at Production Plus, plus Loki's
volume; it is why the smallest machine moved to 10 GiB.

## Commands

Staff ask an instance for things besides its spec from the admin; the instance takes no inbound
connections, so the operator asks for them (`GET /v1/commands`) on every tick, carries each out
once by ID (an answer lost on the way is sent again from memory for an hour, without doing the
work again) and answers (`PUT /v1/commands/{id}`). Kinds:

| Kind | What the operator does | Permissions |
|---|---|---|
| `restart`, a component | patches `kubectl.kubernetes.io/restartedAt` on the pod template of the Deployments and StatefulSets with that `app.kubernetes.io/component` (the names the status report uses); `postgres` is refused (CloudNativePG restarts it), `release` is a Job and has nothing to restart | `deployments`, `statefulsets`: `patch` |
| `restart`, no component | the same for every Trenova workload | as above |
| `collect_logs`, a component | with log keeping on, the last `lines` (default 200, at most 5,000) Loki keeps for the component over the last 7 days, across every pod it had, newest last. With it off, the last `lines` (at most 500) of every container of the component's newest five pods, init containers included (the release Job's steps are init containers; CloudNativePG's pods run the Barman Cloud plugin as one), and a crashing container's previous run first. Either way each line is prefixed with its pod and container, every value of every Secret in the namespace is removed, and the output stops at 256 KiB keeping the newest lines. Works for every component the report names, `postgres` and `release` included, because the chart labels their pods with it | `pods`: `list`; `pods/log`: `get`; `secrets`: `list` |
| `collect_log_bundle`, a bundle and a period | the bundle ([Logs](#logs)); the command is answered when the bundle is stored, failed or refused | `secrets`: `list`, `get` (`trenova-storage`) |

`collect_logs` and `collect_log_bundle` carry `requestedBy {kind, email, name}`; one without it is
answered `failed`. For `kind: staff` the operator first asks the TMS
(`POST /managed/v1/support/log-pulls {email, kind: "lines"|"bundle"}`), which audits the pull and
answers 200 only while that staff address holds a live support access grant. A 403 answers the
command (and the bundle) `refused` with a sentence staff can read; a TMS that cannot be reached
answers it `failed`; neither reads a line. A customer's request needs no grant.

A command that fails is answered `failed` with a one-line reason the control plane accepts.

## Logs

The Trenova chart keeps 7 days of every pod's lines in the release's Loki
([`deploy/helm/trenova`](../../deploy/helm/trenova), `logs.enabled`), labelled `component`, `pod`
and `container`; the operator chart passes its address as `TRENOVA_OPERATOR_LOKI_URL` (empty: log
keeping is off, `collect_logs` reads the kubelet, and a bundle fails saying so).

**Reading Loki** (`internal/logs/loki.go`): `GET /loki/api/v1/label/component/values` for the
sources over the period, and `GET /loki/api/v1/query_range` with `{component="<name>"}`,
`direction=backward`, `start` and `end` in nanoseconds (start inclusive, end exclusive) and
`limit=2000`, newest first. The next page ends one nanosecond after the oldest entry of the last
one, and entries at that boundary already taken are skipped by stream and line, so a timestamp
shared across a page boundary is neither lost nor doubled. A page larger than 16 MiB is asked for
again with half the limit (down to 10). Each request has 30 seconds and is tried four times with
exponential backoff (`cenkalti/backoff/v5`) on a network error, a 429 or a 5xx; a 4xx fails at once.
A component name is checked (`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`) before it reaches a query or a
file name.

**A bundle** (`internal/logs/bundle.go`) is a zip of one `<component>.log` per source, lines
`timestamp pod container message`, oldest first, every line redacted and a line break inside a
message written as `\n`, and `manifest.json` (instance, bundle, period, the caps, and per
component its lines, bytes, oldest and newest kept line and whether and why it was cut). Caps
(`DefaultCaps`): a period of at most 7 days, 128 MiB of lines per source and 256 MiB for the bundle.
The bundle cap is shared: every source still reading gets an equal share of what is left, a source
that runs out returns the rest to the others, and sharing stops when a share would fall under
4 KiB. Lines are read newest first into a spill file per source, so a source cut at a cap keeps its
newest lines (what support needs), then written into the zip backwards, oldest first. Nothing is
held in memory beyond one page.

**Scratch space** is the operator pod's `emptyDir` `/scratch` (`sizeLimit: 1Gi`;
`TRENOVA_OPERATOR_SCRATCH_DIR`): at most 256 MiB of spill files plus the zip of them. Every file is
removed whatever happens, and the worker clears leftovers when it starts.

**Storing it**: `support/logs/<bundleId>.zip` in the instance's documents bucket, with the
documents credential the operator already holds (`trenova-storage`, through the private endpoint
when one is set), as one SigV4-signed `PUT` with its SHA-256 (the SDK's signer, as for CORS; an S3
single `PUT` takes up to 5 GiB), tried three times on a 429 or 5xx. Never the backups bucket: its
credential is the backup tool's alone and its prefix is the backup tool's. The object's metadata
records the per-source counts, so a bundle the operator sees again (a command listed again after a
restart) is found with a `HEAD` and reported without being collected or uploaded twice.

**The worker.** Bundles run one at a time on a background worker (a queue of eight), never on the
loop's tick. One that waited more than 25 minutes to start is failed without collecting (the
control plane fails a bundle nobody started within 30); each must finish within 45 minutes of
starting and 90 of being queued, inside the 2 hours after which the TMS and the control plane fail
it. A report after the control plane has an outcome is answered 204 and changes nothing. The control plane hears
`collecting`, then `stored` (key, size, per-source lines and whether cut) or `failed` (a sentence a
person can read) through `PUT /v1/log-bundles/{bundleId}`, tried five times. A bundle the same
process already finished answers from memory for an hour.

**Requests made in Trenova** (`internal/logs/relay.go`), every tick while the instance is active
and ready: `GET /managed/v1/log-requests` (requested and collecting, at most 10). For each request
the operator is not already working on: a period over 7 days, or log keeping off, is answered
`failed` to the TMS; otherwise it is registered with `POST /v1/log-bundles` (idempotent on the
request ID, so a restart gets the same bundle back), the TMS is told `collecting` with the bundle
ID, and the bundle is queued. When it ends the TMS is told `sent` with the bundle ID or `failed`
with the message, and told again on the next tick while it still lists the request. A 409 from the
TMS (the request already moved on) is left as it is. A 409 `conflict` from the control plane (a
bundle already in flight, or the day's 10 used) is never retried: the TMS is told `failed` with the
problem's `detail`, and nothing is collected.

## Its own version

The operator follows the release its instance runs. When the spec names release V and the
operator runs another, it upgrades itself first and brings Trenova to V only once it runs V:
a new chart may need a new operator (a value it renders, a kind it must be allowed to manage),
while the new operator only meets the older chart for the minute until it applies V. The rule is
the same downgrading. While the operator is not on V, nothing is applied: Trenova stays as it is,
its health is still read and reported, `appliedRevision` stays on the last revision applied, and
the report's message says why.

**How.** k3s's helm-controller installs the operator from the `HelmChart` `kube-system/trenova-operator`
(its release name is the chart's name). The operator patches that object's `spec.version` to V;
helm-controller runs `helm upgrade` at V from the same OCI chart with the same values. The
object's annotations record the move: `trenova.app/operator-upgrade-started`,
`trenova.app/operator-upgrade-failed` and `-reason`. Once the operator at V runs, it clears them,
and it rewrites `spec.version` to its own version if anything else moved it.

**Why the HelmChart is not in k3s's manifests directory.** k3s's deploy controller
(`pkg/deploy/controller.go`) reads `/var/lib/rancher/k3s/server/manifests` every 15 seconds; on
its first pass after every k3s start it applies every file again without comparing checksums,
through wrangler's apply, which builds a three-way merge (JSON merge for a custom resource) from
the object's last-applied annotation, the file and the live object. Every field the file names is
set back to the file's value. An operator that only patched a HelmChart written there would be
put back to the version the machine was made with at the next k3s restart or reboot. A file
named `x.yaml.skip` makes k3s ignore `x.yaml`; removing a file from the directory leaves what it
created (only `--disable` deletes). So the machine creates the operator's HelmChart once, from
`/etc/trenova/trenova-operator.yaml`, after k3s answers (`kubectl get … || kubectl create -f …`,
which never overwrites one that exists): OVH's cloud-init bootstrap script does it, and the local
provider runs the same command in its container. The control plane's `internal/provider/k3s`
renders the file and the command for both.

**When it fails.**

- The operator's Deployment rolls with `maxSurge: 1, maxUnavailable: 0`, and its pods take the
  Lease `trenova-operator` in their namespace: only the holder runs the loop. A new pod is ready
  once it can read its identity and `GET /v1/spec` with it, before it holds the Lease; only then
  does Kubernetes stop the old pod, which releases the Lease after its current tick. A new
  operator that crashes or cannot reach the control plane never becomes ready, and the old one
  keeps running the instance.
- The old operator watches the move: the install Job `kube-system/helm-install-trenova-operator`
  failing three times at V (a version the registry does not have; helm-controller retries up to
  1,000 times by itself), its own Deployment at V reporting `ProgressDeadlineExceeded` (after
  `progressDeadlineSeconds: 600`), or 20 minutes without a new operator taking over. It then puts
  `spec.version` back to its own version, records V as failed with the reason, and reports
  `operator.upgrade.phase: failed`. V is not tried again until the spec names another release,
  or until staff delete `trenova.app/operator-upgrade-failed` from the HelmChart.
- The HelmChart has `failurePolicy: retry`: after a failed `helm upgrade`, helm-controller
  upgrades again over the failed release rather than uninstalling the operator, which its default
  (`reinstall`) would do.

**What is reported.** `operator` on every status report: `version` (the release it runs),
`upgrade` (`version`, `phase` `upgrading` or `failed`) while it is not on the spec's release, and a
one-line `message` (why an upgrade failed, or that a node manifest pins its version). The admin
shows it on the instance page.

**Existing machines.** Machines made before this kept the operator's HelmChart in
`trenova-charts.yaml` in k3s's manifests directory, pinned to the version they were made with.
Their operator reports that (`message`: the node's manifest pins the operator's version) and
still moves itself on: it upgrades itself to V, and after a k3s restart, when k3s has put the
pinned version back, the running operator rewrites `spec.version` to its own version again, which
helm-controller follows within a minute. To make one converge for good, once, on the node:

```bash
touch /var/lib/rancher/k3s/server/manifests/trenova-charts.yaml.skip
```

k3s then ignores that file and leaves everything it created in place; nothing else in it changes
after a machine is made. Never delete the operator's document from that file instead: k3s would
apply the file without it and delete the HelmChart, which uninstalls the operator. An operator
older than this feature (0.12.x) does not upgrade itself: move its HelmChart to the first release
that does by hand, once (`kubectl -n kube-system patch helmchart trenova-operator --type merge
-p '{"spec":{"version":"<release>"}}'` after the `.skip`).

**Off.** An operator installed with `helm install` (CI, a self-managed cluster) finds no HelmChart
and leaves its version alone; `selfUpgrade.enabled: false` turns the RBAC and the lookup off.

## Readiness

A component is ready when its Deployment or StatefulSet has every replica updated and available
(stopped: none wanted and none left), the CloudNativePG cluster is healthy with every instance
ready (hibernated: the hibernation condition is true), and the release Job of the deployed Helm
revision completed. The report is ready when the desired revision is applied and every component
is ready; otherwise its message names the first one that is not.

The chart's own readiness and liveness probes never use Kubernetes' 1-second default timeout: each
sets `timeoutSeconds` of at least 3 (a few set more), so one slow second on a busy node does not
take a component out of its Service and the ingress with it. A readiness probe's `failureThreshold`
is never below 3. `TestNoReadinessOrLivenessProbeRunsOnTheOneSecondDefault` fails a new probe that
leaves the default.

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
StatefulSets, Jobs, NetworkPolicies, Ingresses, CloudNativePG Clusters, ImageCatalogs and
ScheduledBackups, and Barman Cloud ObjectStores, and for log keeping PersistentVolumeClaims,
ServiceAccounts, Roles and RoleBindings; the Secrets permission also covers the escrow and
storage Secrets), CloudNativePG Backups, which it creates itself, and read Pods, their logs and
Events. It can create Alloy's Role (pods and their logs, read) without the `escalate` or `bind`
verbs because it holds those permissions itself. In its own namespace it also holds its Lease and reads its own Deployment. With
`selfUpgrade.enabled` (the default), a third Role in `kube-system` lets it read and patch its own
HelmChart and read its install Job, both by name. Its one cluster-wide permission is the ClusterRole
`<release>-telemetry`: read nodes, their kubelet stats (`nodes/stats`, from the kubelet
directly), and the nodes' metrics ([Telemetry](#telemetry)); the release Role also lists the pods' metrics. The namespace,
which it keeps on uninstall, is created by its chart.

## Tests

- `go test ./...` in `services/operator`, against the in-process fake control plane, a fake
  Kubernetes, and Helm's memory storage with the real chart: enrolment, renewal, re-enrolment,
  revocation, rate limits, every plan and lifecycle through the chart's schema, the sizing
  (every component together and a rollout's largest surge pod fit each plan, the minimums and the local
  cap, read from the rendered chart; one GiB below a minimum does not; turning docling or the embedding service on changes no other
  workload's values or manifest, at every plan, with and without storage and under the local
  cap; a plan change divides again), PostgreSQL's shutdown settings, install and
  no-op detection, recovery, readiness, the escrowed secrets (fetched once, kept when present,
  restored when incomplete, refused when malformed, the install held until they are in place),
  the usage relay (mapping, chunking, scheduling, the token read again on a 401) against an
  in-process TMS, telemetry (the sampler against fake clusters: a part's copies summed, a part
  without a limit, the metrics API or the stats summary down, the same names as the status
  report, a suspended instance on the operator's node; Trenova's data figures carried, asked at
  most every 5 minutes, kept 30 minutes then left out, a 404 logged at most once an hour; the
  kubelet read over TLS verified by a test CA, the token sent and read again after rotation,
  another CA, a 403 and an oversized answer leaving disk out; the relay against the fake control
  plane: 30 oldest first, kept on failure, a refused batch dropped, the ring of 120, no minute
  twice, a 429 waited out, run on the minute and stopped cleanly; a failing relay leaving the status report as it was; the operator
  chart's telemetry ClusterRole exactly), the first administrator (bootstrap values, invalid names and emails, the invite
  installed, rechecked, replaced and reported, a failure omitted), the system email sender (sent
  when the fingerprint differs and not when equal, resent from memory, rotated, removed, retried,
  a mismatch and a 429 waited out, the credential in no log line or error), the system email
  events (nothing fetched while the commands answer says none wait, delivered then acknowledged,
  only the handled ones acknowledged, a TMS failure acknowledging nothing and retried on the next
  tick, one batch of at most 50 a tick, none while suspended or not ready, no payload in a log
  line or error) against the fake control plane and an in-process TMS, the support access
  keys (delivered when the key ids, domain or instance differ, not when they match in any order,
  not without `support`, a rotated set replacing the old, the key in no log line or error, a
  failure in the message and retried) and live grants (reported, an empty list reported, left out
  when the TMS could not be read, bounded and an unknown scope left out) against an in-process TMS
  as strict as the real one, the storage
  Secrets (fetched once into two Secrets, the backups credential never in the application's,
  restored when incomplete, refused when malformed, a `404` asked again only after ten minutes or
  at once on a new revision, the CORS request signed for the bucket), the commands (asked for on
  every tick, carried out when the report or the apply fails, each component's logs init
  containers and previous runs included, the loop and the runner refusing a missing dependency;
  the support access grant refused, granted and unreachable for staff lines and bundles; a bundle
  command collected once and answered when stored), log keeping (Loki's paging newest first
  across shared timestamps and smaller pages for a large one, its request shape and retries; a
  bundle's files oldest first, the per-source and shared caps and the manifest, redaction of every
  line; Loki or storage failing partway leaving no scratch file and a `failed` bundle; a stored
  bundle never uploaded twice, also after a restart; a Trenova request registered, collected,
  stored and sent end to end against the fake control plane and TMS; the documents bucket's
  signed `PUT` and `HEAD`), the operator's own version (moved first, Trenova held meanwhile, a version the registry lacks
  or one that never becomes ready rolled back and not tried again, a pinned HelmChart reported),
  backups (health from the cluster and object store, the
  pre-update and final backups taken once and reported, a restore reported and only its switch
  stopping the old database), the chart with storage (MinIO gone, the object store and schedule
  rendered, the backups Secret named by nothing else, a restore beside the live database and after
  its switch), log keeping (`logs_chart_test.go`: no ClusterRole, ClusterRoleBinding, DaemonSet
  or hostPath anywhere, Alloy's Role exactly pods and their logs, Loki's 168-hour retention, both
  NetworkPolicies, requests on every container, every workload labelled with its component,
  nothing rendered with `logs.enabled: false`, the operator's Loki URL for every release name),
  and the loop end to end.
- `.github/workflows/operator.yml` installs the operator on k3d against
  `cmd/controlplane-stub`, with every image built from the commit and the Trenova chart pushed
  to a registry in the cluster at the version the stub pins. It waits for a ready report on
  revision 1, checks the escrowed secrets were fetched once into `trenova-escrow` and that the
  TMS reads them from there, waits for a usage report relayed from the TMS's managed API,
  checks the release Job bootstrapped the first administrator for a setup link with no password,
  puts an invite in the spec (revision 2) and waits for it on a report, suspends (revision 3:
  workloads at zero, PostgreSQL hibernated) and resumes (revision 4).
- The contract: the operator's `api/instance.yaml` is a copy of the control plane's, and the
  control plane's CI compares the two, so a contract change lands in both.

## Not in the operator yet

- **Final backup size.** The plugin's `Backup` status carries no size, so `finalBackup.sizeBytes`
  is 0.
- **Removing a database a restore replaced.** It stays hibernated, with its volume, until the
  instance is torn down.
- **CI.** `.github/workflows/helm.yml` renders no shape with `backup.enabled` and has no schema for
  the Barman Cloud `ObjectStore`; the values tests render it through the chart instead.
