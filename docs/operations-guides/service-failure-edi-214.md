# Service Failure EDI 214

Service failures are system-detected operational exceptions. Manual creation remains disabled; users review, resolve, or void detected failures.

## Lifecycle Rules

- Review and resolve may generate an outbound X12 214 only when the shipment customer has an active outbound EDI partner, shipment status capability, and exactly one active outbound X12 214 partner document profile configured for Service Failure 214.
- Void never generates a Service Failure 214.
- If no matching mandatory profile exists, review and resolve are not blocked by EDI.
- If a mandatory profile matches and the preview payload is invalid or profile selection is ambiguous, review or resolve is blocked before persistence.
- Non-mandatory invalid payloads are logged or skipped and do not block the lifecycle transition.
- Duplicate lifecycle generation is scoped to tenant, transaction set 214, outbound direction, service failure id, and lifecycle trigger. Generic shipment-event 214 messages do not count as Service Failure 214 duplicates.

## Partner Settings Contract

`EDIPartnerDocumentProfile.PartnerSettings["serviceFailure214"]` is reserved for outbound X12 214 service failure lifecycle automation.

Missing or malformed settings disable Service Failure 214 generation. Missing `enabled` is treated as `false`.

Supported keys:

| Key | Type | Default | Meaning |
| --- | --- | --- | --- |
| `enabled` | boolean | `false` | Enables Service Failure 214 evaluation for the profile. |
| `sendOnReviewed` | boolean | `false` | Generate after a successful review lifecycle transition. |
| `sendOnResolved` | boolean | `false` | Generate after a successful resolve lifecycle transition. |
| `mandatoryOnReviewed` | boolean | `false` | Block review when the reviewed payload/profile is invalid. Also enables the reviewed trigger. |
| `mandatoryOnResolved` | boolean | `false` | Block resolve when the resolved payload/profile is invalid. Also enables the resolved trigger. |
| `statusCode` | string | empty | Optional profile status code override. Trimmed and uppercased. Does not imply sending. |
| `requireStatusReasonCode` | boolean | `false` | Requires a reason code for non-`SD` status codes. `SD` always requires a reason code. |
| `requireLocation` | boolean | `false` | Requires a location id or name in the 214 payload. |
| `requireStop` | boolean | `false` | Requires a shipment stop id in the 214 payload. |
| `requireProNumber` | boolean | `false` | Requires a PRO number in the 214 payload. |
| `requireBol` | boolean | `false` | Requires a BOL in the 214 payload. |
| `acceptedReasonCodes` | string array | empty | Optional accepted reason-code allow list. Values are trimmed and uppercased; empty means unrestricted. |

Status code precedence:

1. `ServiceFailure.X12StatusCodeOverride`
2. `partnerSettings.serviceFailure214.statusCode`
3. Service failure reason-code default status
4. Fallback `SD`

Status and reason values are normalized before validation and generation. `internalNotes` are never copied into payload references. Operational `Notes` may remain in references intentionally.

Example:

```json
{
  "serviceFailure214": {
    "enabled": true,
    "sendOnReviewed": true,
    "sendOnResolved": true,
    "mandatoryOnReviewed": false,
    "mandatoryOnResolved": false,
    "statusCode": "SD",
    "requireStatusReasonCode": true,
    "requireLocation": false,
    "requireStop": false,
    "requireProNumber": false,
    "requireBol": false,
    "acceptedReasonCodes": ["NS"]
  }
}
```
