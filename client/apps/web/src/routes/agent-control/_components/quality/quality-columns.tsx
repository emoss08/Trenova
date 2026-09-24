import { Sparkline } from "@/components/kpi/sparkline";
import type {
  AgentQualityRow,
  AgentSuiteRun,
  AgentSuiteRunCase,
  AgentSuiteRunStatus,
} from "@/lib/graphql/agent-quality";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { readCaseChecks } from "./cases/case-checks";
import {
  SUITE_RUN_STATUS,
  formatDelta,
  formatShare,
  formatUsd,
  sparklineValues,
} from "./quality-model";

const RUN_TIME_FORMAT = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

const DELTA_TONE = {
  success: "text-success-foreground",
  danger: "text-danger-foreground",
  muted: "text-muted-foreground",
} as const;

export function SuiteRunStatusBadge({ status }: { status: AgentSuiteRunStatus }) {
  const t = useT();
  const attrs = SUITE_RUN_STATUS[status];

  return (
    <Badge
      variant={phaseTone(attrs.phase)}
      title={attrs.description ? t(attrs.description) : undefined}
    >
      {t(attrs.text)}
    </Badge>
  );
}

function LastRunCell({ run }: { run: AgentSuiteRun | null }) {
  const t = useT();
  if (!run) {
    return <span className="text-muted-foreground text-xs">{t("Never run")}</span>;
  }

  return (
    <div className="flex flex-col items-start gap-0.5">
      <div className="flex items-center gap-1.5">
        <SuiteRunStatusBadge status={run.status} />
        {run.regression ? <Badge variant="danger">{t("Regressed")}</Badge> : null}
      </div>
      <span className="text-muted-foreground text-xs">
        {formatUnixInUserTimezone(run.finishedAt ?? run.startedAt, RUN_TIME_FORMAT)}
      </span>
    </div>
  );
}

function QualityLineCell({ row }: { row: AgentQualityRow }) {
  const t = useT();
  const values = sparklineValues(row.qualityPoints);
  if (values.length === 0) {
    return <span className="text-muted-foreground text-xs">{t("No scored runs")}</span>;
  }

  return (
    <div className="flex items-center gap-2">
      <Sparkline
        data={values}
        color={row.openRegression ? "var(--chart-5)" : "var(--chart-1)"}
        width={80}
        height={22}
      />
      <span className="tabular-nums">{formatShare(row.qualityScore)}</span>
    </div>
  );
}

function SatisfactionCell({ row }: { row: AgentQualityRow }) {
  const t = useT();
  if (!row.ratingsVisible) {
    return <span className="text-muted-foreground text-xs">{t("Hidden")}</span>;
  }
  if (row.ratings === 0) {
    return <span className="text-muted-foreground text-xs">{t("No ratings")}</span>;
  }
  const delta = formatDelta(row.satisfactionDelta);

  return (
    <div className="flex flex-col">
      <span className="tabular-nums">{formatShare(row.satisfaction)}</span>
      <span className={cn("text-xs tabular-nums", DELTA_TONE[delta.tone])}>
        {delta.text === "—" ? t("No earlier ratings") : delta.text}
      </span>
    </div>
  );
}

/** Every agent, one line an agent: how people rate it and how it scores. */
export function agentQualityColumns(t: TranslateFn): ColumnDef<AgentQualityRow>[] {
  return [
    {
      id: "agent",
      header: t("Agent"),
      cell: ({ row }) => (
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate">{row.original.name}</span>
          {!row.original.enabled ? (
            <Badge variant="neutral" appearance="outline">
              {t("Off")}
            </Badge>
          ) : null}
        </div>
      ),
      meta: { cellClassName: "max-w-64" },
    },
    {
      id: "satisfaction",
      header: t("Satisfaction"),
      cell: ({ row }) => <SatisfactionCell row={row.original} />,
    },
    {
      id: "quality",
      header: t("Quality score"),
      cell: ({ row }) => <QualityLineCell row={row.original} />,
    },
    {
      id: "lastRun",
      header: t("Last run"),
      cell: ({ row }) => <LastRunCell run={row.original.lastSuiteRun} />,
    },
  ];
}

/** An agent's suite runs, newest first. */
export function suiteRunColumns(t: TranslateFn): ColumnDef<AgentSuiteRun>[] {
  return [
    {
      id: "started",
      header: t("Started"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatUnixInUserTimezone(row.original.startedAt, RUN_TIME_FORMAT)}
        </span>
      ),
    },
    {
      id: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <SuiteRunStatusBadge status={row.original.status} />
          {row.original.regression ? <Badge variant="danger">{t("Regressed")}</Badge> : null}
        </div>
      ),
    },
    {
      id: "score",
      header: t("Quality score"),
      cell: ({ row }) => (
        <span className="tabular-nums">{formatShare(row.original.qualityScore)}</span>
      ),
    },
    {
      id: "cases",
      header: t("Cases"),
      cell: ({ row }) => (
        <span className="text-muted-foreground tabular-nums">
          {t(
            "{0} passed, {1} failed, {2} skipped",
            row.original.casesPassed,
            row.original.casesFailed,
            row.original.casesSkipped,
          )}
        </span>
      ),
    },
    {
      id: "cost",
      header: t("Cost"),
      cell: ({ row }) => <span className="tabular-nums">{formatUsd(row.original.costUsd)}</span>,
    },
  ];
}

function judgeNote(value: unknown): string | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }
  const rationale = (value as Record<string, unknown>).rationale;

  return typeof rationale === "string" && rationale.trim() !== "" ? rationale : null;
}

function judgeScore(value: unknown): number | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }
  const score = (value as Record<string, unknown>).score;

  return typeof score === "number" && Number.isFinite(score) ? score : null;
}

export const readJudge = (value: unknown) => ({ note: judgeNote(value), score: judgeScore(value) });

/** One suite run's cases in the order they were asked. */
export function suiteCaseColumns(t: TranslateFn): ColumnDef<AgentSuiteRunCase>[] {
  return [
    {
      id: "ordinal",
      header: t("Case"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.suiteOrdinal ?? "—"}</span>,
      meta: { headerClassName: "w-14", cellClassName: "w-14" },
    },
    {
      id: "result",
      header: t("Result"),
      cell: ({ row }) => <CaseResultCell evaluation={row.original} />,
    },
    {
      id: "score",
      header: t("Score"),
      cell: ({ row }) => (
        <span className="tabular-nums">{formatShare(row.original.caseScore ?? null)}</span>
      ),
    },
    {
      id: "judge",
      header: t("Judge"),
      cell: ({ row }) => {
        const judge = readJudge(row.original.judge);

        return judge.score === null ? (
          <span className="text-muted-foreground text-xs">{t("Not judged")}</span>
        ) : (
          <span className="tabular-nums">{formatShare(judge.score)}</span>
        );
      },
    },
  ];
}

function CaseResultCell({ evaluation }: { evaluation: AgentSuiteRunCase }) {
  const t = useT();
  const checks = readCaseChecks(evaluation.checks);
  if (evaluation.status === "Skipped") {
    return <Badge variant="neutral">{t("Not asked")}</Badge>;
  }
  if (evaluation.status === "Failed") {
    return <Badge variant="danger">{t("Could not run")}</Badge>;
  }
  if (!checks) {
    return <Badge variant="info">{t("Waiting")}</Badge>;
  }
  if (checks.hardFailure) {
    return <Badge variant="danger">{t("Failed a hard check")}</Badge>;
  }

  return checks.passed ? (
    <Badge variant="success">{t("Passed")}</Badge>
  ) : (
    <Badge variant="warning">{t("Below the pass mark")}</Badge>
  );
}
