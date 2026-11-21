# Trust boundaries

Who can reach Trenova, how each proves who they are and what bounds them. Start here to review a
new entry point or to scope an audit. Paths are under `services/tms/internal/` unless they say
otherwise. The defects these boundaries tend to grow are in
[backend/security-review.md](backend/security-review.md).

## People and programs with an identity

| Who | Proves it with | Established in | Bounded by |
|---|---|---|---|
| A person (`PrincipalTypeUser`) | a session, sealed against tampering in Redis (`infrastructure/redis/authstore/seal.go`) | `authctx.SetSessionAuthContext` | the permission engine, active roles, row-level security |
| A federated person | an OIDC sign-in, then a session that records the issuer | `identity/app/authservice` | as above, and only in organizations that trust that issuer (`requireTargetSignIn`) |
| An API key (`PrincipalTypeAPIKey`) | a key looked up by hash | `authctx.SetAPIKeyContext` | its own grants |
| A SCIM directory (an identity provider provisioning people) | a token looked up by prefix, its hash compared in constant time, for the slug in the path | `scimservice.Authenticate` | its own organization and business unit, only the people it provisioned or linked, and for roles only what the person who saved each group mapping may still grant ([scim.md](scim.md)) |
| An agent (`PrincipalTypeAgent`) | the run it belongs to | the agent runtime | `agentceiling`, autonomy tiers, taint, and a person's approval for what it may not do alone |
| A Capture device | device tokens from a pairing a person approved | `api/handlers/capturehandler/device.go` | the person it acts for, the roles active at approval, their session epoch |
| A local print client | the Windows process on the other end of the socket | `desktop/capture/bins/trenova-capture-svc/src/win/peer.rs` | `ClientAccess` in `capture-ipp`: only the spooler may print, others may only discover |
| The operator | a client certificate named for its instance | `services/operator/internal/credential` | one instance's spec; it refuses a spec for another |
| The tenant agent (a shared server) | a client certificate named `urn:trenova:shared-server:<id>`, kept in a 0700 directory | `services/operator/internal/credential` (`KindSharedServer`) | the `/v1/shared/` routes for the tenants placed on that server, its own escrowed secrets, telemetry and email events; never an instance's spec |
| The tenant agent, calling the TMS | the bearer `managed.token`, compared in constant time, on the managed listener bound to the Compose network | `managedHandler.authenticate`, then `internal/cloud/api/tenanthandler` | `PUT /managed/v1/tenants/{instanceId}` makes or changes exactly the tenant the path names (an entry for another instance is a 400) and never binds an organization or business unit from the body (`internal/cloud/lint_test.go`); names and addresses are validated and an address taken anywhere on the server is refused without naming who holds it; the organization and its first administrator never change after the tenant is made; removal needs a recorded final archive. `GET /managed/v1/tenants` and `.../readings` answer ids, states, revisions, whether the administrator is set up and the live invite's hash, and per-organization meter counts, never a name or record; `PUT .../usage` sets each named tenant's usage state ([shared-tenancy.md](shared-tenancy.md#the-listeners-tenant-routes)) |
| Trenova support staff (managed instances only) | a single-use EdDSA assertion the control plane signed, posted to `POST /api/v1/support/enter`, then a session sealed like a person's | `identity/app/supportaccessservice.Enter` | a live grant an administrator of that organization made for exactly that staff address; the grant's support principal (never a person of the organization); read-only (every REST write and GraphQL mutation refused at the guard as well as by the engine), or an administrator's reach less the deny list (`identity/domain/supportaccess/policy.go`); never where money is sent, refused by a database trigger whichever path writes; nothing sent, shared or scheduled outside the organization and no change to what outside readers see, refused by a second trigger whatever the scope; the assistant and every model call only when the grant allows it; never the person a held change waits for; the grant's end, read on every request ([managed-mode.md](managed-mode.md#what-a-session-may-do)) |
| The operator, calling the TMS | the bearer `managed.token`, compared in constant time, on the managed listener (`api/managedserver*.go`, port 8090) that the chart's NetworkPolicy opens to the operator's namespace alone | `managedHandler.authenticate` | the instance-wide state the control plane decides: usage state, features with the add-ons set to end (at most 50, each key shaped like a catalogue add-on key) and the plan page's URL (an absolute `http`/`https` URL with a host and no user information, at most 2,048 characters, since the plan page renders it as a link), when the plan ends (the same channel and the same trust: a time no more than a day past and no more than 400 days ahead, which only drives the plan page's notice and the administrators' notifications), the first administrator's invite, the system email sender (`PUT`/`GET`/`DELETE /managed/v1/system-email`: an 8 KiB strict body, stored sealed, answered with whether one is set, its from address and fingerprint, never the credential), the bounces and spam complaints for email Trenova's sender carried (`POST /managed/v1/system-email/events`: an 8 MiB strict body of 1 to 50 events, each a provider webhook body the control plane verified and relayed; the instance cannot verify it again and trusts it for the channel, so it only ever adds a suppression to the organization whose Trenova-sent message, found by provider and provider message ID under a system scope, it names, and is otherwise dropped and logged without its payload), the requests to send logs to Trenova support that are waiting or being collected and its reports on them (`GET`/`PUT /managed/v1/log-requests`: a 16 KiB strict body, a status that only moves forward, read and written under a system scope by the request's id), and whether a staff address may pull logs (`POST /managed/v1/support/log-pulls`: answered 200 and audited in each organization whose live grant names that address, 403 alike for every refusal; it grants nothing), and the live support grants it reports to the control plane (`GET /managed/v1/support-access/grants`: read under a system scope across organizations, at most 100, each the staff address, organization id and name, scope and expiry, never who made it or why) ([managed-mode.md](managed-mode.md#logs-for-trenova-support), [the live grants](managed-mode.md#the-live-grants-instance-to-operator-to-control-plane)) |

Every root resolver and route reaches a permission check or is listed with a reason in `authzlint`.
Field resolvers are not covered by that lint: a field that crosses into another resource checks for
itself or goes through the field guard (`gqlext`).

The tenant archive routes on the managed listener (`internal/cloud/api/tenanthandler/archives.go`)
take a `SharedArchiveTarget` that must name the path's tenant, the command and its kind, and
reach only `tenants/<instanceId>/` of the bucket it names (or `imports/<id>/` for the dedicated
CLI). Its credentials and archive keys are sealed with the install's key while the command runs
and erased when it settles; they never reach logs, `cloud_tenants` or Temporal history. An
archive's own integrity rests on `expectSha256` (the manifest's digest, which names every part's)
and on AES-256-GCM per part: an archive is refused before anything is written if any byte differs.
An archive load runs under the system role in one transaction, with `trenova.archive_load`
believed only for roles that bypass row-level security. The same setting is the only thing that lets an
imported AI audit chain be signed again; the source's signatures stay beside the new ones.

Tenants on one shared server never share a row. In-network EDI (connections, tenders, shipment
links, tender recipients and changes) and memberships connect organizations of one business unit,
and each tenant has its own, so no company can link to, read or change another company's rows
through them; the application never offers a partner in another business unit, and the database
holds every such reference to the row's business unit (`20261231023600`).

## Callers with no identity

These run before a tenant is known, so they use `dbscope.WithSystem` and the credential they present
is the only thing that narrows the query.

| Entry | What narrows it | Feature (answer when off) |
|---|---|---|
| Customer shipment tracking, carrier tracking | a token looked up by its hash | Customer tracking (the invalid-link error) |
| Tender offer, rate confirmation | a token looked up by its hash | Carrier brokerage (the invalid-link error) |
| Invoice share, password reset, Capture pairing | a token looked up by its hash | base |
| Driver portal invitation | a token looked up by its hash | Driver portal (`PLAN_RESTRICTED`) |
| Telematics webhooks | the provider's signature, then a realm or account the tenant verified | Samsara (401, as for an unknown token) |
| Accounting webhooks | the provider's signature, then a realm the tenant verified | Accounting (200, as for an unknown realm: acknowledged and dropped) |
| Inbound mailbox | the mailbox token, then the provider's signature | Inbound triage (404, as for an unknown mailbox) |
| Outbound email delivery events | the provider's signature | base |
| AS2 | the profile's own inbound URL, an unguessable token that names one profile and so its tenant, then the identifier pair that profile holds and the partner's signature. A receipt is read inside that tenant and must belong to the profile's partner. A profile may accept unsigned documents, which the partner page warns about (`acceptsUnsignedInbound`) | EDI (the negative MDN an unknown pair gets) |
| Web watch deliveries | an HMAC over timestamp and body, per watch (`shared/exa`) | Agents (404, as for an unknown watch) |
| SCIM | the directory's bearer token | Single sign-on and provisioning (the 401 a bad token gets) |
| SSO start and callback | the organization's login slug and the provider it configured | Single sign-on and provisioning (the not-found an unconfigured provider gets); password and MFA sign-in are base |
| Trenova support entry (`POST /api/v1/support/enter`) | the control plane's signature with a key the operator delivered, this instance as audience, at most a minute's life, an assertion id never used before, then the one live grant for that staff address and organization: one lookup under the system scope, then the grant's tenant. Every refusal looks the same to the caller, and the browser provenance guard lets this one path through because it carries no cookie | not a plan feature: managed mode only, unrouted (404) when `managed.enabled` is off |

Each route group names its feature where it is registered, through `middleware.FeatureGuard`
([backend/api.md](backend/api.md)); `TestEveryRouteWithoutASessionDeclaresItsFeature` keeps
this table honest. The signed-in driver app (`/portal/` and the `My…` roots) needs the Driver
portal feature too: the guard on its REST group and `driverportalservice.ResolveWorker` answer
`PLAN_RESTRICTED`.

Tenants configure no outbound webhook. An integration's base URL goes through the
`shared/httpsafe` dial guard and refuses redirects; a new URL a tenant can set does the same.

## Text from outside

Anything written outside the organization (an email, a document, a web page, an EDI payload, a
portal) is data. Once a run reads it the run is tainted, and a tainted run holds writes that leave
the organization for a person ([agent-runtime.md](agent-runtime.md), "Taint"). Text that survives
the run carries the taint with it, into memory for one. Approvals bind to a digest of the merged
parameters, so what a person approved is what runs ([proposal-previews.md](proposal-previews.md)).

## What runs on an instance

| Link | Trust rests on |
|---|---|
| Control plane to operator | mutual TLS against the enrolment bundle's CA |
| Operator to TMS (the managed listener) | the bearer `managed.token`, compared in constant time, and a NetworkPolicy admitting only `managed.from`. It sets the usage state, the feature list with the add-ons set to end and the plan page's URL (validated as a web address before it is stored, since the plan page links to it), the first administrator's invite hash and the support assertion keys, relays the bounces and complaints for Trenova's system email (provider webhook bodies the control plane verified; the instance has no public webhook for Trenova's sender and holds no webhook secret, so this channel is the only way they arrive, and the worst a forged one can do is suppress an address in the organization whose message it names), and reads and reports on log requests and staff log pulls; none of them lets anyone in alone, and logs leave the instance only when an administrator asked or a live grant names the staff address. It also reads the live support grants (`GET /managed/v1/support-access/grants`, every organization's, under a system scope, at most 100) and passes them to the control plane on its status report: for each grant the staff address it is for, the organization's id and name, the scope and the expiry. It never carries who made a grant, the reason they gave or anything done under it, which stay in the organization's audit log; the list grants nothing, since entry still needs the live grant on the instance |
| Control plane to the tenant agent | mutual TLS against the enrolment bundle's CA, the server's own certificate |
| Tenant agent to TMS | the bearer `managed.token` on the managed listener, which Compose publishes on its internal network only. It applies each tenant's entry, sets each tenant's usage state, the system email sender and its events, and reads the tenants' states, readings and the server's data usage. A taken-over agent can change any tenant's plan, suspend or resume it, or (with a final archive recorded) remove it; it cannot read a tenant's records |
| Release to operator | a keyless Sigstore signature from `release.yml` at a version tag, carried in the chart's provenance layer ([operator.md](operator.md)) |
| Chart to images | digests written into the signed chart at publish |
| Operator to the cluster | namespaced `Role`s, no `ClusterRole`; a namespace the chart creates enforces `baseline` Pod Security. To upgrade itself it may `get` and `patch` one object outside its namespaces: its own `HelmChart` in helm-controller's namespace, by name (`selfUpgrade.enabled`). helm-controller installs what that object names as cluster admin, and RBAC cannot limit a patch to one field, so an operator that is taken over can reach the node through it. The node serves one instance, whose data the operator already holds |
| TMS to imaging worker | mutual TLS ([imaging-worker.md](imaging-worker.md)) |
| Every client to PostgreSQL | `verify-full` against the install's CA: CloudNativePG's on Kubernetes, `postgres-certs` in Compose. No image carries a key |
| TMS to Gotenberg | basic auth with a generated password on every route but `/health`, and the network: NetworkPolicies in the chart, a network of its own in Compose |
| TMS to Temporal | mutual TLS against a CA issued per install (the chart's Secrets, `temporal-certs` in Compose), on the frontend and between Temporal's services; production refuses a Temporal connection that presents no client certificate, API key or profile (`temporalconfig.ValidateProduction`) |
| GTC's dashboard | nothing but the network: NetworkPolicies in the chart, the data network in Compose. It takes no credential, so nothing that parses files may share a network with it |
| Alloy to the cluster (log keeping, `logs.enabled`; on for every managed instance) | a ServiceAccount bound to one namespaced `Role`: `pods` `get`/`list`/`watch` and `pods/log` `get` in the release namespace, nothing else (no Secrets, no other namespace, no `ClusterRole`, no hostPath). It is the only pod in the release that mounts a token. Taken over, it reads every log line of the instance, which carry no secrets by design but may carry customer data, and can push lines into Loki; its NetworkPolicy lets it reach only Loki, DNS and the Kubernetes API's ports |
| Anyone to Loki | nothing but the network: Loki has no authentication. Its NetworkPolicy admits Alloy (which writes) and `managed.from`, the operator's namespace (which reads, to send an administrator's or a support-granted bundle), plus `logs.queryFrom` on a self-hosted install, and Loki opens no connections. Anything else in the release namespace, the TMS included, cannot reach it. The operator removes every Secret value from what it sends; the lines in Loki itself are kept unredacted for 168 hours |
| Operator to logs it sends (`services/operator/internal/logs`, `internal/commands`) | It reads Loki (when `TRENOVA_OPERATOR_LOKI_URL` is set) and, without it, pod logs through the Kubernetes API, and only to answer: a `collect_logs` or `collect_log_bundle` command from the control plane, or a request an administrator made in Trenova (`GET /managed/v1/log-requests`). The grant is enforced on the instance, never by the control plane's word: before any read for a command whose `requestedBy.kind` is `staff`, the operator calls `POST /managed/v1/support/log-pulls`; only a 200 lets it read, a 403 answers the command and the bundle `refused`, and a TMS it cannot reach collects nothing. A customer's request (console or Trenova) needs no grant. Every line, from either source, has every value of every Secret in the release namespace and anything shaped like a credential replaced before it leaves (`internal/redact`) |
| Operator to the instance's object storage | the documents bucket's credential from `trenova-storage`, which the TMS holds too. The operator writes one kind of object there, `support/logs/<bundleId>.zip`, and reads only that key's metadata (to never collect a stored bundle twice). It never uses the backups credential, so a bundle never lands under the backup tool's path. The control plane reads the zip with its escrowed copy of the same credential and deletes it 30 days after it was stored |

A scanner driver is native code in the process that loads it, so what it returns crosses no
boundary; parse it defensively anyway. A printed job comes from whoever printed, so `pwg.rs` checks
every header field before it sizes anything.
