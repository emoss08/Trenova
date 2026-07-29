import {
  resolveUserTimeFormat,
  resolveUserTimezone,
  type UserDatePreferences,
} from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useMemo } from "react";

/**
 * The signed-in user's resolved date preferences, as a subscription. The
 * formatters in `@trenova/shared/lib/date` read the same values out of module
 * state, so a component only needs this hook when it has to *react* to a
 * preference change — rendering a clock, or re-keying a subtree so already
 * painted timestamps are reformatted the moment the user switches zones.
 */
export function useUserDatePreferences(): UserDatePreferences {
  const timezone = useAuthStore((state) => state.user?.timezone);
  const timeFormat = useAuthStore((state) => state.user?.timeFormat);

  return useMemo(
    () => ({
      timezone: resolveUserTimezone(timezone),
      timeFormat: resolveUserTimeFormat(timeFormat),
    }),
    [timezone, timeFormat],
  );
}

export function useUserDatePreferenceKey(): string {
  const { timezone, timeFormat } = useUserDatePreferences();
  return `${timezone}|${timeFormat}`;
}
