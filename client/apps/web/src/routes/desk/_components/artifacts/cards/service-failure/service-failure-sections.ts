import { Resource } from "@trenova/shared/types/permission";

/**
 * The service failure card's sections built from records other than the
 * failure, by the resource each reads: what has been done to it is the audit
 * log. Someone who may read the failure but not the audit log sees the card
 * without that section. Its own module, so the skeleton reads it without the
 * card's code.
 */
export const SERVICE_FAILURE_SECTIONS = {
  activity: Resource.AuditLog,
} as const;
