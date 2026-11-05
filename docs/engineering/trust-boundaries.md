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
| An agent (`PrincipalTypeAgent`) | the run it belongs to | the agent runtime | `agentceiling`, autonomy tiers, taint, and a person's approval for what it may not do alone |
| A Capture device | device tokens from a pairing a person approved | `api/handlers/capturehandler/device.go` | the person it acts for, the roles active at approval, their session epoch |
| A local print client | the Windows process on the other end of the socket | `desktop/capture/bins/trenova-capture-svc/src/win/peer.rs` | `ClientAccess` in `capture-ipp`: only the spooler may print, others may only discover |
| The operator | a client certificate named for its instance | `services/operator/internal/credential` | one instance's spec; it refuses a spec for another |

Every root resolver and route reaches a permission check or is listed with a reason in `authzlint`.
Field resolvers are not covered by that lint: a field that crosses into another resource checks for
itself or goes through the field guard (`gqlext`).

## Callers with no identity

These run before a tenant is known, so they use `dbscope.WithSystem` and the credential they present
is the only thing that narrows the query.

| Entry | What narrows it |
|---|---|
| Carrier tracking, tender, rate confirmation, invoice share, portal invitation, password reset, Capture pairing | a token looked up by its hash |
| Telematics, email, mailbox and accounting webhooks | the provider's signature, then a realm or account the tenant verified |
| AS2 | the partner's identifier and its signature. A profile may accept unsigned documents, which the partner page warns about (`acceptsUnsignedInbound`) |
| Web watch deliveries | an HMAC over timestamp and body, per watch (`shared/exa`) |

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
| Release to operator | a keyless Sigstore signature from `release.yml` at a version tag, carried in the chart's provenance layer ([operator.md](operator.md)) |
| Chart to images | digests written into the signed chart at publish |
| Operator to the cluster | a namespaced `Role`, no `ClusterRole`; a namespace the chart creates enforces `baseline` Pod Security |
| TMS to imaging worker | mutual TLS ([imaging-worker.md](imaging-worker.md)) |
| Every client to PostgreSQL | `verify-full` against the install's CA: CloudNativePG's on Kubernetes, `postgres-certs` in Compose. No image carries a key |
| TMS to Gotenberg | basic auth with a generated password on every route but `/health`, and the network: NetworkPolicies in the chart, a network of its own in Compose |
| TMS to Temporal, GTC's dashboard | nothing but the network: NetworkPolicies in the chart, separate Docker networks in Compose. Neither takes a credential, so nothing that parses files may share a network with them |

A scanner driver is native code in the process that loads it, so what it returns crosses no
boundary; parse it defensively anyway. A printed job comes from whoever printed, so `pwg.rs` checks
every header field before it sizes anything.
