import type { ExtractionRollout } from "@/lib/graphql/extraction-rollout";
import type { UpdateExtractionRolloutInput } from "@trenova/graphql/generated/graphql";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import { z } from "zod";

export const MIN_ROLLOUT_PERCENT = 1;
export const MAX_ROLLOUT_PERCENT = 100;
export const MIN_ACCURACY_DROP_POINTS = 1;
export const MAX_ACCURACY_DROP_POINTS = 50;
export const MIN_REJECTION_INCREASE_POINTS = 1;
export const MAX_REJECTION_INCREASE_POINTS = 100;

const PROBLEM = {
  providerId: { message: "Choose the AI provider to roll out" },
  percent: { message: "Send between 1 and 100 percent of documents to the candidate" },
  maxAccuracyDropPoints: { message: "The accuracy guard must allow between 1 and 50 points" },
  maxRejectionIncreasePoints: {
    message: "The rejection guard must allow between 1 and 100 points",
  },
} as const;

function wholeNumberBetween(min: number, max: number, message: string) {
  return z
    .string()
    .trim()
    .regex(/^\d+$/, { message })
    .transform(Number)
    .pipe(z.number().min(min, { message }).max(max, { message }));
}

export const rolloutDraftSchema = z
  .object({
    enabled: z.boolean(),
    providerId: z.string(),
    percent: wholeNumberBetween(MIN_ROLLOUT_PERCENT, MAX_ROLLOUT_PERCENT, PROBLEM.percent.message),
    maxAccuracyDropPoints: wholeNumberBetween(
      MIN_ACCURACY_DROP_POINTS,
      MAX_ACCURACY_DROP_POINTS,
      PROBLEM.maxAccuracyDropPoints.message,
    ),
    maxRejectionIncreasePoints: wholeNumberBetween(
      MIN_REJECTION_INCREASE_POINTS,
      MAX_REJECTION_INCREASE_POINTS,
      PROBLEM.maxRejectionIncreasePoints.message,
    ),
  })
  .refine((draft) => !draft.enabled || draft.providerId !== "", {
    message: PROBLEM.providerId.message,
    path: ["providerId"],
  });

export type RolloutDraft = z.input<typeof rolloutDraftSchema>;

export type RolloutProblems = Partial<
  Record<"providerId" | "percent" | "maxAccuracyDropPoints" | "maxRejectionIncreasePoints", string>
>;

export function rolloutDraftOf(rollout: ExtractionRollout): RolloutDraft {
  return {
    enabled: rollout.enabled,
    providerId: rollout.providerId ?? "",
    percent: String(rollout.percent),
    maxAccuracyDropPoints: String(rollout.maxAccuracyDropPoints),
    maxRejectionIncreasePoints: String(rollout.maxRejectionIncreasePoints),
  };
}

/** The problems that stop the rollout from being saved, as the server would report them. */
export function rolloutProblems(draft: RolloutDraft): RolloutProblems {
  const parsed = rolloutDraftSchema.safeParse(draft);
  if (parsed.success) return {};

  const problems: RolloutProblems = {};
  for (const issue of parsed.error.issues) {
    const field = issue.path[0] as keyof RolloutProblems;
    problems[field] ??= issue.message;
  }
  return problems;
}

/** The mutation input for a valid draft; null while the draft has problems. */
export function rolloutInput(
  draft: RolloutDraft,
  version: number,
): UpdateExtractionRolloutInput | null {
  const parsed = rolloutDraftSchema.safeParse(draft);
  if (!parsed.success) return null;

  return {
    enabled: parsed.data.enabled,
    providerId: parsed.data.providerId === "" ? null : parsed.data.providerId,
    percent: parsed.data.percent,
    maxAccuracyDropPoints: parsed.data.maxAccuracyDropPoints,
    maxRejectionIncreasePoints: parsed.data.maxRejectionIncreasePoints,
    version,
  };
}

export type RolloutState = "unset" | "off" | "serving" | "halted";

export function rolloutState(rollout: ExtractionRollout): RolloutState {
  if (rollout.haltedAt) return "halted";
  if (rollout.serving) return "serving";
  return rollout.providerId ? "off" : "unset";
}

export const ROLLOUT_STATE: Record<RolloutState, BadgeAttrProps> = {
  unset: { phase: "draft", text: "Not set up", description: "No candidate has been chosen" },
  off: { phase: "closed", text: "Off", description: "Every document is read by production" },
  serving: {
    phase: "active",
    text: "Serving",
    description: "The candidate reads its share of documents",
  },
  halted: {
    phase: "failed",
    text: "Stopped by a guard",
    description: "A guard stopped the rollout; every document is back on production",
  },
};

/** Whether a side has enough evidence for its guard to act. */
export function guardReady(scored: number, needed: number): boolean {
  return scored >= needed;
}
