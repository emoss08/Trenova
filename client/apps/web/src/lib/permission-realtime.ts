import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";

export const PERMISSIONS_RESOURCE = "permissions";

/**
 * Whether a realtime event says the reader's permissions in their current
 * organization may have changed: a role, grant, inheritance or constraint
 * edit in the organization, or a role given to or taken from them.
 */
export function isPermissionChangeFor(
  event: ResourceInvalidationEvent | null,
  organizationId: string | undefined,
): boolean {
  if (!event || event.resource !== PERMISSIONS_RESOURCE) {
    return false;
  }
  return !organizationId || event.organizationId === organizationId;
}
