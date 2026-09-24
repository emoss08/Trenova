import { Sparkline } from "@/components/kpi/sparkline";
import type {
  AgentQualityRow,
  AgentSuiteRun,
  AgentSuiteRunCase,
  AgentSuiteRunCaseRow,
  AgentSuiteRunRow,
  AgentSuiteRunStatus,
  AgentWorstRatedAnswer,
  AgentWorstRatedRow,
} from "@/lib/graphql/agent-quality";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { evaluationStatusChoices } from "../activity/agent-badges";
import { readCaseChecks } from "./cases/case-checks";
import {
  SUITE_RUN_STATUS,
  TARGET_TYPE_LABEL,
  formatDelta,
  formatShare,
  formatUsd,
  sparklineValues,
  suiteRunStatusChoices,
  targetTypeChoices,
} from "./quality-model";

const RUN_TIME_FORMAT = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

const RATED_FORMAT = { month: "short", day: "numeric" } as const;

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

/**
 * Every agent, one line an agent: how people rate it and how it scores. What
 * a column is filtered or sorted by is the server's field for it, so the
 * table's builders offer only what the server can answer.
 */
export function getAgentQualityColumns(
  t: TranslateFn,
  ratingsVisible: boolean,
): ColumnDef<AgentQualityRow>[] {
  return [
    {
      accessorKey: "name",
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
      size: 240,
      meta: {
        label: t("Agent"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "enabled",
      header: t("On"),
      cell: ({ row }) => (row.original.enabled ? t("Yes") : t("No")),
      size: 80,
      meta: {
        label: t("On"),
        apiField: "enabled",
        filterable: true,
        sortable: true,
        filterType: "boolean",
      },
    },
    {
      accessorKey: "satisfaction",
      header: t("Satisfaction"),
      cell: ({ row }) => <SatisfactionCell row={row.original} />,
      size: 150,
      meta: {
        label: t("Satisfaction"),
        apiField: "satisfaction",
        filterable: false,
        sortable: ratingsVisible,
      },
    },
    {
      accessorKey: "qualityScore",
      header: t("Quality score"),
      cell: ({ row }) => <QualityLineCell row={row.original} />,
      size: 170,
      meta: {
        label: t("Quality score"),
        apiField: "qualityScore",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "openRegression",
      header: t("Regressed"),
      cell: ({ row }) => (row.original.openRegression ? t("Yes") : t("No")),
      size: 110,
      meta: {
        label: t("Regressed"),
        apiField: "openRegression",
        filterable: true,
        sortable: true,
        filterType: "boolean",
      },
    },
    {
      id: "lastRun",
      accessorFn: (row) => row.lastSuiteRun?.status ?? null,
      header: t("Last run"),
      cell: ({ row }) => <LastRunCell run={row.original.lastSuiteRun ?? null} />,
      size: 190,
      meta: {
        label: t("Last run"),
        apiField: "lastRunStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: suiteRunStatusChoices(t),
        defaultFilterOperator: "eq",
      },
    },
  ];
}

/** Suite runs, newest first unless sorted. */
export function getSuiteRunColumns(t: TranslateFn): ColumnDef<AgentSuiteRunRow>[] {
  return [
    {
      accessorKey: "startedAt",
      header: t("Started"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatUnixInUserTimezone(row.original.startedAt, RUN_TIME_FORMAT)}
        </span>
      ),
      size: 160,
      meta: {
        label: t("Started"),
        apiField: "startedAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "agentName",
      header: t("Agent"),
      cell: ({ row }) => <span className="truncate">{row.original.agentName}</span>,
      size: 200,
      meta: {
        label: t("Agent"),
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <SuiteRunStatusBadge status={row.original.status} />
          {row.original.regression ? <Badge variant="danger">{t("Regressed")}</Badge> : null}
        </div>
      ),
      size: 190,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: suiteRunStatusChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "regression",
      header: t("Regressed"),
      cell: ({ row }) => (row.original.regression ? t("Yes") : t("No")),
      size: 110,
      meta: {
        label: t("Regressed"),
        apiField: "regression",
        filterable: true,
        sortable: true,
        filterType: "boolean",
      },
    },
    {
      accessorKey: "qualityScore",
      header: t("Quality score"),
      cell: ({ row }) => (
        <span className="tabular-nums">{formatShare(row.original.qualityScore)}</span>
      ),
      size: 130,
      meta: {
        label: t("Quality score"),
        apiField: "qualityScore",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "casesFailed",
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
      size: 220,
      meta: {
        label: t("Cases failed"),
        apiField: "casesFailed",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "changeSummary",
      header: t("What changed"),
      cell: ({ row }) => (
        <span className="text-muted-foreground line-clamp-2">{row.original.changeSummary}</span>
      ),
      size: 280,
      meta: {
        label: t("What changed"),
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "costUsd",
      header: t("Cost"),
      cell: ({ row }) => <span className="tabular-nums">{formatUsd(row.original.costUsd)}</span>,
      size: 110,
      meta: {
        label: t("Cost"),
        apiField: "costUsd",
        filterable: false,
        sortable: true,
      },
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

/** One suite run's cases, in the order they were asked unless sorted. */
export function getSuiteCaseColumns(t: TranslateFn): ColumnDef<AgentSuiteRunCaseRow>[] {
  return [
    {
      accessorKey: "suiteOrdinal",
      header: t("Case"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.suiteOrdinal ?? "—"}</span>,
      size: 80,
      meta: {
        label: t("Case"),
        apiField: "suiteOrdinal",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "status",
      header: t("Result"),
      cell: ({ row }) => <CaseResultCell evaluation={row.original} />,
      size: 190,
      meta: {
        label: t("Result"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: evaluationStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "caseScore",
      header: t("Score"),
      cell: ({ row }) => (
        <span className="tabular-nums">{formatShare(row.original.caseScore ?? null)}</span>
      ),
      size: 100,
      meta: {
        label: t("Score"),
        apiField: "caseScore",
        filterable: false,
        sortable: true,
      },
    },
    {
      id: "judge",
      accessorFn: (row) => readJudge(row.judge).score,
      header: t("Judge"),
      cell: ({ row }) => {
        const judge = readJudge(row.original.judge);

        return judge.score === null ? (
          <span className="text-muted-foreground text-xs">{t("Not judged")}</span>
        ) : (
          <span className="tabular-nums">{formatShare(judge.score)}</span>
        );
      },
      size: 110,
      meta: {
        label: t("Judge"),
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "model",
      header: t("Model"),
      cell: ({ row }) => (
        <span className="text-muted-foreground font-mono text-xs">{row.original.model || "—"}</span>
      ),
      size: 180,
      meta: {
        label: t("Model"),
        apiField: "model",
        filterable: true,
        sortable: true,
        filterType: "text",
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

/** What was asked, or a note that it was not kept. */
export function answerQuestion(answer: AgentWorstRatedAnswer): string {
  return answer.sample?.turnSnapshot?.question.trim() ?? "";
}

/** The answers people liked least, most disliked first unless sorted. */
export function getWorstRatedColumns(t: TranslateFn): ColumnDef<AgentWorstRatedRow>[] {
  return [
    {
      id: "question",
      accessorFn: (row) => answerQuestion(row),
      header: t("Question"),
      cell: ({ row }) => (
        <span className="line-clamp-2">
          {answerQuestion(row.original) || t("What was asked was not kept")}
        </span>
      ),
      size: 380,
      meta: {
        label: t("Question"),
        apiField: "question",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "agentName",
      header: t("Agent"),
      cell: ({ row }) => <span className="truncate">{row.original.agentName || "—"}</span>,
      size: 180,
      meta: {
        label: t("Agent"),
        apiField: "agentName",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "targetType",
      header: t("Answer type"),
      cell: ({ row }) => t(TARGET_TYPE_LABEL[row.original.targetType]),
      size: 170,
      meta: {
        label: t("Answer type"),
        apiField: "targetType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: targetTypeChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "negative",
      header: t("Down / up"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.original.negative} / {row.original.positive}
        </span>
      ),
      size: 110,
      meta: {
        label: t("Thumbs down"),
        apiField: "negative",
        filterable: true,
        sortable: true,
        filterType: "number",
      },
    },
    {
      accessorKey: "lastRatedAt",
      header: t("Last rated"),
      cell: ({ row }) => (
        <span className="text-muted-foreground tabular-nums">
          {formatUnixInUserTimezone(row.original.lastRatedAt, RATED_FORMAT)}
        </span>
      ),
      size: 120,
      meta: {
        label: t("Last rated"),
        apiField: "lastRatedAt",
        filterable: false,
        sortable: true,
      },
    },
  ];
}
