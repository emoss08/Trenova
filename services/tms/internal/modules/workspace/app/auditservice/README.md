# auditservice

Writes and reads `audit_entries`. The design, retention, seals, exports and alerts are in
[docs/engineering/security-audit.md](../../../../../../../docs/engineering/security-audit.md).

## Writing an entry

```go
err := s.audit.LogAction(ctx, &auditports.LogActionParams{
	Resource:       permission.ResourceShipment,
	ResourceID:     updated.ID.String(),
	Operation:      permission.OpUpdate,
	UserID:         actor.UserID,
	PrincipalType:  actor.PrincipalType,
	PrincipalID:    actor.PrincipalID,
	CurrentState:   jsonutils.MustToJSON(updated),
	PreviousState:  jsonutils.MustToJSON(original),
	OrganizationID: updated.OrganizationID,
	BusinessUnitID: updated.BusinessUnitID,
}, auditservice.WithComment("Shipment updated"), auditservice.WithDiff(original, updated))
```

- Pass the request's `ctx`. The service reads the request ID, client IP, user agent, trace,
  channel and agent run from it; there is no option for any of them.
- Call it inside the write's transaction when there is one: the entry then commits or rolls
  back with the change. Outside one it is written on its own, after the change.
- `WithAction` names the event when `<resource>.<operation>` does not (`api_key.rotated`);
  `WithSecurity` marks a security entry (kept for the security class, readable only with
  `security_audit_log:read`, matched by the alert policies). Use
  `permissionports.SecurityAuditor.RecordChange` for access changes; it does both.
- An update with a diff stores only the diff. Secrets are left out and identifying numbers
  fingerprinted by field name (`Redactor`); name a field after what it holds.
- `LogActions` writes a batch in one statement.

A failed write returns an error and logs the entry in full. There is no buffer to fall back on.

## Files

| File | What |
|---|---|
| `service.go` | `LogAction`, `LogActions`, context enrichment, the write |
| `read.go` | list and get for the GraphQL resolvers |
| `redaction.go` | the field policy |
| `alerts.go` | matches committed entries against the alert policies and starts `AuditAlertWorkflow` |
| `realtime.go` | publishes committed entries to the audit log page |
| `agentcredit.go` | names the agent definition on an unattended run's entries |
| `security.go` | the `SecurityAuditor` implementation |
| `record.go`, `option.go` | the `Record` helper and the `LogOption`s |
