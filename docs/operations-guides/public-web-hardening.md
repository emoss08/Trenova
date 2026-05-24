# Public Web Hardening

## Scope

This runbook covers the public Trenova web surfaces:

- `cloud.trenova.app`, served by the React SPA Cloudflare Worker.
- `api.trenova.app`, served by the TMS API.
- Self-hosted deployments using `deploy/Caddyfile`.

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
