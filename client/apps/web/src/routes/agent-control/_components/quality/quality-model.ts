import type {
  AgentQualityControl,
  AgentSuiteRunStatus,
  UpdateAgentQualityControlInput,
} from "@/lib/graphql/agent-quality";
import type { AiFeedbackTargetType } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import { z } from "zod";

/** A section's figures move with the nightly sweep; a minute old is still true. */
export const QUALITY_STALE_MS = 60_000;

/**
 * Where a suite run sits in its lifecycle. A skipped run is closed rather
 * than failed: nothing about the agent changed, so there was nothing to ask.
 * A run the budget stopped scored what it could and needs someone to look at
 * the budget.
 */
export const SUITE_RUN_STATUS: Record<AgentSuiteRunStatus, BadgeAttrProps> = {
  Running: { phase: "active", text: "Running", description: "Replaying the agent's cases" },
  Completed: {
    phase: "complete",
    text: "Completed",
    description: "Every drawn case was asked and scored",
  },
  Skipped: {
    phase: "closed",
    text: "Skipped",
    description: "Nothing about the agent or its cases changed since its last run",
  },
  BudgetStopped: {
    phase: "attention",
    text: "Budget stopped",
    description: "The evaluation budget ran out before every case was asked",
  },
  Failed: { phase: "failed", text: "Failed", description: "The run could not finish" },
};

/** The suite run statuses a table filters by, in lifecycle order. */
export function suiteRunStatusChoices(t: TranslateFn): { value: string; label: string }[] {
  return (Object.keys(SUITE_RUN_STATUS) as AgentSuiteRunStatus[]).map((status) => ({
    value: status,
    label: t(SUITE_RUN_STATUS[status].text),
  }));
}

/** What a rated answer was, in the words of the place a person rated it. */
export const TARGET_TYPE_LABEL: Record<AiFeedbackTargetType, string> = {
  AssistantMessage: "Assistant answer",
  DelegatedAnswer: "Answer from another agent",
  Briefing: "Briefing",
  BriefingSection: "Briefing section",
  Insight: "Insight",
  WatchtowerItem: "Watchtower item",
};

export function targetTypeChoices(t: TranslateFn): { value: string; label: string }[] {
  return (Object.keys(TARGET_TYPE_LABEL) as AiFeedbackTargetType[]).map((type) => ({
    value: type,
    label: t(TARGET_TYPE_LABEL[type]),
  }));
}

/** A share from 0 to 1 as a whole percentage, or a dash when there is none. */
export function formatShare(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return "—";
  }

  return `${Math.round(value * 100)}%`;
}

export type DeltaTone = "success" | "danger" | "muted";

/**
 * A change in satisfaction as points, signed, with the tone a reader expects:
 * better is good news, worse is not, and a change under one point is noise.
 */
export function formatDelta(value: number | null | undefined): { text: string; tone: DeltaTone } {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return { text: "—", tone: "muted" };
  }

  const points = Math.round(value * 100);
  if (points === 0) {
    return { text: "±0 pts", tone: "muted" };
  }

  return points > 0
    ? { text: `+${points} pts`, tone: "success" }
    : { text: `${points} pts`, tone: "danger" };
}

/** A decimal dollar amount as the Decimal scalar carries it, shown with its cents. */
export function formatUsd(value: string | null | undefined): string {
  const parsed = Number(value ?? "0");
  if (!Number.isFinite(parsed)) {
    return "$0.00";
  }

  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(parsed);
}

/** The scores on an agent's quality line, oldest first, as the sparkline draws them. */
export function sparklineValues(points: { qualityScore: number }[]): number[] {
  return points.map((point) => Math.round(point.qualityScore * 1000) / 10);
}

/** Dollars as the Decimal scalar carries them, from the cents a money field holds. */
export function centsToDecimal(cents: number): string {
  return (Math.round(cents) / 100).toFixed(2);
}

export function decimalToCents(value: string): number {
  const parsed = Number(value);

  return Number.isFinite(parsed) ? Math.round(parsed * 100) : 0;
}

export const qualityControlSchema = z
  .object({
    enabled: z.boolean(),
    runHour: z.string().regex(/^(?:[0-9]|1[0-9]|2[0-3])$/, "Pick an hour of the day"),
    timezone: z.string().max(100),
    maxCasesPerAgent: z.number().int().min(1).max(500),
    nightlyBudgetCents: z.number().int().min(0).max(10_000_000),
    monthlyBudgetCents: z.number().int().min(0).max(10_000_000),
    judgeEnabled: z.boolean(),
    judgeSamplePercent: z.number().min(0).max(100),
    regressionThresholdPoints: z.number().min(1).max(100),
    minCases: z.number().int().min(1).max(500),
    forceRerunDays: z.number().int().min(1).max(90),
  })
  .refine((values) => values.monthlyBudgetCents >= values.nightlyBudgetCents, {
    path: ["monthlyBudgetCents"],
    message: "The monthly budget cannot be less than one night's budget",
  });

export type QualityControlFormValues = z.infer<typeof qualityControlSchema>;

/** What the settings form starts from: the saved controls, in the units a person edits. */
export function toFormValues(control: AgentQualityControl): QualityControlFormValues {
  return {
    enabled: control.enabled,
    runHour: String(control.runHourLocal),
    timezone: control.timezone,
    maxCasesPerAgent: control.maxCasesPerAgent,
    nightlyBudgetCents: decimalToCents(control.nightlyBudgetUsd),
    monthlyBudgetCents: decimalToCents(control.monthlyBudgetUsd),
    judgeEnabled: control.judgeEnabled,
    judgeSamplePercent: Math.round(control.judgeSampleRate * 100),
    regressionThresholdPoints: Math.round(control.regressionThreshold * 100),
    minCases: control.minCases,
    forceRerunDays: control.forceRerunDays,
  };
}

/** The saved controls from the form, at the version the form was opened on. */
export function toUpdateInput(
  values: QualityControlFormValues,
  version: number,
): UpdateAgentQualityControlInput {
  const timezone = values.timezone.trim();

  return {
    version,
    enabled: values.enabled,
    runHourLocal: Number(values.runHour),
    ...(timezone === "" ? {} : { timezone }),
    maxCasesPerAgent: values.maxCasesPerAgent,
    nightlyBudgetUsd: centsToDecimal(values.nightlyBudgetCents),
    monthlyBudgetUsd: centsToDecimal(values.monthlyBudgetCents),
    judgeEnabled: values.judgeEnabled,
    judgeSampleRate: values.judgeSamplePercent / 100,
    regressionThreshold: values.regressionThresholdPoints / 100,
    minCases: values.minCases,
    forceRerunDays: values.forceRerunDays,
  };
}

/** The hours of the night a sweep can start at, as a select lists them. */
export function runHourOptions(): { value: string; label: string }[] {
  return Array.from({ length: 24 }, (_, hour) => ({
    value: String(hour),
    label: `${String(hour).padStart(2, "0")}:00`,
  }));
}
