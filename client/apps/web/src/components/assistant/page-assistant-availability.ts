import { describeApiError } from "@/lib/api-error-message";
import { ApiRequestError } from "@trenova/shared/lib/api";

const UNAVAILABLE_FALLBACK = "The assistant could not be opened on this page. Try again later.";

/**
 * Whether a page's assistant can be used, and if not, why, in the words a
 * person can act on. The route asks for assistant create first, then access to
 * the page's agent, then that agent being on; each is its own refusal.
 */
export type PageAssistantAvailability =
  | { state: "checking" }
  | { state: "ready" }
  | { state: "no-permission" }
  | { state: "no-access"; message: string }
  | { state: "unavailable"; message: string };

export function pageAssistantAvailability({
  permissionsLoading,
  canAsk,
  error,
}: {
  permissionsLoading: boolean;
  canAsk: boolean;
  error: unknown;
}): PageAssistantAvailability {
  if (permissionsLoading) {
    return { state: "checking" };
  }
  if (!canAsk) {
    return { state: "no-permission" };
  }
  if (error === null || error === undefined) {
    return { state: "ready" };
  }

  const message = describeApiError(error, UNAVAILABLE_FALLBACK);
  if (error instanceof ApiRequestError && error.status === 403) {
    return { state: "no-access", message };
  }

  return { state: "unavailable", message };
}
