import type { AICorrectionFieldResult } from "@/lib/graphql/extraction-eval";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type {
  ExtractionShadowResultStatus,
  ExtractionShadowVerdict,
} from "@trenova/graphql/generated/graphql";

/** How far back the shadow comparison is read. */
export const SHADOW_WINDOW_DAYS = 30;

export const MIN_SAMPLE_PERCENT = 1;
export const MAX_SAMPLE_PERCENT = 100;
export const MIN_DAILY_LIMIT = 1;
export const MAX_DAILY_LIMIT = 5000;

export const SHADOW_STATUS: Record<ExtractionShadowResultStatus, BadgeAttrProps> = {
  Pending: { phase: "queued", text: "Pending", description: "Waiting for the candidate to answer" },
  Completed: {
    phase: "complete",
    text: "Answered",
    description: "The candidate answered; it is scored once a person confirms the document",
  },
  Failed: { phase: "failed", text: "Failed", description: "The candidate could not answer" },
  Skipped: {
    phase: "closed",
    text: "Skipped",
    description: "Not run: the document changed, was deleted, or the budget was spent",
  },
};

export const SHADOW_VERDICT: Record<ExtractionShadowVerdict, BadgeAttrProps> = {
  Better: {
    phase: "complete",
    text: "Better",
    description: "The candidate read more fields right than production",
  },
  Worse: {
    phase: "failed",
    text: "Worse",
    description: "The candidate read fewer fields right than production",
  },
  Same: { phase: "closed", text: "Same", description: "Both read the same number of fields right" },
};

export type ShadowSettingsDraft = {
  enabled: boolean;
  providerId: string;
  samplePercent: string;
  dailyLimit: string;
};

export type ShadowSettingsProblems = {
  providerId?: string;
  samplePercent?: string;
  dailyLimit?: string;
};

const PROBLEM = {
  providerId: { message: "Choose the AI provider to shadow production with" },
  samplePercent: { message: "Sample between 1 and 100 percent of extractions" },
  dailyLimit: { message: "The daily limit must be between 1 and 5000 extractions" },
} as const;

function wholeNumberBetween(value: string, min: number, max: number): number | null {
  const trimmed = value.trim();
  if (!/^\d+$/.test(trimmed)) return null;
  const parsed = Number(trimmed);
  return parsed >= min && parsed <= max ? parsed : null;
}

/** The problems that stop the settings from being saved, as the server would report them. */
export function shadowSettingsProblems(draft: ShadowSettingsDraft): ShadowSettingsProblems {
  const problems: ShadowSettingsProblems = {};
  if (draft.enabled && draft.providerId === "") {
    problems.providerId = PROBLEM.providerId.message;
  }
  if (wholeNumberBetween(draft.samplePercent, MIN_SAMPLE_PERCENT, MAX_SAMPLE_PERCENT) === null) {
    problems.samplePercent = PROBLEM.samplePercent.message;
  }
  if (wholeNumberBetween(draft.dailyLimit, MIN_DAILY_LIMIT, MAX_DAILY_LIMIT) === null) {
    problems.dailyLimit = PROBLEM.dailyLimit.message;
  }
  return problems;
}

export function hasProblems(problems: ShadowSettingsProblems): boolean {
  return Object.values(problems).some((problem) => problem !== undefined);
}

export type ShadowFieldRow = {
  key: string;
  confirmed: string;
  candidate?: AICorrectionFieldResult;
  production?: AICorrectionFieldResult;
};

/**
 * Each field the two sides were scored on, side by side. A field one side left
 * empty is still listed, because a miss is part of the comparison.
 */
export function pairFieldResults(
  candidate: AICorrectionFieldResult[],
  production: AICorrectionFieldResult[],
): ShadowFieldRow[] {
  const rows = new Map<string, ShadowFieldRow>();
  const rowFor = (key: string) => {
    let row = rows.get(key);
    if (!row) {
      row = { key, confirmed: "" };
      rows.set(key, row);
    }
    return row;
  };

  for (const result of candidate) {
    const row = rowFor(result.key);
    row.candidate = result;
    row.confirmed ||= result.confirmed;
  }
  for (const result of production) {
    const row = rowFor(result.key);
    row.production = result;
    row.confirmed ||= result.confirmed;
  }

  return [...rows.values()];
}

/** Whether the two sides disagree on a field's outcome. */
export function outcomesDiffer(row: ShadowFieldRow): boolean {
  return row.candidate?.outcome !== row.production?.outcome;
}

/** The candidate's accuracy less production's, in whole percentage points; null without scores. */
export function accuracyGap(
  candidate: { accuracy: number; scored: number },
  production: { accuracy: number; scored: number },
): number | null {
  if (candidate.scored === 0 || production.scored === 0) return null;
  return Math.round((candidate.accuracy - production.accuracy) * 100);
}
