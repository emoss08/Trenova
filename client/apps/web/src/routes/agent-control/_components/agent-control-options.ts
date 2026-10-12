import type { AgentControl } from "@/lib/graphql/agent-control";
import type { AgentControlInput } from "@trenova/graphql/generated/graphql";

export type ControlPatch = Partial<
  Pick<
    AgentControlInput,
    | "shadowMode"
    | "earnedAutonomy"
    | "promotionThreshold"
    | "aiTrainingConsent"
    | "personMonthlyMessages"
    | "learningOff"
  >
>;

/**
 * The input the mutation sends: the current switches with one of them changed.
 * Training consent is sent only when it is the switch being changed, so saving
 * any other switch never re-records who consented. The version loaded travels
 * with it, so a switch flipped over someone else's save is refused, not lost.
 */
export function controlInput(current: AgentControl, patch: ControlPatch): AgentControlInput {
  return {
    shadowMode: patch.shadowMode ?? current.shadowMode,
    earnedAutonomy: patch.earnedAutonomy ?? current.earnedAutonomy,
    promotionThreshold: patch.promotionThreshold ?? current.promotionThreshold,
    personMonthlyMessages: patch.personMonthlyMessages ?? current.personMonthlyMessages,
    learningOff: patch.learningOff ?? current.learningOff,
    version: current.version,
    ...(patch.aiTrainingConsent === undefined
      ? {}
      : { aiTrainingConsent: patch.aiTrainingConsent }),
  };
}

/**
 * How many clean approvals in a row earn a tool its next tier. A short list
 * keeps the choice legible; a value set some other way still shows up as the
 * pressed option rather than leaving the control blank.
 */
export const PROMOTION_THRESHOLD_PRESETS = [5, 10, 25, 50] as const;

export type ThresholdOption = { value: number; label: string };

/**
 * A short list of presets to choose from, with the current value added when it was set
 * some other way, so the control never shows nothing pressed. Sorted ascending.
 */
export function withPresets(presets: readonly number[], current: number): number[] {
  const values = new Set<number>(presets);
  if (Number.isFinite(current)) {
    values.add(current);
  }

  return [...values].sort((a, b) => a - b);
}

export function promotionThresholdOptions(current: number): ThresholdOption[] {
  return withPresets(
    PROMOTION_THRESHOLD_PRESETS,
    Number.isInteger(current) && current > 0 ? current : NaN,
  ).map((value) => ({ value, label: String(value) }));
}

/**
 * How many questions one person may ask the agents in a month. Nought is
 * unlimited and comes first, since it is where every organization starts.
 */
export const PERSON_ALLOWANCE_PRESETS = [0, 100, 250, 500, 1000] as const;

export function personAllowanceOptions(current: number): number[] {
  return withPresets(
    PERSON_ALLOWANCE_PRESETS,
    Number.isInteger(current) && current >= 0 ? current : NaN,
  );
}
