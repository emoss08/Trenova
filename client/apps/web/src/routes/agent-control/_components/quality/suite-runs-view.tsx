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
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import type { DataTablePanelProps, RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { ArrowLeftIcon, CircleAlertIcon, ListChecksIcon, TriangleAlertIcon } from "lucide-react";
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
import { QUALITY_STALE_MS, formatShare, formatUsd } from "./quality-model";

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
      queryKey={AGENT_SUITE_RUN_LIST_KEY}
      graphql={graphql}
      resource={Resource.AgentEvalSuite}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={SuiteRunPanel}
      enableCreateAction={false}
      enableReadOnlyPanel
      initialColumnVisibility={{ regression: false }}
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

/** One suite run, read-only, with the way to what each of its cases scored. */
function SuiteRunPanel({ open, onOpenChange, row }: DataTablePanelProps<AgentSuiteRunRow>) {
  const t = useT();
  const [agentId] = useQueryState(QUALITY_AGENT_PARAM, qualityAgentParser);
  const openCases = useOpenCases(agentId);

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.agentName ?? t("Suite run")}
      description={
        row ? formatUnixInUserTimezone(row.finishedAt ?? row.startedAt, RUN_TIME_FORMAT) : undefined
      }
      size="lg"
      footer={
        row && row.status !== "Skipped" ? (
          <Button size="sm" onClick={() => openCases(row.id)}>
            <ListChecksIcon className="size-3.5" />
            {t("See the cases")}
          </Button>
        ) : undefined
      }
    >
      {row ? <SuiteRunDetails run={row} /> : null}
    </DataTablePanelContainer>
  );
}

function SuiteRunDetails({ run }: { run: AgentSuiteRunRow }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-3">
      {run.regression ? (
        <Alert variant="warning" size="sm">
          <TriangleAlertIcon />
          <AlertDescription>
            {run.comments || t("The agent's score fell after it changed.")}
          </AlertDescription>
        </Alert>
      ) : null}
      <DescriptionList layout="stacked" columns={2}>
        <DescriptionItem label={t("Status")}>
          <SuiteRunStatusBadge status={run.status} />
        </DescriptionItem>
        <DescriptionItem label={t("Quality score")}>
          {formatShare(run.qualityScore)}
        </DescriptionItem>
        <DescriptionItem label={t("Recent median")}>
          {formatShare(run.baselineScore)}
        </DescriptionItem>
        <DescriptionItem label={t("Cost")}>{formatUsd(run.costUsd)}</DescriptionItem>
        <DescriptionItem label={t("Cases")}>
          {t(
            "{0} passed, {1} failed, {2} skipped",
            run.casesPassed,
            run.casesFailed,
            run.casesSkipped,
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Hard failures")}>{run.hardFailures}</DescriptionItem>
        <DescriptionItem label={t("What changed")} span="full">
          {run.changeSummary}
        </DescriptionItem>
        {run.comments && !run.regression ? (
          <DescriptionItem label={t("Comments")} span="full">
            {run.comments}
          </DescriptionItem>
        ) : null}
      </DescriptionList>
    </div>
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
            <CircleAlertIcon className="size-3.5" />
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
