import type { WatchtowerSeverity } from "@/lib/graphql/watchtower";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * The class that tints a row's tile and the card's dot by severity. Severity
 * is an ordering, so it is drawn as a tone — danger, warning, then neutral —
 * rather than a categorical accent: a reader scanning the queue is sorting by
 * loudness whether they mean to or not.
 */
export function severityClass(severity: WatchtowerSeverity): string {
  return `dk-s-${severity}`;
}

/** The severity in the reader's words, for those who cannot see the colour. */
export function severityLabel(t: TranslateFn, severity: WatchtowerSeverity): string {
  switch (severity) {
    case "Critical":
      return t("Critical");
    case "Warning":
      return t("Warning");
    default:
      return t("Info");
  }
}

/** Every severity, loudest first: the order a filter offers them in. */
export const SEVERITIES: readonly WatchtowerSeverity[] = ["Critical", "Warning", "Info"];

// i18n-ignore: class names, never shown to a person
const MARK_CLASS: Record<WatchtowerSeverity, string> = {
  Critical: "bg-danger",
  Warning: "bg-dsk-warn",
  Info: "bg-dsk-blue",
};

/** The fill of the small square that says a severity: on a row, a filter, a token. */
export function severityMarkClass(severity: WatchtowerSeverity): string {
  return MARK_CLASS[severity];
}
