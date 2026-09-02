import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";

const FALLBACK = "Something went wrong. Please try again.";

/**
 * Extracts the message a person should read from a failed request: every
 * field-level validation message when the server sent some, otherwise the
 * problem detail, otherwise the error's own message.
 */
export function describeApiError(error: unknown, fallback: string = FALLBACK): string {
  if (error instanceof ApiRequestError || error instanceof GraphQLRequestError) {
    const normalized = error.normalize();
    const fieldMessages = normalized.fieldErrors
      .map((fieldError) => fieldError.message)
      .filter((message): message is string => Boolean(message));
    if (fieldMessages.length > 0) {
      return [...new Set(fieldMessages)].join(" ");
    }
    return normalized.detail || normalized.message || fallback;
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallback;
}
