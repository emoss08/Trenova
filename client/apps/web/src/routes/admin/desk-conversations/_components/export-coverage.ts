import type { DeskTranscriptExportRow } from "@/lib/graphql/desk-transcripts";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { MAX_SELECTED_CONVERSATIONS } from "@/lib/desk-transcript-notices";
import { originLabel, timeframeLabel, type ConversationScope } from "./conversation-scope";

type Coverage = DeskTranscriptExportRow["coverage"];

/**
 * What an export covers, in words: "Last 30 days, everyone", "12 selected
 * conversations", "1 Mar 2026 to 31 Mar 2026, Dana Reyes, with Billing
 * exceptions". Built from the names the server kept when it was asked for.
 */
export function describeCoverage(t: TranslateFn, coverage: Coverage): string {
  if (coverage.kind === "Selection") {
    return t(
      "{0, plural, one {# selected conversation} other {# selected conversations}}",
      coverage.selectedCount,
    );
  }

  const when = coverage.timeframe ? timeframeLabel(t, coverage.timeframe) : t("All time");
  let words = t("{0}, {1}", when, coverage.personName || t("everyone"));
  if (coverage.agentName) {
    words = t("{0}, with {1}", words, coverage.agentName);
  }
  if (coverage.origins.length > 0) {
    words = t(
      "{0}, begun in {1}",
      words,
      coverage.origins.map((origin) => originLabel(t, origin)).join(", "),
    );
  }

  return words;
}

/** The people whose conversations the file holds: up to twenty by name, then how many more. */
export function describeOwners(t: TranslateFn, coverage: Coverage): string {
  if (coverage.ownerCount === 0) {
    return "";
  }
  const named = coverage.ownerNames.join(", ");
  const more = coverage.ownerCount - coverage.ownerNames.length;
  if (more <= 0) {
    return named;
  }

  return t("{0} and {1, plural, one {# other} other {# others}}", named, more);
}

/** The selection bar's action, naming how many transcripts it downloads. */
export function selectionLabel(t: TranslateFn, count: number): string {
  if (count > MAX_SELECTED_CONVERSATIONS) {
    return t("Select at most {0} to download", MAX_SELECTED_CONVERSATIONS);
  }

  return t("Download {0, plural, one {# transcript} other {# transcripts}}", count);
}

/** What a timeframe export will hold, said before it starts. */
export function exportSummary(
  t: TranslateFn,
  preview: { count: number; exact: boolean; limit: number } | undefined,
  scope: ConversationScope,
): string {
  const when = timeframeLabel(t, scope.timeframe);
  if (!preview) {
    return t("Counting the conversations in {0}…", when);
  }
  if (!preview.exact) {
    return t(
      "More than {0} conversations match. One export holds at most {0}; choose a shorter timeframe, one person or one agent.",
      preview.limit.toLocaleString(),
    );
  }
  if (preview.count === 0) {
    return t("No conversation matches. Choose a longer timeframe, or another person or agent.");
  }

  return t(
    "{0, plural, one {# conversation} other {# conversations}} will be exported as one zip: a Markdown transcript for each, and conversations.jsonl with all of them.",
    preview.count,
  );
}
