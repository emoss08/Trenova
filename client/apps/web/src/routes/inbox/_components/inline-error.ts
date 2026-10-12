import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describeError } from "@trenova/shared/lib/error-presentation";

/**
 * A refusal as one line to show where the action was taken: the server's
 * own words when it wrote some for people (a rule it would not save, an
 * address already in use), a plain sentence otherwise. Transport and crash
 * text is never shown.
 */
export function inlineErrorText(t: TranslateFn, error: unknown): string {
  const description = describeError(error);
  if (description.detail !== undefined && description.detail.trim() !== "") {
    return description.detail;
  }
  switch (description.kind) {
    case "offline":
    case "network":
      return t("Could not reach Trenova. Check the connection and try again.");
    case "forbidden":
      return t("You don't have permission to do that.");
    default:
      return t("That didn't go through. Try again.");
  }
}
