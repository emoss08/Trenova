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
| Trenova support staff (managed instances only) | a single-use EdDSA assertion the control plane signed, posted to `POST /api/v1/support/enter`, then a session sealed like a person's | `identity/app/supportaccessservice.Enter` | a live grant an administrator of that organization made for exactly that staff address; the grant's support principal (never a person of the organization); read-only, or an administrator's reach less the deny list (`identity/domain/supportaccess/policy.go`); the grant's end, read on every request ([managed-mode.md](managed-mode.md#support-access)) |
| The operator, calling the TMS | the bearer `managed.token`, compared in constant time, on the managed listener (`api/managedserver*.go`, port 8090) that the chart's NetworkPolicy opens to the operator's namespace alone | `managedHandler.authenticate` | the instance-wide state the control plane decides: usage state, features, the first administrator's invite, the system email sender (`PUT`/`GET`/`DELETE /managed/v1/system-email`: an 8 KiB strict body, stored sealed, answered with whether one is set, its from address and fingerprint, never the credential) ([managed-mode.md](managed-mode.md)) |

Every root resolver and route reaches a permission check or is listed with a reason in `authzlint`.
Field resolvers are not covered by that lint: a field that crosses into another resource checks for
itself or goes through the field guard (`gqlext`).

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
| Delivery events for the Trenova system email sender (`/webhooks/email/system/{resend,postmark}/`) | the signature or basic authentication made with the instance's system sender webhook secret, then a message the Trenova sender carried, found by provider message ID; anything else is answered 204 and dropped | base (404 on an instance with no Trenova sender or no webhook secret) |
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
| Operator to TMS (the managed listener) | the bearer `managed.token`, compared in constant time, and a NetworkPolicy admitting only `managed.from`. It sets the usage state, the feature list, the first administrator's invite hash and the support assertion keys; none of them lets anyone in alone |
| Release to operator | a keyless Sigstore signature from `release.yml` at a version tag, carried in the chart's provenance layer ([operator.md](operator.md)) |
| Chart to images | digests written into the signed chart at publish |
| Operator to the cluster | a namespaced `Role`, no `ClusterRole`; a namespace the chart creates enforces `baseline` Pod Security |
| TMS to imaging worker | mutual TLS ([imaging-worker.md](imaging-worker.md)) |
| Every client to PostgreSQL | `verify-full` against the install's CA: CloudNativePG's on Kubernetes, `postgres-certs` in Compose. No image carries a key |
| TMS to Gotenberg | basic auth with a generated password on every route but `/health`, and the network: NetworkPolicies in the chart, a network of its own in Compose |
| TMS to Temporal | mutual TLS against a CA issued per install (the chart's Secrets, `temporal-certs` in Compose), on the frontend and between Temporal's services; production refuses a Temporal connection that presents no client certificate, API key or profile (`temporalconfig.ValidateProduction`) |
| GTC's dashboard | nothing but the network: NetworkPolicies in the chart, the data network in Compose. It takes no credential, so nothing that parses files may share a network with it |

A scanner driver is native code in the process that loads it, so what it returns crosses no
boundary; parse it defensively anyway. A printed job comes from whoever printed, so `pwg.rs` checks
every header field before it sizes anything.
