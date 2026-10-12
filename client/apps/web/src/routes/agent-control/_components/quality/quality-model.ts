import type {
  AgentQualityControl,
  AgentQualityOverview,
  AgentSuiteRun,
  AgentSuiteRunCase,
  AgentSuiteRunStatus,
  UpdateAgentQualityControlInput,
} from "@/lib/graphql/agent-quality";
import { NEGATIVE_REASONS, POSITIVE_REASONS } from "@/components/ai-feedback/feedback-reasons";
import { readCaseChecks } from "./cases/case-checks";
import { centsToDecimal, decimalToCents } from "@/lib/decimal-cents";
import type { AiFeedbackTargetType } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import { z } from "zod";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";

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
export const TARGET_TYPE_LABEL: Record<AiFeedbackTargetType, string> = defineLabels({
  AssistantMessage: "Assistant answer",
  DelegatedAnswer: "Answer from another agent",
  Briefing: "Briefing",
  BriefingSection: "Briefing section",
  Insight: "Insight",
  WatchtowerItem: "Watchtower item",
});

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
    return { text: translate("±0 pts"), tone: "muted" };
  }

  return points > 0
    ? { text: translate("{0, plural, one {+# pt} other {+# pts}}", points), tone: "success" }
    : { text: translate("{0, plural, one {# pt} other {# pts}}", points), tone: "danger" };
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

export { centsToDecimal, decimalToCents };

export const qualityControlSchema = z
  .object({
    enabled: z.boolean(),
    runHour: z
      .string()
      .regex(/^(?:[0-9]|1[0-9]|2[0-3])$/, { error: () => translate("Pick an hour of the day") }),
    timezone: z.string().max(100),
    maxCasesPerAgent: z.number().int().min(1).max(500),
    nightlyBudgetCents: z.number().int().min(0).max(10_000_000),
    monthlyBudgetCents: z.number().int().min(0).max(10_000_000),
    judgeEnabled: z.boolean(),
    judgeSamplePercent: z.number().min(0).max(100),
    regressionThresholdPoints: z.number().min(1).max(100),
    minCases: z.number().int().min(1).max(500),
    forceRerunDays: z.number().int().min(1).max(90),
    version: z.number().int().min(0),
  })
  .refine((values) => values.monthlyBudgetCents >= values.nightlyBudgetCents, {
    path: ["monthlyBudgetCents"],
    error: () => translate("The monthly budget cannot be less than one night's budget"),
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
    version: control.version,
  };
}

/** The saved controls from the form, at the version the form was opened on. */
export function toUpdateInput(values: QualityControlFormValues): UpdateAgentQualityControlInput {
  const timezone = values.timezone.trim();

  return {
    version: values.version,
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

/**
 * How far a run's score sits from its agent's recent median, in whole points: negative
 * when it fell. Null when either is missing, so nothing is compared.
 */
export function pointsChange(run: Pick<AgentSuiteRun, "qualityScore" | "baselineScore"> | null) {
  if (!run || run.qualityScore == null || run.baselineScore == null) {
    return null;
  }

  return Math.round((run.qualityScore - run.baselineScore) * 100);
}

export type QualityHeroFacts = {
  /** The mean of each scored agent's latest score, as a share; null when none is scored. */
  score: number | null;
  /** The share of rated answers people liked; null when hidden or nobody rated. */
  liked: number | null;
  /** No agent has a golden set, so the sweep has nothing to replay. */
  noCases: boolean;
  /** The open regression that fell furthest, with how far. */
  worst: { agentId: string; agentName: string; points: number | null } | null;
};

/** What Nova says at the head of Quality, from the overview. */
export function qualityHeroFacts(overview: AgentQualityOverview): QualityHeroFacts {
  const worst = overview.worstRegression;

  return {
    score: overview.qualityScore ?? null,
    liked: overview.ratingsVisible && overview.ratings > 0 ? (overview.satisfaction ?? null) : null,
    noCases: overview.agentsWithCases === 0,
    worst: worst
      ? {
          agentId: worst.agentDefinitionId,
          agentName: worst.agentName,
          points: pointsChange(worst),
        }
      : null,
  };
}

export type CaseOutcome = "passed" | "failed" | "unasked";

export function caseOutcomeLabel(t: TranslateFn, outcome: CaseOutcome): string {
  switch (outcome) {
    case "passed":
      return t("Passed");
    case "failed":
      return t("Below the bar");
    case "unasked":
      return t("Not asked");
  }
}

/** One case of a suite run as a square: passed, failed (or below the bar), or not asked. */
export function caseOutcome(evaluation: Pick<AgentSuiteRunCase, "status" | "checks">): CaseOutcome {
  if (evaluation.status === "Failed") {
    return "failed";
  }
  if (evaluation.status !== "Completed") {
    return "unasked";
  }
  const checks = readCaseChecks(evaluation.checks);
  if (!checks) {
    return "unasked";
  }

  return checks.passed ? "passed" : "failed";
}

const REASON_LABEL = new Map<string, string>(
  [...NEGATIVE_REASONS, ...POSITIVE_REASONS].map((option) => [option.value, option.label]),
);

/** The reasons a person picked when rating an answer, in their words. */
export function ratingReasonLabels(t: TranslateFn, reasons: readonly string[]): string[] {
  return reasons.map((reason) => t(REASON_LABEL.get(reason) ?? reason));
}
