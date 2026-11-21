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
   ├── health.Check: Deployments, StatefulSets, CNPG Cluster, the deployed release's Job
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
| `cmd/tenant-agent` | the shared server's agent ([The tenant agent](#the-tenant-agent)); no Kubernetes or Helm import |
| `internal/instanceapi` | the client generated from `api/instance.yaml`, the operator's copy of the control plane's contract |
| `internal/controlplane` | that client with problem details turned into errors: `ErrUnauthenticated` (never retry with that certificate), `RateLimitedError` (`Retry-After`), `ProblemError` |
| `internal/credential` | the enrolment bundle, the operator's own key and identity (an instance's or a shared server's, `Kind`), the `Store` interface, the transport that swaps certificates, enrolment and renewal (`Manager`); `credential/secretstore` keeps the identity in Kubernetes Secrets, `credential/dirstore` in a directory (mode 0700, files 0600, written atomically) |
| `internal/values` | the spec turned into chart values: hostname, lifecycle, components, sizing, managed mode, the escrow Secret; the chart's names (`Fullname`, `AppSecret`, `APIService`, `EscrowSecret`) |
| `internal/escrow` | the escrowed secrets' shapes and validation; `escrow/escrowsecret` keeps them in a Secret in the target namespace, `escrow/escrowfile` in a directory, one file per key; both fetch from the control plane when missing or incomplete |
| `internal/release` | the chart from OCI at the pinned version, `upgrade --install`, recovery of an interrupted release |
| `internal/health` | each component's readiness, read from the cluster |
| `internal/tms` | the client of the TMS's managed API (the tenant routes included) and a token read from a file; `internal/tmssecret` reads it from the release's app Secret |
| `internal/usage` | the usage relay: readings from the TMS to the control plane, its answer back |
| `internal/telemetry` | what the instance uses, sampled every minute from the cluster (the sampler) and sent beside the loop; `internal/telemetryrelay` is the cluster-free queue and sender and the TMS data reader, `internal/hosttelemetry` samples a host (gopsutil) |
| `internal/tenantagent` | the tenant agent's loop |
| `internal/runlock` | the tenant agent's lock file |
| `internal/atomicfile` | write-then-rename with mode 0600 |
| `internal/securefile` | reads a file the process keeps (plain, its own, 0600) or one staff supply read-only (plain, writable only by its owner), opened with `O_NOFOLLOW` and checked on the descriptor |
| `internal/storage` | the instance's buckets: the documents and backups Secrets, fetched from `GET /v1/storage` when missing or incomplete, and the documents bucket's CORS rules for the instance's origin |
| `internal/backup` | backups read from and asked of CloudNativePG: health for the status report, the base backup before an update, the final backup, and a restore's progress |
| `internal/databases` | the instance's databases ([Database generations](#database-generations)): the gate before an apply that goes back to a kept database, stopping the others, retiring, and the `databases` list |
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
- **The images.** At publish, the workflow packages the release manifest
  (`release-manifest.json`: every Trenova image's digest and the version it was built under) and
  `image-digests.yaml` (every third-party image's digest) into the chart, and so inside the
  signature. The chart runs a Trenova image as `<repository>:<its version>@<digest>` and a
  third-party one as `<repository>:<tag>@<digest>`; an override to another repository or tag finds
  no entry and is used as given ([releases.md](releases.md)).
- **A release that changed little.** Moving to a release whose manifest carried an image's digest
  forward renders that image's pods exactly as before, so Helm's upgrade restarts nothing but what
  was rebuilt, and the release Job runs only when what it runs changed (its
  `trenova.app/release-inputs` hash). `NeedsApply` still applies the new chart version; the report's
  `version` is the deployed chart's appVersion, the release.

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
5. compare with the deployed release: apply only when the chart version or the values differ,
   and the database gate lets it ([Database generations](#database-generations)); after the
   report's health, stop the databases the instance does not run on and remove retired ones;
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

**Refused revisions.** A revision the operator will not apply as it is written is refused as a
whole, and the status report says so in two ways: `message`, a sentence for people, and
`refused` (`revision`, `code`, `reason`), for the control plane, which otherwise could not tell
"refused" from "not yet" and would wait on a going back or an update for ever. Nothing of a
refused revision runs, its `retiredDatabases` included; the report carries the instance's real
state beside it (the deployed release's health and readiness, `appliedRevision` on the revision
it runs). The operator works it out again on every tick, so it is on every report while the
cause stands (and a new or changed refusal sends a report at once), and it is gone once a later
revision is applied or the same revision becomes applicable (the database it names appears, for
one). The paths (`reconcile.refuse`, and the database gate):

| Code | When |
|---|---|
| `invalid` | the spec names another instance than the operator's certificate |
| `invalid` | `values.Render` returns `ErrSchemaVersion` (an unknown `schemaVersion`) |
| `invalid` | `values.Render` returns `ErrInvalidSpec`: a release version that is not semantic, a malformed hostname, an unknown lifecycle, a first administrator's name, organization or email out of shape, a `database` or `restore` generation out of shape, a `customHostname` out of shape or equal to the hostname or the files host, a `restore` on an instance without backup storage, or a plan (after the node's capacity) below the minimum |
| `database_not_held` | `database` names a generation the cluster does not hold and `restore` does not name, while it holds others of the instance |
| `first_database_removed` | no `database`, and the first database is gone while the cluster holds others |

Not refusals, because the revision is right and something else is in the way, each with its own
signal: the storage or escrowed secrets the control plane serves missing or malformed (fetched
again), the chart not loading (the registry, or a signature that does not verify), a failed Helm
apply, the operator's own move to the release (`operator.upgrade.phase`), a going back waiting
for its database, and a cluster list that fails. A database in `retiredDatabases` that is
protected (live, connected, restoring or a restore's source) does not refuse the revision either:
the rest is applied, the database stays listed, and `message` names it.

**Capabilities.** Every status report carries `capabilities`: what this operator can do that an
older one could not (today `custom_hostname`), so the control plane offers a feature only to an
instance that can carry it out. An operator from before a field decodes the spec, ignores it and
still reports the revision applied; without this list the control plane would mark a custom
address live that nothing serves. The list is `values.Capabilities`, beside the code that turns
`customHostname` into values (a test fails if one is there without the other); the loop puts it
on every report it sends, the first after a start and a refused revision's included.

**Recovery.** A release left `pending-*` by a crash mid-upgrade is rolled back to its last
deployed revision before the next apply; one that never deployed is uninstalled and installed
again. A failed release with a deployed one before it is upgraded over. An upgrade that fails
deletes what it had created before it failed (`CleanupOnFail`): Helm compares the next upgrade
with the last deployed release, so a resource a failed upgrade made is in no release's manifest
and would stay for ever (a restore's network policy did, when its database was refused). Only
resources made by that upgrade go: a database the release already held, kept or live, existed
before it and is never among them. A restore's candidate made by the failed upgrade is deleted
and made again by the next one; it holds nothing the backups do not.

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
| `customHostname` | `aliases: [customHostname]` and `primaryDomain: customHostname`: the instance is served on both addresses and links use the customer's. Absent, neither is set (and any the base values carry are dropped). Checked like `hostname`, and refused when it equals `hostname` or the files host: an invalid one refuses the revision like any other invalid spec |
| `instanceId` | `managed.instanceId`, which the chart passes to the TMS as `managed.instanceId`: the tag on the email Trenova's sender carries ([System email](#system-email)) |
| `account.supportUrl`, `account.newSupportRequestUrl` | `managed.supportUrl`, `managed.newSupportRequestUrl`, which the chart passes to the TMS under `managed`: the console's support page and its new request form for this instance, behind the account menu's "Your support requests" and "Contact Trenova support" ([managed-mode.md](managed-mode.md#trenova-support-from-the-account-menu)). Each is rendered only when it is an absolute `http` or `https` address with a host, no user information, fragment, space or control character, at most 2,048 characters; otherwise, or when the spec has no `account`, it is left out and so is its menu item |
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
| `release.version` | the chart version, whose release manifest pins every Trenova image by digest |
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
`maxUnavailable: 0, maxSurge: 1` because they must keep serving, and so does the client, whose
Caddy proxies `/api` and `/graphql`: its old pod also sleeps ten seconds (`preStop`) after it
leaves the endpoints and before Caddy stops, so the ingress controller has dropped it first (without
the pause a client restart failed API calls for about two seconds). The imaging worker and
Gotenberg keep Kubernetes's default (a 25% surge, one pod at one replica): the API calls both on
its request paths. Kubernetes creates a
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

**One start per restart.** Temporal runs every service in one process and advertises `127.0.0.1`
for its cluster membership (`TEMPORAL_BROADCAST_ADDRESS`). With the pod's address, a restarted pod
read the gone pod's rows from the membership table, waited 30 seconds to join them, crashed and
started again: about a minute before matching answered, while the probe said ready. Loopback is
the same address in every pod, so a restart is one start (measured: ready in 13 to 15 seconds,
matching answering in 8 to 11, no crash, against 17 to 21 and 49 to 64 with a crash before), and
it holds only because Temporal runs one replica. Its readiness probe describes a task queue
in `temporal-system`, which goes through matching, so ready means matching answers. gtc's readiness
probe is `/readiness` (503 while the replication stream or a sink is down), its liveness `/health`,
so a stalled pipeline shows as a part not ready and a sink that is away never gets gtc restarted.
`tools/helm/check-restarts.py` holds the client's pause, gtc's probes and Temporal's address.

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

A volume takes its size when it is made and keeps it: a claim can only grow, and k3s's
`local-path` class cannot grow one. A StatefulSet's volume claims cannot change after it is
created, so Redis, Meilisearch, MinIO and ClamAV keep the chart's sizes. Loki's claim is its own
object and takes the plan's size when it is made, then keeps it (the chart reads the live claim
back), because following a plan change would fail the upgrade. A PostgreSQL cluster takes the
plan's size when it is made, so a database made after a plan change (a restore's new generation,
a downgrade's new server) has the new one, and the operator sets
`postgres.storage.resizeInUseVolumes: false`: CloudNativePG would otherwise try to grow the
running cluster's claims, be refused, and apply nothing after that, so the plan's new memory and
parameters would never reach the database and the cluster would stay in `Applying
configuration`. None of this limits what the database can hold: a `local-path` claim's size is
not enforced, and every claim is a directory on the machine's data volume.

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
| `parts[]`: name, `ready`, `replicas`, `readyReplicas` | `health.Checker.Parts`, the same reading the status report's `components` come from (Deployments, StatefulSets, the CloudNativePG `Cluster`, the release Job the deployed release names), so the names are exactly the report's | `parts` is empty |
| `parts[].restarts` | the restart counts of every container (init containers included) of the part's pods, summed over its copies | left out (the pod list failed) |
| `parts[].startedAt` | when the newest running container (init containers left out) of the part's pods started; it moves forward whenever the part starts again, by a crash, a `restart` command, an update or a moved copy, which the counter above cannot show because a replaced pod counts from zero | left out (the pod list failed, or none of its containers runs) |
| `parts[].cpuLimitMillicores`, `memoryLimitBytes` | the limits of the containers of the part's running pods, summed; left out when any container has none, or no pod runs | left out |
| `parts[].cpuMillicores`, `memoryBytes` | `metrics.k8s.io` `PodMetrics` in the target namespace, summed over the part's running pods; 0 when none runs | left out, also when one running pod has no metrics yet |
| `parts[].version` | the tag of the workload's first container image (the Cluster's `status.image`), without registry, name or digest | left out when the image has no tag |
| `machine.cpuCapacityMillicores`, `memoryCapacityBytes` | the Node's `status.capacity` | left out |
| `machine.cpuMillicores`, `memoryBytes` | `metrics.k8s.io` `NodeMetrics` (memory is the working set) | left out |
| `machine.diskBytes`, `diskCapacityBytes` | the kubelet's stats summary, read from the kubelet itself (below): the image filesystem (`node.runtime.imageFs`), which is where k3s keeps its images and its `local-path` volumes, so on a machine with a data volume mounted for k3s it is that volume and not the disk the machine boots from; `node.fs` when the kubelet reports no image filesystem. What is taken is capacity less what is available, because the image filesystem's own `usedBytes` counts only the images | left out: the node has no InternalIP, the certificate does not verify, the kubelet refuses or the answer is over 8 MiB |
| `data.databaseBytes`, `documentsBytes`, `searchBytes` | Trenova itself: `GET /managed/v1/data-usage` on the TMS's managed API (`tms.Client.DataUsage`, the managed token as for every TMS call), at most every 5 minutes with 10 seconds, never on the loop's tick; each figure is absent when Trenova could not measure it | the last good answer is used for up to 30 minutes, then the figures are left out; a release without the endpoint (404) leaves them out and says so at most once an hour |
| `data.logsBytes` | the stats summary's volume statistics: used bytes of the volume claims mounted by the pod labelled `loki` | left out when the summary fails, the volume reports no usage, or the volume shares one of the node's own filesystems: a volume that is a directory on the node's disk (k3s's `local-path`, a host path) reports the whole disk's capacity and usage, which says nothing about the volume, so a volume whose capacity equals the node filesystem's or the image filesystem's is not counted |

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
admit every origin the instance is served on: `https://<hostname>` and, while the spec carries
one, `https://<customHostname>`. A `PUT ?cors` replaces the bucket's whole configuration, so the
operator puts one document whose single rule lists every origin (GET, HEAD and PUT), with the
documents credential, signed with AWS Signature V4, and records the set on the `trenova-storage`
Secret (`trenova.app/cors-origin`, the origins comma-separated, the first address first). A set
that differs from the recorded one in either direction (an address added, removed or changed) is
put once; an unchanged set costs no call. A custom hostname the spec carries but the operator
refuses is never put. A failure is logged and retried five minutes later; it never holds
converging.

**Backups** are CloudNativePG's Barman Cloud plugin: the chart renders the `ObjectStore` (the
backups bucket, `retentionPolicy: 30d`, server-side encryption) and a daily `ScheduledBackup`
(suspended while PostgreSQL is hibernated), and the cluster archives WAL through the plugin under
its own name. The plugin and cert-manager are installed once per VM by cloud-init. The operator
creates two kinds of base backup itself, as CloudNativePG `Backup`s with `method: plugin`:

- **before an update:** the spec's `backup.id`, as `trenova-backup-<last 16 hex digits>`, once the
  revision carrying it is applied and the instance is active, annotated
  `trenova.app/backup-origin: requested` and `trenova.app/backup-request-id: <id>` (and labelled
  `trenova.app/backup-request`). It is reported on `backup.request` (`running`, `completed` with
  its ID and time, or `failed` with the reason); the control plane writes the new version only
  after `completed`.
- **the final backup:** once a `decommissioned` revision is applied, `trenova-final-r<revision>`,
  annotated `trenova.app/backup-origin: final`. Completed, it is reported as `finalBackup` with
  that revision as `appliedRevision`; failed, it is deleted ten minutes after it stopped and taken
  again, and the health message says so.

**Which database.** Health, the backup list, the backups the operator takes and a restore's
`switched` all describe the database the application runs on, the one the deployed release points
it at, never the spec's `database` while a move to it waits ([Database
generations](#database-generations)): the control plane must never read another database's
backups as the instance's. Without a release, it is the spec's (the one the chart makes). A tick
that could not read the release leaves `backup` and `restore` off the report, and the control
plane keeps what it has.

**Health**, on every report while the instance has storage (`backup`): `walArchiving` from the
cluster's `ContinuousArchiving` condition (`healthy`, `failing`, `unknown`), `lastArchivedWalAt`
from the cluster's status, `lastBaseBackupAt` and `firstRecoverabilityPoint` from the object
store's recovery window for the cluster, `lastBaseBackupId` from the newest completed `Backup`,
and a `message` when archiving or the last backup failed. A change in any of them sends a report at
once.

**The backup list** (`backup.backups`, `internal/backup/list.go`) is every base backup of the
cluster the instance runs on, from the same single `Backup` list call that `lastBaseBackupId`
comes from (no call per backup):

| Field | From |
|---|---|
| `id` | `status.backupId`, or the object's name before the plugin has one |
| `phase` | `completed` and `failed` as CloudNativePG says; anything else (`pending`, `started`, `running`, `finalizing`, `walArchivingFailing`, none yet) is `running` |
| `startedAt` | `status.startedAt`, or the object's creation time before it has one |
| `completedAt` | `status.stoppedAt` of a completed or failed backup |
| `origin` | `requested` or `final` from the operator's `trenova.app/backup-origin` annotation (with `requestId` from `trenova.app/backup-request-id`); `scheduled` when a `ScheduledBackup` owns it (the chart's schedule sets `backupOwnerReference: self`); left out for one made by hand |
| `message` | `status.error` of a failed backup, on one line |
| `sizeBytes` | never: the plugin's `Backup` status carries no size |

It keeps completed backups, running ones, and failed ones that stopped in the last 7 days, newest
first by `startedAt`, at most 100. CloudNativePG does not delete a `Backup` object when the
plugin's retention (`retentionPolicy: 30d`) removes its files from the bucket, so the object alone
does not say whether it can still be restored from: a completed backup that stopped more than a
minute before the object store's `firstRecoverabilityPoint` is left out, since retention has
removed it (the minute absorbs the object's and the catalog's clocks rounding differently). With
no recovery window yet, every completed backup is kept.

The list is not on every report. The loop keeps the SHA-256 of the last list a report carried
that the control plane answered 2xx to (`Loop.lastBackups`, beside `last`, the report it
accepted), over each entry's fields in order; a list whose digest differs, which includes the
first report after the operator starts, is put on the report and sends it at once. A refused or
failed report records nothing, so the list goes again on the next tick. A failed list read leaves
`backups` out, changes nothing else in the report (the health fields are read as before), and is
logged at most once a minute; the control plane keeps the list it has.

**Restores.** A spec with `restore` renders a second cluster, `<release>-postgres-<generation>`,
bootstrapped by recovery from the source generation's backups to the target time, with small
requests and its own NetworkPolicy; the live cluster is unchanged. The operator reports it
`preparing`, `ready` once the new cluster is healthy (with `recoveredTo`), or `failed` if it is
not after six hours. A spec whose `database` names that generation switches to it: every
connection follows the cluster's name, and once the instance is ready on that revision the
operator reports `switched`; the database it ran on is stopped by the rules below. Readiness
checks the cluster `database` names.

## Database generations

Each database of an instance is a CloudNativePG `Cluster` of its own: the first one,
`<fullname>-postgres`, and one per restore, `<fullname>-postgres-<generation>` (`r` and eight hex
digits). `internal/databases` keeps them; the contract's `SpecDatabase`, `retiredDatabases` and
`DatabaseState` say what the control plane asks and reads.

**How the application follows a database.** The chart renders two kinds of `Cluster` and no
more: the live one (`postgres.generation`, with `helm.sh/resource-policy: keep`) and a restore
candidate (`postgres.restore`, without `keep`). Everything the application connects with is named
after the live cluster: the host `<cluster>-rw`, the CA `<cluster>-ca` that every client verifies
against, the release Job's superuser Secret `<cluster>-superuser`, the `cnpg.io/cluster` selectors
of the NetworkPolicies and the `ScheduledBackup` `<cluster>-daily`. Moving the application is
therefore one Helm upgrade with a new `postgres.generation`: the API and the worker roll
(`maxSurge: 1`, their new pods wait in `Init` for the database), the release Job runs against the
new cluster, and the old `Cluster`, no longer rendered, stays where it is because of `keep`. Going
back renders a cluster the release left behind: Helm adopts it (its `meta.helm.sh/release-name`
annotation is still the release's), using the live object as the baseline for its patch.

**What is per generation.** The chart renders only the `Cluster` (and, for a candidate, its
NetworkPolicy `<fullname>-postgres-<generation>`; for the live one the `ScheduledBackup`); Helm
removes those two with the values that rendered them. CloudNativePG makes everything else and owns
it through the `Cluster`: its pods, its volume claims (`<cluster>-1`, labelled `cnpg.io/cluster`
and, through the chart's `inheritedMetadata`, with the release and `component: postgres`), its
Services (`-rw`, `-ro`, `-r`), its Secrets (`-ca`, `-server`, `-replication`, `-superuser`) and its
disruption budget. Shared by every generation, never per generation: the `ObjectStore`
`<fullname>-backups`, the `ImageCatalog`, the database credentials Secret `<fullname>-db`, and the
backups bucket, where each generation archives under its own cluster name.

**On every reconcile**, in this order (the live database is the generation `database` names,
or the first one without it):

1. **The gate**, before anything is applied (`Manager.Gate`). Nothing to decide when the
   deployed release already points the application at the live database, or when `database`
   names the restore's own generation (the switch; the chart makes or keeps it). Otherwise the
   release's clusters are listed:
   - the live database is not held (or is being deleted) while the cluster holds others of the
     instance: **refused** ([Refused revisions](#the-loop): `database_not_held`, or
     `first_database_removed` without `database`). Nothing is applied, the instance stays on the database it runs on,
     and the message says which was named. A `database` absent while the first database has been
     removed is refused the same way, so an empty first database is never made under a running
     instance. When the cluster holds none of the instance's databases, the chart installs
     (the first install, or a new machine recovering `database` from the bucket);
   - held and stopped (`cnpg.io/hibernation: on`), the instance active and a release deployed:
     **going back**. The operator sets the annotation to `off`, applies nothing, and comes back
     on the next tick; it applies once CloudNativePG reports the cluster healthy with every
     instance ready and the hibernation condition gone. Meanwhile the deployed release is
     reported as it runs, `appliedRevision` stays on the revision it runs, and the message says
     `going back to <database>: starting it` or `…: waiting for it to accept connections
     (<phase>)`. Every change in the revision waits with it. The gate never blocks the loop and
     has no deadline: a database that never starts keeps the instance where it was, saying why;
   - suspended or decommissioned, nothing is started: the chart's lifecycle values decide
     whether the database runs (suspended hibernates it, decommissioned keeps it up for the
     final backup), as they do for any live database.

   A list that fails holds the apply (a move needs it), with the reason in the message.
2. **The chart** is applied as before; the application follows `postgres.generation`.
3. **Stopping the others** (`Manager.Settle`), only once the desired revision is applied and
   the instance is ready on it, so the database the application ran on stays up until every new
   pod runs and the release Job finished: every cluster of the release that is not the live
   one, not the one the deployed release points at, not the restore's `generation` and not
   already being deleted gets `cnpg.io/hibernation: on`. Its volumes are kept. This runs every
   reconcile, so a database left running by hand, or by a going back that was called off, is
   stopped too. This replaces stopping the others only at the moment a restore reported
   `switched`, which left a database a going back abandoned running.
4. **Retiring.** Each generation in `retiredDatabases` is removed unless it is the live one, the
   one the deployed release points at, the restore's `generation` or its `sourceGeneration`;
   those are left alone and named in the message (`not removed, though the spec retires them:
   …`). A held `Cluster` is deleted in the foreground with its UID as a precondition, so exactly
   the object listed is deleted and it stays visible, `removing`, until Kubernetes has removed
   what it owns. Once the `Cluster` is gone, any volume claim still labelled with its name, the
   release and `component: postgres` is deleted the same way. Removing one that is already gone
   does nothing. The backups bucket, the `ObjectStore`, the `Backup` objects (the operator's own
   and the schedule's) and every other generation are never touched.

**The owner check.** A cluster is the instance's only when the label selector
(`app.kubernetes.io/instance=<release>,app.kubernetes.io/component=postgres`) and its name both
match: the first one's name, or that name, a dash and a generation. A cluster of another release
in the same namespace, a cluster whose name is not a generation, and a volume claim labelled with
the cluster's name but another release, are never listed, stopped or removed.

**A decommissioned instance** is left to decommissioning: the gate refuses a database that is not
held, as always, but starts nothing, and nothing is stopped or removed. Suspended, the others are
stopped and retired databases are removed as when active; nothing is started.

**The list** (`databases` on the status report) is every cluster of the instance, the one the
deployed release points the application at first (`live`), then newest first, at most 20:

| `state` | When |
|---|---|
| `removing` | being deleted, retired on this tick, or gone with volume claims left |
| `stopped` | kept with no server by intent: `cnpg.io/hibernation: on` (also the live one of a suspended instance); and a database that is not meant to run and does not accept connections, which the next ready reconcile stops |
| `running` | healthy, every instance ready, not hibernated |
| `starting` | should run and does not accept connections yet: the live one (the spec's `database`, which is the one a going back is waking) or the one the application is connected to, while it starts, restarts or fails |
| `restoring` | the restore's `generation`, not live, not accepting connections yet |

`sizeBytes` is the sum of `status.capacity.storage` of its volume claims, left out when any of
them has none (k3s's `local-path` reports capacity, not usage); `createdAt` is the `Cluster`'s
creation time. It goes on the report under the backup list's rule, through the same mechanism
(`acceptedLists` in the loop): the first report after a start, then only when its digest differs
from the last list the control plane answered 2xx to; a refused report records nothing. A failed
cluster list leaves `databases` out and is logged at most once a minute.

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
| `restart`, no component | every Trenova workload but the database, in tiers (`restartOrder` in `internal/commands/restart.go`): the systems the application starts against (cache, search, Temporal, file storage, virus scanning, document reading, search by meaning, Loki), then search indexing and Alloy, then the API, the worker, imaging and Gotenberg, then the client, and a component the list does not name last. Each tier is patched together and the next waits until every workload in it is ready by the rule the status report uses (`health.Deployment`, `health.StatefulSet`). A tier not ready within ten minutes does not hold the rest: the restart goes on and the command is answered `failed`, naming the parts that did not come back. It runs beside the tick and is answered at the tick after it ends; a second one waits until the first is done. A test renders the chart with every part on and fails for a workload the list does not place | as above, and `get` |
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
object's annotations record the move: `trenova.app/operator-upgrade-started`, `-from` (the
release it moved from), `trenova.app/operator-upgrade-failed` and `-reason`. Once the operator at
V runs, it clears them, and it rewrites `spec.version` to its own version if anything else moved
it.

**Which version it runs.** Nothing in the operator's pod names a release: the chart puts it on the
Deployment (`trenova.app/operator-version`), and the operator reads it there on every tick. While
every pod of the Deployment is the current one, that is the version it runs; while a new pod rolls
out, the running one is still on the version in `-from`. So a release that did not rebuild the
operator's image moves its chart without a new pod, and the same process is on V a tick later; a
release that did rebuild it rolls a new pod, watched as below.

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
ready (hibernated: the hibernation condition is true), and the release Job the deployed release's
manifest names completed (`release.Deployed.ReleaseJob`: a revision that changed nothing the Job
runs keeps an earlier revision's Job, so its name is read, never derived from the revision). The report is ready when the desired revision is applied and every component
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
  pre-update and final backups taken once and reported, a restore reported and its switch
  stopping nothing itself; the backup list's phases and origins, the operator's own backups
  listed with theirs, the 100 newest, a backup past the first recoverability point and a failure
  older than 7 days left out, a failed list read leaving the health as it was; against the fake
  control plane, the list sent on the first report, not again while unchanged, at once when a
  backup completes or is removed, again after a refused report, and left out when the read
  fails), database generations (`internal/databases` against a fake cluster, and the loop end to
  end with a log of every start, stop, removal and chart apply in order: going back to the first
  database and to an older restored one started, waited for, moved to, then the old one stopped,
  the chart never applied while the database it names does not accept connections; a database
  not held refused with nothing changed; the others stopped with no restore, never the restore
  candidate and not before the instance is ready on the revision; retiring exactly the named
  generation's cluster and volumes beside a decoy of another release, a volume claim of another
  release labelled with the same cluster and another generation of the same release, with the
  UID precondition and foreground deletion, then done when asked again; the live, connected,
  candidate and source refused in the message, with no `refused` on the report; refused
  revisions (`database_not_held`, `first_database_removed`, `invalid` for a plan below the
  minimum, an unknown schema and another instance's spec) reported beside the real state and
  applied revision, on the next report too, recorded by the fake control plane, nothing of them
  run, and cleared by a later applied revision or once the same revision becomes applicable; each state in the list, live first, sizes from
  capacity, `starting` only for the live or connected one and `stopped` for one kept by intent;
  backup health read from the database the application runs on while going back waits; the list sent first, not while unchanged, again after a refused report; suspended
  going back applied straight with nothing started; decommissioned stopping and removing
  nothing), the chart with storage (MinIO gone, the object store and schedule
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

## Move to Managed: the import

A spec carrying `import` (`SpecImport`) makes the instance's data from a tenant archive the control
plane copied into its own backups bucket under `imports/<id>/`, instead of the first-install
bootstrap (`internal/importer`, `values/import.go`, the chart's release Job). `getSpec` answers
`503` until the import is listed; the loop waits for its `Retry-After` and installs nothing.

- **Held, not stopped.** The spec is `lifecycle: suspended` until the switch. With `import` the
  operator renders `stopped: true` (no workload serves a person) but `postgres.hibernate: false`,
  and the chart runs the release Job while `import.id` is set even though the release is stopped:
  migrations, roles, publication, then the `import` container in place of `bootstrap`. The import
  runs whatever the lifecycle; the workloads follow the lifecycle.
- **The keys.** `import.archiveKeys` (served only) is exactly one key, `current`, made by Trenova
  for this import alone (the copy's data key is re-wrapped under it); `values.CheckImport` refuses
  any other count (`invalid`), so no shared server's key ever reaches a dedicated instance. It is written by the operator into a Secret,
  `<fullname>-import-keys` (`archiveKeys`, annotated with the import id), and reach the container as
  `TRENOVA_IMPORT_ARCHIVE_KEYS` by `secretKeyRef`; the values Helm stores name only the Secret. The
  bucket is read with the backup Secret (`backup.existingSecret`: `ACCESS_KEY_ID` and
  `ACCESS_SECRET_KEY` by `secretKeyRef`, the key id passed as `$(…)` so the pod spec holds no value).
  The Secret is deleted once the import settles, and whenever the spec no longer carries `import`.
  Nothing logs the keys.
- **The command.** `trenova db import-archive --id --endpoint <backup.endpointURL> --region
  $(TRENOVA_IMPORT_REGION) --bucket --access-key-id $(…) --expect-sha256
  --status-file /dev/termination-log`. The Job is annotated `trenova.app/import: <id>`; its
  `podFailurePolicy` fails it at once on exit 2 (refused), exit 1 is retried within the Job's
  `backoffLimit`, and `import.activeDeadlineSeconds` (3 hours) bounds it.
- **The report.** `StatusReport.import` from the Job of the deployed Helm revision: `importing`
  (with when the Job started) once it exists, `done` when it completes, `refused` when it fails,
  with the reason from the container's termination message (the CLI's last `ImportStatus` line):
  "The archive was refused: …" on exit 2, "The import could not complete: …" otherwise. It is sent
  while the spec carries `import`, and always names the import's id.
- **Once.** The settled outcome is recorded in the ConfigMap `<fullname>-import` (id, state, at,
  reason). From then on the values carry `import: {id, settled: true}`, which runs neither the import
  nor the bootstrap again; a refused or failed import runs again only under a new `import.id`; an
  instance made by an import refuses any later import, and after the switch (no `import`) its
  bootstrap values stay empty, so `trenova db bootstrap` is never run against it. No first
  administrator invite is installed while the spec carries `import` or once the instance was
  imported (the imported people sign in with their passwords).

Tests: `values/import_test.go` (the chart's Job, the held values, no key in the manifest or values),
`reconcile/import_test.go` (a `503`, then importing, done, the switch to active with workloads
started and no invite; refused and failed with their reasons, never re-run, a new id runs),
`importer/importer_test.go` (the keys' Secret and the log).

## The tenant agent

`cmd/tenant-agent` runs a shared server: one Docker Compose service beside the TMS, many tenants.
It reuses the operator's packages and links no Kubernetes or Helm
(`cmd/tenant-agent/imports_test.go` runs `go list -deps` and fails on `k8s.io/`, `sigs.k8s.io/`,
`helm.sh/`, `reconcile`, `release` and `values`).

**Identity.** Staff issue the server's enrolment bundle (`serverId`, `apiUrl`, certificate, key, CA
bundle); it is installed at `<state-dir>/identity/enrolment.json` (or `--enrolment`). The certificate
names `urn:trenova:shared-server:<id>` (`credential.KindSharedServer`). `credential.Manager` enrols
onto a key the agent generates, keeps the identity in `<state-dir>/identity/identity.json` and
renews at two thirds of its life, exactly as the operator does with its Secret; a truncated
identity is invalid and the agent enrols again from the bundle. Files are read by one of two rules
(`internal/securefile`, each opened with `O_NOFOLLOW` and checked with `fstat` on the descriptor,
so nothing is swapped between the check and the read): a file the agent writes and keeps (the
identity, each escrowed secret in `--secrets-dir`) must be a plain file owned by the agent's uid
with no access for group or others, in a 0700 directory it owns, written atomically at 0600; a file
staff supply read-only (`--enrolment`, `--tms-token-file`) must be a plain file nobody but its owner
can write, and may be readable by everyone, because the shared server mounts it as a Compose file
secret that keeps the host's `0444 root` mode inside a `0700 root` secrets directory, the only mode
uid 65532 can read. A link is refused under either rule. Kubernetes' own mounts (the ServiceAccount
token and CA, the base values ConfigMap) are links by design and are read as Kubernetes gives them. A 401 makes it read the bundle again (`Recover`).

**The loop** (`tenantagent.Agent`), every `--poll` (15 s) with a fifth of jitter, never overlapping
(one goroutine), and exponential backoff up to 5 minutes while iterations fail. Every shared-server
operation goes through `tenantagent.PacedControlPlane`: a 429 or 503 holds that operation (the
tenants list, the status report, the storage read, the usage report) until its `Retry-After`
(seconds or an HTTP date, `controlplane.RetryAfter`, a minute when absent, at most an hour) and
never for less than the poll, while the others carry on; a held operation sends nothing. The
cadence stays inside the control plane's limits (status 600 an hour, usage 120 an hour per server)
whatever the number of tenants, because one report carries them all: a status report goes out when
it changed, at most once an iteration (about 270 an hour at the shortest jittered poll of 13.5 s,
for example while an archive's progress moves), and once a minute (60 an hour) when nothing
changed; the usage relay runs every 15 minutes and sends its two hours of readings in chunks of
5,000; a tenant has at most its four metered meters × two hours = 8 readings, so up to 625 tenants
take one request a relay: 100 tenants are four usage requests an hour. A done restore's `note` from the TMS is reported as
`SharedRestoreReport.note`, as one printable line of at most 500 characters, never in `reason`
and never with a failed restore.

1. renew the certificate when due; make sure the escrowed secrets are on disk
   (`escrowfile`, `--secrets-dir`, one 0600 file per key: `encryptionKey`, `rlsScopeKeyId`,
   `rlsScopeKey`, `mfaRecoveryCodePepper`, for the TMS's configuration to read);
2. `GET /v1/shared/tenants` with `If-None-Match` (the last list is kept on 304 and on failure);
3. bring the TMS to the list's system email sender (`systememail.Relay`) and its support access
   (`supportkeys.Relay.SyncShared`: the list's top-level `support`, staff domain and keys, as
   `PUT /managed/v1/support-access` without an `instanceId` whenever the key ids or the domain
   differ from what `GET` answers; `support` absent removes the keys with `DELETE`, so the server
   refuses every assertion);
4. `GET /managed/v1/tenants` (every page), then `PUT /managed/v1/tenants/{id}` with the entry's
   exact JSON for each tenant the TMS does not hold at that revision, or whose entry changed since
   it was last put (the digest of its bytes; the TMS also no-ops an unchanged digest), four at a
   time (`errgroup`); a tenant the TMS refused at that revision, one being removed, and a removed
   one are not put again; the commands are carried out in the same pass (see below);
5. `PUT /v1/shared/status`: version (`--server-version`), ready and the `tms` component (whether
   the managed listener answered), and each listed tenant the TMS holds (applied revision, state,
   administrator, `lastArchive`, `lastRestore`, `commandFailures`, `supportGrants`, refusal). It is sent when it changes and at least every minute; a
   tenant the answer names `not_placed` is no longer reported until a later list names it; a
   tenant the control plane no longer lists is not reported;
6. every `--usage-every` (15 minutes, the most the control plane allows): `usage.SharedRelay` reads
   `GET /managed/v1/tenants/readings` for the last two hours (`usage.Lookback`, so a missed hour is
   read again), maps each tenant's app meters to its plan's meters with that tenant's allowances (the
   dedicated relay's mapping), posts `POST /v1/shared/usage` in chunks of 5,000, and writes each
   tenant's answer back with `PUT /managed/v1/tenants/usage`. A failed report is retried on the next
   iteration. An entry the answer refuses (any `code`, `not_placed`, `implausible` or one this agent
   does not know) marks that tenant's hours in that report done: they are withheld from later reports
   until they leave the look-back, every other tenant and later hour is relayed as usual, and the
   refusal is logged with its code once per tenant per hour;
7. every minute, the email events (`emailevents.Relay`): the TMS finds each event's tenant from its
   message, and acknowledged events are deleted.

Telemetry runs beside the loop (`telemetryrelay.Relay`, one sample a minute): the host's processor
(cores and use), memory and the disk holding `--data-path` (`hosttelemetry`, gopsutil), and the
TMS's `GET /managed/v1/data-usage`; parts are left out (Compose has no replica counts to read).

**Commands** (`tenantagent/commands.go`). Each entry's `commands` (the `SharedTenantCommand` union;
an unknown kind is left be) are carried out oldest first, one at a time per tenant, and only once
the TMS holds the entry's lifecycle at its revision (an import: on the tenant's first entry). The
agent keeps no record of them: the TMS's routes are idempotent by command id, so for the first
command the TMS status does not show settled (`lastArchive`/`lastRestore` naming it, a
`commandFailures` entry, an applied revision for an import) it asks the TMS for it by id
(`GET /managed/v1/tenants/{id}/archives|restores|imports/{commandId}`): found and running is polled,
found and settled moves to the next, `404` is started. A restarted agent therefore never starts a
command twice. To start one it needs the command's storage target, which comes from the storage the
agent holds (see **Storage** below). A target is used only when it names the same tenant and command id,
an `https` endpoint, the server's own bucket and endpoint (those of `nightly`, when it is present:
a server has one bucket), the same kind, `write` for an archive and `read` for a restore or import,
the prefix `tenants/<id>/archives/<commandId>/` for an archive and exactly the command's `location`
for a restore or import, archive keys with exactly one `current` for every archive but an export
and none for an export; anything else is logged (without the credential) and not used. Then: an `archive` is
`POST .../archives {commandId, purpose, target}`, a `restore` `POST .../restores {commandId,
expectSha256, target}`, an `import` the tenant's first `PUT` with `importTarget` (held back while
no target fits, so the TMS never sees an import entry without its source). `423` (the tenant runs
another command) waits; `400`/`409` is a refusal the agent holds in memory while the command is
listed and reports as a `commandFailure` (archive, import) or as a failed `lastRestore` whose reason
starts "Nothing was changed: "; a restore the TMS ran keeps the TMS's reason, which starts "Nothing
was changed: " or "The tenant is incomplete: " (an unprefixed one is reported as incomplete, the
safe reading). A later revision never carries the import again: the target is attached only while
the TMS has no import command of that id.

**Storage** (`tenantagent/storage.go`). A shared server has its own archive bucket, its own access
key for it and its own archive keys; `GET /v1/shared/storage` names them as the standing nightly
target (`nightly`: `access: write`, `prefix: tenants/`, the keys with one `current`) and, for each
listed command, a target inside that bucket. The tenants document carries `storageRevision`, which
changes exactly when the storage answer would. The agent reads the storage once at start, then only
when:

- `storageRevision` differs from the one the held storage was read at;
- the TMS lists a `commandFailures` entry the agent has not seen whose `code` is `storage` (a bucket
  refused the credential) or `key_unknown` (a manifest names a key id the TMS was not given); a
  restarted agent counts the failures it finds as seen, since it reads the storage then anyway;
- a listed command is in `waiting`, at most once a minute, as the contract asks.

Never hourly and never on every poll. A read that fails keeps the held storage and is retried with
exponential backoff (from the poll interval up to five minutes).

The TMS's nightly archive storage comes from `nightly` and nothing else:
`PUT /managed/v1/tenants/archive-storage` with its endpoint, region, bucket, prefix, credential and
archive keys whenever it differs from what was last sent successfully (a SHA-256 of the body held in
memory, never on disk, so a restart sends once). When `nightly` is absent (no bucket recorded, or the
server is retired) the agent sends `DELETE` on that route once, so the TMS drops the credential and
keys and its nightly run does nothing. A `nightly` out of shape (access, prefix, an endpoint that is
not `https`, no access key, not exactly one `current` key) is not sent, and a storage the TMS answers
`400` (its prefix, its `managed.archive.allowedEndpoints`, https and private-address rules) is not
sent again until it changes: either is logged at error once per change and reported as the
`archive-storage` component (not ready, with the reason), which makes the report not ready. Once
the storage has been read the report always carries that component; with no `nightly` it is ready
and says nightly archives are off.

**Secrets.** Bucket credentials, session tokens and archive keys pass from the control plane to the
TMS over the Compose network with the bearer token and nowhere else: never logged (targets and
grants print as their bucket and prefix), never written to the state directory (the nightly digest
lives in memory), never in a status report. `TestNoSecretTheAgentHandlesReachesItsLogsOrItsStateDirectory` runs every command kind and
scans the logs, every report and every file under the state directory.

**When things fail.** The control plane down: the last list is converged and reported again later;
usage readings stay in the TMS and are re-read. The TMS down: the status says not ready with the
reason, nothing is put, and the next iteration tries again. The process never exits on these; it
exits on configuration errors, and on a second agent: `<state-dir>/tenant-agent.lock` is held with
`flock`, so a second copy started by mistake refuses to start. Should two run anyway (two hosts
sharing nothing), every TMS route is idempotent by instance id and revision (and digest), usage
answers are whole states, and the control plane replaces each report.

**Configuration** (flag, or `TRENOVA_TENANT_AGENT_*`): `--state-dir` (`STATE_DIR`,
`/var/lib/trenova-tenant-agent`), `--enrolment` (`ENROLMENT`), `--secrets-dir` (`SECRETS_DIR`,
`/run/trenova/escrow`), `--tms-url` (`TMS_URL`, `http://api:8090`), `--tms-token-file`
(`TMS_TOKEN_FILE`, `/run/secrets/managed_token`), `--data-path` (`DATA_PATH`), `--server-version`
(`SERVER_VERSION`), `--health-addr` (`HEALTH_ADDR`, `:8081`, `/healthz` and `/livez`), `--poll`,
`--usage-every` (`USAGE_EVERY`), `--log-level` (`LOG_LEVEL`). `--once` converges and reports once and exits (an init step);
`--healthcheck` asks a running agent's `/healthz` and exits 0 when healthy (the Compose healthcheck,
no curl in the image). Logs are JSON and carry no secret: the token, keys and escrowed values are
never logged.

**The image.** `deploy/Dockerfile.operator`, target `tenant-agent`: the same source stage and
distroless `static-debian12:nonroot` base as the operator, the binary at
`/usr/local/bin/trenova-tenant-agent`, user 65532, port 8081; it writes only its state and secrets
directories, so it runs with a read-only root. `release.yml` publishes
`ghcr.io/emoss08/trenova/tenant-agent` with every release's tags (version, major.minor, `latest`),
building it, with an SBOM and provenance, only when `services/operator` or its Dockerfile changed
([releases.md](releases.md)); `operator.yml` builds it in CI.

## Not in the operator yet

- **Backup sizes.** The plugin's `Backup` status carries no size, so `finalBackup.sizeBytes` and
  each listed backup's `sizeBytes` are left out.
- **CI.** `.github/workflows/helm.yml` renders no shape with `backup.enabled` and has no schema for
  the Barman Cloud `ObjectStore`; the values tests render it through the chart instead.
