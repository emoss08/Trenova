# Trenova Cloud Control Plane

Trenova is split into two deployable boundaries:

- Public Trenova owns the modular-monolith TMS, freight and customer billing, platform catalog, local/community entitlement providers, no-op usage providers, UI entitlement display, and the signed HTTP control-plane client.
- `trenova-cloud` owns SaaS subscription billing, plans, entitlements, usage metering, licenses, instances, provisioning, and future managed services.

Self-hosted and local/community deployments can run without `trenova-cloud`. When the control plane is disabled, catalog entitlement checks use the local/community provider and usage checks use the no-op provider. When the control plane is enabled, all catalog entitlement and usage checks are remote.

## Deployment Configuration

Use `TRENOVA_DEPLOYMENT_MODE` for the deployment mode:

- `self_hosted`
- `cloud`
- `development`

The legacy `platform.mode` setting remains accepted for existing configuration files. `community` and `enterprise` are treated as backward-compatible aliases for self-hosted behavior.

Control-plane configuration:

- `TRENOVA_CONTROL_PLANE_ENABLED`
- `TRENOVA_CONTROL_PLANE_ENDPOINT`
- `TRENOVA_INSTANCE_ID`
- `TRENOVA_CONTROL_PLANE_API_KEY`
- `TRENOVA_CONTROL_PLANE_FAIL_OPEN_ON_ERROR`

Self-hosted instances that connect to the control plane must have a provisioned `TRENOVA_INSTANCE_ID` and `TRENOVA_CONTROL_PLANE_API_KEY`. Cloud-hosted Trenova uses the same signed HTTP contract internally.

## Signed Request Contract

Every control-plane request sends:

- `Authorization: Bearer <TRENOVA_CONTROL_PLANE_API_KEY>`
- `X-Trenova-Instance-ID`
- `X-Trenova-Timestamp`
- `X-Trenova-Body-SHA256`
- `X-Trenova-Signature`

The signature payload is:

```txt
METHOD + "\n" + PATH + "\n" + BODY_SHA256 + "\n" + TIMESTAMP
```

The signature is an HMAC-SHA256 hex digest. `trenova-cloud` verifies the signature with a constant-time comparison and rejects timestamps outside the replay window. Usage recording also requires an idempotency key so repeated write requests return the original result instead of double counting usage.

## Failure Behavior

Development deployments may fail open only when `TRENOVA_CONTROL_PLANE_FAIL_OPEN_ON_ERROR=true`. Self-hosted, production, and cloud deployments fail closed on control-plane errors.
