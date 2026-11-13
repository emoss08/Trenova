import type { AgentLearningRow } from "@/lib/graphql/agent-memories";
import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { reflectionSignalLabel } from "./reflection-signals";

/** The stretches the learning figures can be read over, in days. */
export const LEARNING_PERIODS = [7, 30, 90] as const;

export type LearningPeriod = (typeof LEARNING_PERIODS)[number];

/** How the memories of one row came to be kept, in a person's words. */
export function learningRowLabel(
  row: Pick<AgentLearningRow, "source" | "trigger">,
  t: TranslateFn,
): string {
  switch (row.source) {
    case "Reflection":
      return row.trigger ? reflectionSignalLabel(row.trigger, t) : t("Looking back, other");
    case "Agent":
      return t("Saved in a conversation when asked");
    case "User":
      return t("Written down by a person");
    case "Decision":
      return t("A reason given on a decision");
    case "Feedback":
      return t("Ratings on replies");
    default:
      return row.source;
  }
}

/**
 * The share of decided offers that were kept, as a whole percentage. Offers
 * still waiting and offers that lapsed were not decided, so they are left
 * out; null when nobody has decided any, and for what a person wrote down
 * themselves, which was never an offer.
 */
export function acceptedShare(
  row: Pick<AgentLearningRow, "source" | "kept" | "dismissed" | "retired">,
): number | null {
  if (row.source === "User") {
    return null;
  }
  const accepted = row.kept + row.retired;
  const decided = accepted + row.dismissed;
  if (decided === 0) {
    return null;
  }

  return Math.round((accepted / decided) * 100);
}

/** A figure, or a dash where the row has no such figure. */
export function figure(value: number, applies = true): string {
  return applies && value > 0 ? formatNumber(value) : "—";
}
