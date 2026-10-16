import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { DictationIssue } from "./use-dictation";

/**
 * Why dictation stopped without the person stopping it, in the few words the
 * notice above the composer holds.
 */
export function dictationIssueText(t: TranslateFn, issue: DictationIssue): string {
  switch (issue) {
    case "denied":
      return t("The microphone is blocked for this site.");
    case "blocked":
      return t("This page can't use the microphone.");
    case "service":
    case "no-service":
      return t("Dictation isn't available in this browser");
    case "no-microphone":
      return t("No microphone was found.");
    case "microphone-busy":
      return t("Another app is using the microphone.");
    case "network":
      return t("Dictation lost its connection.");
    case "no-speech":
      return t("Dictation stopped after a long silence.");
    case "language":
      return t("Dictation doesn't support your language.");
    case "interrupted":
      return t("Something else took the microphone.");
    case "failed":
      return t("Dictation stopped unexpectedly.");
  }
}
