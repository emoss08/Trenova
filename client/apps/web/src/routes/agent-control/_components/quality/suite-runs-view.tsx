import { DataTable } from "@/components/data-table/data-table";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import {
  AGENT_SUITE_RUN_CASE_LIST_KEY,
  AGENT_SUITE_RUN_LIST_KEY,
  createAgentSuiteRunCaseTableGraphQLConfig,
  createAgentSuiteRunTableGraphQLConfig,
  type AgentSuiteRunCaseRow,
  type AgentSuiteRunRow,
} from "@/lib/graphql/agent-quality";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Callout } from "../edit/fields";
import { Ic } from "../kit/ic";
import { Tile } from "../kit/marks";
import { ReadSheet } from "../kit/read-sheet";
import { KV, Pts } from "../kit/values";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import type { DataTablePanelProps, RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { AlertCircleIcon, ArrowLeftIcon, ListChecksIcon } from "@trenova/shared/components/icons";
import { useQueryState } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  QUALITY_AGENT_PARAM,
  QUALITY_SUITE_RUN_PARAM,
  qualityAgentParser,
  qualitySuiteRunParser,
} from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { AgentScope } from "./agent-scope";
import { CaseScore } from "./cases/case-score";
import {
  SuiteRunStatusBadge,
  getSuiteCaseColumns,
  getSuiteRunColumns,
  readJudge,
} from "./quality-columns";
import {
  QUALITY_STALE_MS,
  caseOutcome,
  caseOutcomeLabel,
  formatShare,
  formatUsd,
  pointsChange,
  type CaseOutcome,
} from "./quality-model";

const RUN_TIME_FORMAT = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

/**
 * Every suite run, or one agent's, newest first; a run opens to what it
 * scored, and its cases replace the runs until the way back is taken. One
 * table shows at a time, because both keep their paging and filters in the
 * same place in the address.
 */
export default function SuiteRunsView() {
  const [agentId] = useQueryState(QUALITY_AGENT_PARAM, qualityAgentParser);
  const [suiteRunId] = useQueryState(QUALITY_SUITE_RUN_PARAM, qualitySuiteRunParser);

  return suiteRunId ? (
    <SuiteRunCases suiteRunId={suiteRunId} agentId={agentId} />
  ) : (
    <SuiteRunsTable agentId={agentId} />
  );
}

function useOpenCases(agentId: string | null) {
  const navigate = useAIControlNavigation();

  return useCallback(
    (suiteRunId: string) =>
      navigate({ tab: "quality", view: "runs", qualityAgent: agentId, suiteRun: suiteRunId }),
    [agentId, navigate],
  );
}

function SuiteRunsTable({ agentId }: { agentId: string | null }) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const openCases = useOpenCases(agentId);
  const columns = useMemo(() => getSuiteRunColumns(t), [t]);
  const graphql = useMemo(() => createAgentSuiteRunTableGraphQLConfig(agentId), [agentId]);

  const contextMenuActions: RowAction<AgentSuiteRunRow>[] = [
    {
      id: "cases",
      label: t("See the cases"),
      icon: ListChecksIcon,
      onClick: (row) => openCases(row.original.id),
      hidden: (row) => row.original.status === "Skipped",
    },
  ];

  const table = (
    <DataTable<AgentSuiteRunRow>
      name="Suite Run"
      emptyTitle={t("No suite runs yet")}
      queryKey={AGENT_SUITE_RUN_LIST_KEY}
      graphql={graphql}
      resource={Resource.AgentEvalSuite}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={SuiteRunPanel}
      enableCreateAction={false}
      enableReadOnlyPanel
      initialColumnVisibility={{
        regression: false,
        change: false,
        costUsd: false,
        changeSummary: false,
      }}
    />
  );

  if (!agentId) {
    return table;
  }

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <AgentScope agentId={agentId} onClear={() => navigate({ tab: "quality", view: "runs" })} />
      {table}
    </div>
  );
}

/** One suite run, read-only: what it scored, against what, and a square for every case. */
function SuiteRunPanel({ open, onOpenChange, row }: DataTablePanelProps<AgentSuiteRunRow>) {
  const t = useT();
  const [agentId] = useQueryState(QUALITY_AGENT_PARAM, qualityAgentParser);
  const openCases = useOpenCases(agentId);
  const run = open ? row : null;

  return (
    <ReadSheet
      open={run !== null}
      onClose={() => onOpenChange(false)}
      label={run ? t("Suite run · {0}", run.agentName) : t("Suite run")}
      head={
        run && (
          <>
            <Tile agent={{ id: run.agentDefinitionId, name: run.agentName }} s={36} />
            <div className="sh-t">
              <b>{t("Suite run · {0}", run.agentName)}</b>
              <span>
                {formatUnixInUserTimezone(run.finishedAt ?? run.startedAt, RUN_TIME_FORMAT)}
              </span>
            </div>
            <SuiteRunStatusBadge status={run.status} />
          </>
        )
      }
    >
      {run && (
        <>
          <SuiteRunDetails run={run} />
          {run.status !== "Skipped" && (
            <div className="ad-bar sh-f">
              <span className="sp" />
              <button type="button" className="xa" onClick={() => openCases(run.id)}>
                <Ic n="table" s={13} />
                {t("See the cases")}
              </button>
            </div>
          )}
        </>
      )}
    </ReadSheet>
  );
}

function SuiteRunDetails({ run }: { run: AgentSuiteRunRow }) {
  const t = useT();
  const asked = run.casesPassed + run.casesFailed;

  return (
    <>
      {run.regression && (
        <div className="sh-p">
          <Callout tone="w">
            {run.comments || t("The agent's score fell after it changed.")}
          </Callout>
        </div>
      )}
      <KV
        items={[
          [t("Score"), formatShare(run.qualityScore)],
          [t("Change"), <Pts key="change" value={pointsChange(run)} />],
          [t("Cases asked"), t("{0} of {1}", asked, run.casesTotal)],
          [t("Cost"), formatUsd(run.costUsd)],
          [t("Recent median"), formatShare(run.baselineScore)],
          [t("Hard failures"), run.hardFailures],
          [t("What changed"), run.changeSummary],
          run.comments && !run.regression ? [t("Comments"), run.comments] : null,
        ]}
      />
      <div className="sh-p">
        <h4 className="sh-k">{t("Cases")}</h4>
        {run.status === "Skipped" ? (
          <p className="ad-h">
            {t("Nothing about the agent or its cases changed since its last run.")}
          </p>
        ) : (
          <CaseSquares run={run} />
        )}
      </div>
    </>
  );
}

const SQUARE_CLASS: Record<CaseOutcome, string | undefined> = {
  passed: undefined,
  failed: "f",
  unasked: "z",
};

function CaseSquares({ run }: { run: AgentSuiteRunRow }) {
  const t = useT();
  const cases = useQuery({
    ...queries.agentQuality.suiteRunCases(run.id),
    staleTime: QUALITY_STALE_MS,
  });

  if (cases.isError) {
    return <p className="ad-h">{t("This run's cases could not be loaded.")}</p>;
  }
  if (!cases.data) {
    return <p className="ad-h">{t("Loading…")}</p>;
  }

  return (
    <>
      <div className="cases" role="list" aria-label={t("Cases")}>
        {cases.data.map((evaluation, index) => {
          const outcome = caseOutcome(evaluation);
          const ordinal = evaluation.suiteOrdinal ?? index + 1;
          return (
            <i
              key={evaluation.id}
              role="listitem"
              className={SQUARE_CLASS[outcome]}
              title={t("Case {0} · {1}", ordinal, caseOutcomeLabel(t, outcome))}
              aria-label={t("Case {0} · {1}", ordinal, caseOutcomeLabel(t, outcome))}
            />
          );
        })}
      </div>
      <p className="ad-h">
        {run.status === "BudgetStopped"
          ? t("The evaluation budget ran out before every case was asked.")
          : t("Each square is a golden case; red ones scored below the bar.")}
      </p>
    </>
  );
}

function SuiteRunCases({ suiteRunId, agentId }: { suiteRunId: string; agentId: string | null }) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const run = useQuery({
    ...queries.agentQuality.suiteRun(suiteRunId),
    staleTime: QUALITY_STALE_MS,
  });
  const columns = useMemo(() => getSuiteCaseColumns(t), [t]);
  const graphql = useMemo(
    () => createAgentSuiteRunCaseTableGraphQLConfig(suiteRunId),
    [suiteRunId],
  );

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="flex min-w-0 items-center gap-2">
        <Button
          variant="ghost"
          size="xs"
          onClick={() => navigate({ tab: "quality", view: "runs", qualityAgent: agentId })}
        >
          <ArrowLeftIcon className="size-3.5" />
          {t("Back to suite runs")}
        </Button>
        {run.isError ? (
          <span className="text-muted-foreground flex items-center gap-1 text-sm">
            <AlertCircleIcon className="size-3.5" />
            {t("This run could not be loaded.")}
          </span>
        ) : run.data ? (
          <span className="flex min-w-0 items-center gap-2 text-sm">
            <span className="truncate">
              {t(
                "{0}, {1}",
                run.data.agentName,
                formatUnixInUserTimezone(run.data.startedAt, RUN_TIME_FORMAT),
              )}
            </span>
            <SuiteRunStatusBadge status={run.data.status} />
          </span>
        ) : run.isPending ? (
          <Skeleton className="h-4 w-48" aria-busy />
        ) : (
          <span className="text-muted-foreground text-sm">{t("This run no longer exists.")}</span>
        )}
      </div>
      <DataTable<AgentSuiteRunCaseRow>
        name="Suite Run Case"
        emptyTitle={t("No suite run cases yet")}
        queryKey={AGENT_SUITE_RUN_CASE_LIST_KEY}
        graphql={graphql}
        resource={Resource.AgentEvalSuite}
        columns={columns}
        TablePanel={SuiteCasePanel}
        enableCreateAction={false}
        enableReadOnlyPanel
        initialColumnVisibility={{ model: false }}
      />
    </div>
  );
}

/** What one case scored, which checks held and what the judge said. */
function SuiteCasePanel({ open, onOpenChange, row }: DataTablePanelProps<AgentSuiteRunCaseRow>) {
  const t = useT();
  const judge = readJudge(row?.judge);

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row ? t("Case {0}", row.suiteOrdinal ?? "") : t("Case")}
      size="lg"
    >
      {row ? (
        <div className="flex flex-col gap-3">
          <CaseScore checks={row.checks} caseScore={row.caseScore ?? null} />
          {judge.note ? (
            <DescriptionList layout="split">
              <DescriptionItem label={t("Judge's note")}>
                <span className="whitespace-pre-wrap">{judge.note}</span>
              </DescriptionItem>
            </DescriptionList>
          ) : null}
          {row.errorMessage ? (
            <p className="text-muted-foreground text-xs">{row.errorMessage}</p>
          ) : null}
          {row.reply ? (
            <DescriptionList layout="split">
              <DescriptionItem label={t("Reply")}>
                <span className="whitespace-pre-wrap">{row.reply}</span>
              </DescriptionItem>
            </DescriptionList>
          ) : null}
        </div>
      ) : null}
    </DataTablePanelContainer>
  );
}
