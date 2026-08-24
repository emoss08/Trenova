import { ApiRequestError } from "@trenova/shared/lib/api";

export type PublicLinkErrorKind = "invalid" | "throttled" | "unavailable";

/**
 * Only a definitive 4xx rejection means the link itself is dead. A network
 * failure, a 5xx, or anything unrecognized is a transient server-side problem
 * and must not be presented as an expired link.
 */
export function publicLinkErrorKind(error: unknown): PublicLinkErrorKind {
  if (error instanceof ApiRequestError) {
    if (error.status === 429) return "throttled";
    if (error.status === 422 || error.isValidationError()) return "invalid";
    if (error.status >= 400 && error.status < 500) return "invalid";
  }
  return "unavailable";
}
