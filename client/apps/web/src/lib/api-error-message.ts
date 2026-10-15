import { translate } from "@trenova/shared/i18n/runtime";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";

/**
 * Extracts the message a person should read from a failed request: every
 * field-level validation message when the server sent some, otherwise the
 * problem detail, otherwise the error's own message.
 */
export function describeApiError(error: unknown, fallback?: string): string {
  const fallbackMessage = fallback ?? translate("Something went wrong. Please try again.");
  if (error instanceof ApiRequestError || error instanceof GraphQLRequestError) {
    const normalized = error.normalize();
    const fieldMessages = normalized.fieldErrors
      .map((fieldError) => fieldError.message)
      .filter((message): message is string => Boolean(message));
    if (fieldMessages.length > 0) {
      return [...new Set(fieldMessages)].join(" ");
    }
    return normalized.detail || normalized.message || fallbackMessage;
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallbackMessage;
}
