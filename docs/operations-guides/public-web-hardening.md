# Public Web Hardening

## Scope

This runbook covers the public Trenova web surfaces:

- `cloud.trenova.app`, served by the React SPA Cloudflare Worker.
- `api.trenova.app`, served by the TMS API.
- Self-hosted deployments using `deploy/Caddyfile`.

Both the SPA and the API additionally expose unauthenticated, token-addressed
carrier pages; those are covered in Public Carrier Token Surfaces below.

The marketing site at `trenova.app` is a separate Astro deployment and must
receive equivalent browser security headers in that deployment.

## Cloudflare SPA

- Deploy the frontend with `client/wrangler.jsonc`; the Worker runs before
  static assets and blocks sensitive or file-like misses before the SPA fallback.
- Keep the Worker CSP enforced. If a browser smoke test reveals a missing source,
  add only the narrow origin required by that integration.
- Production Cloudflare builds must set `VITE_API_URL=https://api.trenova.app/api/v1`
  so browser API traffic targets the API host instead of the SPA Worker.
- Verify after each production deploy:

```bash
curl -I https://cloud.trenova.app
curl -I https://cloud.trenova.app/logo.ico
curl -I https://cloud.trenova.app/metrics
curl -I https://cloud.trenova.app/openapi.json
```

Expected results:

- App and asset responses include `Strict-Transport-Security`,
  `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`,
  `Permissions-Policy`, and `Content-Security-Policy`.
- `/metrics`, `/openapi.json`, `/config.json`, missing source maps, and other
  sensitive-looking paths return `404`.

## TMS API

- API responses include security headers from Gin middleware.
- HSTS is emitted only in `staging` and `production` environments.
- Verify after deploy:

```bash
curl -I https://api.trenova.app/api/v1/version
curl -I https://api.trenova.app/api/v1/not-found
```

Expected results:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: no-referrer`
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`
- `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'; base-uri 'none'`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains` in staging
  and production.

## Public Carrier Token Surfaces

Two unauthenticated surfaces exist so an external carrier can act on an emailed
link without a Trenova login. Both are reachable from the public internet and
both are addressed only by a single-use opaque token.

SPA routes (no auth, no app chrome):

- `/tender-offer/:token` — offer preview.
- `/tender-offer/:token/accept` and `/tender-offer/:token/decline` — the same
  page reached from the email buttons. The suffix only highlights the matching
  action (and pre-expands the decline reason); it never auto-submits.
- `/rate-confirmation/:token` — rate confirmation preview and signing page.

API counterparts under `api.trenova.app`:

- `GET /api/v1/tender-offers/{token}/`
- `POST /api/v1/tender-offers/{token}/accept/`
- `POST /api/v1/tender-offers/{token}/decline/`
- `GET /api/v1/rate-confirmation-links/{token}/`
- `POST /api/v1/rate-confirmation-links/{token}/confirm/`

The signing surface lives under `/rate-confirmation-links` rather than
`/rate-confirmations`: the authenticated routes already bind
`/rate-confirmations/:rateConfirmationID`, and Gin rejects a second wildcard
name in the same segment position.

Hardening already in place:

- **Token redaction.** A router-level middleware rewrites the request URL
  before logging and tracing run, replacing both the `:token` path parameter
  and any `token` query value with `REDACTED`. Access logs and spans therefore
  never carry a usable credential.
- **Per-token rate limiting.** Each handler throttles per token — 10 requests
  per minute, burst 5 — on top of the coarse global per-IP limiter, so one
  leaked link cannot be hammered from many addresses. Over the limit the
  response is `429` with `Retry-After: 60`. Idle buckets are evicted after 30
  minutes and the bucket table is hard-capped at 10,000 entries with
  oldest-first eviction, so the limiter cannot itself be used to exhaust
  memory.
- **One vague error for every invalid-token mode.** Missing, malformed,
  expired, already used, revoked, and (for rate confirmations) voided all
  return the same `422` and the same message. Nothing about the token space is
  enumerable.
- **GET-pure previews.** Both preview endpoints are strictly read-only, so an
  email scanner or link prefetcher cannot consume an offer or a signature by
  fetching the link. Only the explicit `POST` actions mutate state, and each
  burns its token only after the underlying write succeeds.
- **No customer data.** Public views are built from the frozen offer or
  rate confirmation snapshot: lane, timing, equipment, and money — never the
  customer.

Verify after deploy:

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  https://api.trenova.app/api/v1/tender-offers/not-a-real-token/
curl -s -o /dev/null -w '%{http_code}\n' \
  https://api.trenova.app/api/v1/rate-confirmation-links/not-a-real-token/
curl -I https://cloud.trenova.app/tender-offer/not-a-real-token
```

Expected results:

- Both API requests return `422`. The offer endpoint answers
  `This offer link is no longer valid` and the signing endpoint answers
  `This rate confirmation link is no longer valid` — nothing distinguishes an
  unknown token from an expired, used, or revoked one.
- Repeating either request more than 10 times in a minute returns `429` with
  `Retry-After: 60`.
- The SPA route returns the app shell with the standard browser security
  headers; the invalid-token state is rendered client-side.
- The token never appears in access logs for any of the above.

## Google Maps Keys

In Google Cloud Console, use separate browser keys for local development and
production. Restrict production browser Maps keys with:

- Application restriction: `HTTP referrers (web sites)`.
- Website referrers: `https://cloud.trenova.app/*`.
- API restrictions: only the Google Maps Platform APIs used by the frontend.

Add local development origins only to a non-production key.

## Hetzner And Origin Access

- Keep SSH and administrative services Tailscale-only.
- Public inbound `80` and `443` should be reachable only from Cloudflare source
  ranges when the host is meant to receive proxied Cloudflare traffic.
- Source Cloudflare ranges from the official endpoints:
  - `https://www.cloudflare.com/ips-v4`
  - `https://www.cloudflare.com/ips-v6`
  - `https://api.cloudflare.com/client/v4/ips`
- Automate updates or schedule an operational review so allowlists do not drift.

## Self-Hosted Caddy

`deploy/Caddyfile` exposes only the frontend and `/api/*` reverse proxy by
default. Temporal UI, MinIO Console, and MinIO API routes are intentionally
absent from the public file.

Operators who need those admin tools can start from
`deploy/Caddyfile.admin.example`, which binds to `127.0.0.1` by default for
trusted SSH or Tailscale forwarding. Do not put that admin Caddyfile on public
ingress.
