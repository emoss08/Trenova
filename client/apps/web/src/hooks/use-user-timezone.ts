import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";

/** The signed-in person's IANA time zone, with "auto" resolved to the browser's own. */
export function useUserTimezone(): string {
  return resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
}
