# TMS Security Remediation Tracker

This document tracks remediation work from the TMS backend audit. It is safe
for a public repository: it avoids secret values, customer data, infrastructure
coordinates, and exploit recipes.

## Scope

- Backend service under `services/tms`.
- Shared Go packages only where used by the TMS backend.
- Frontend, unrelated services, generated API specs, and local ignored config
  files are out of scope.

## Goals

- Improve security posture for SOC 2 readiness and common security-control
  expectations.
- Preserve existing public API behavior unless the behavior is itself unsafe.
- Add focused tests for each behavior change where practical.
- Keep changes small enough to review and verify independently.

Compliance cannot be achieved by code changes alone. SOC 2 and similar
security certifications also require operational controls, evidence collection,
vendor management, access reviews, incident response, change management,
backup/restore testing, and periodic risk assessment.

## Findings And Remediation

### API Server Timeouts

- Severity: Medium
- Risk: slow clients or long responses can hold handler and connection
  resources longer than intended.
- Remediation: wire configured read, write, idle, and shutdown timeouts into the
  HTTP server and add a read-header timeout.
- Verification: unit test the server timeout configuration.

### Multipart Upload Body Limits

- Severity: Medium
- Risk: large multipart requests can be parsed before business validation
  rejects file size.
- Remediation: enforce request body limits before multipart parsing and keep the
  existing service-level file validation as a second layer.
- Verification: handler-level tests for oversized multipart bodies.

### Rate Limiting

- Severity: Medium
- Risk: authentication endpoints are more exposed to brute force and credential
  stuffing without active throttling.
- Remediation: install rate-limiting middleware when enabled, with focused
  throttling for authentication routes.
- Verification: middleware tests for allowed and throttled requests.

### CSRF Protection For Cookie Authentication

- Severity: Medium
- Risk: unsafe methods authenticated by session cookie need a server-enforced
  request token.
- Remediation: require a CSRF token for unsafe methods when authentication uses
  a session cookie. Bearer/API-key requests remain tokenless.
- Verification: middleware tests for cookie-auth unsafe methods, safe methods,
  and bearer-auth requests.

### SSO Error Redirect Fallback

- Severity: Low/Medium
- Risk: an error redirect path can panic when no redirect origin is configured.
- Remediation: resolve a safe fallback origin before constructing the redirect.
- Verification: handler test for missing allowed origins on SSO error.

### Exchange-Rate HTTP Timeouts

- Severity: Medium
- Risk: upstream exchange-rate calls can block without an explicit client
  timeout when caller contexts lack deadlines.
- Remediation: use an injected or configured HTTP client with a bounded timeout
  and transport.
- Verification: unit test the default client timeout behavior.

### Inline HTML Document Viewing

- Severity: Low/Medium
- Risk: uploaded active content should not be rendered inline from storage URLs.
- Remediation: remove HTML from the default allowed upload MIME types and force
  attachment disposition for active content if already present.
- Verification: document-service tests for MIME defaults and view disposition.

## Verification Commands

Run checks package-by-package with low parallelism:

```bash
cd services/tms
go test -p 1 ./internal/api/... ./internal/core/services/documentservice ./internal/core/services/exchangerateservice
golangci-lint run --concurrency 1 ./internal/api/... ./internal/core/services/documentservice ./internal/core/services/exchangerateservice
```

Run broader checks only when local resources allow it.
