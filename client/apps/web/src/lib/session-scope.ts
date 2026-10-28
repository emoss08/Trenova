import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";

/**
 * Whose session this is: the signed-in person in their current organization, or null when
 * there is no one. Anything the browser keeps for a session is read and written only for the
 * scope signed in at that moment, so one person's never shows for another, nor one
 * organization's for the next.
 */
export function sessionScope(
  user: Pick<User, "id" | "currentOrganizationId"> | null,
): string | null {
  if (!user?.id || !user.currentOrganizationId) return null;
  return `${user.id}:${user.currentOrganizationId}`;
}

/** The scope signed in right now. */
export function activeSessionScope(): string | null {
  return sessionScope(useAuthStore.getState().user);
}
