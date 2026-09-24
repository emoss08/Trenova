import { SectionTable } from "@/components/data-table/section-table";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { Sparkline } from "@/components/kpi/sparkline";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  runAgentSuite,
  type AgentQualityDetail,
  type AgentSuiteRun,
  type AgentSuiteRunCase,
} from "@/lib/graphql/agent-quality";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleAlertIcon, PlayIcon, TriangleAlertIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";
import { CaseScore } from "./cases/case-score";
import {
  SuiteRunStatusBadge,
  readJudge,
  suiteCaseColumns,
  suiteRunColumns,
} from "./quality-columns";
import { formatShare, formatUsd, sparklineValues } from "./quality-model";
import { useQualityPages, QUALITY_STALE_MS } from "./use-quality-pages";
import { WorstRatedPanel } from "./worst-rated";

type AgentQualitySheetProps = {
  agentId: string | null;
  suiteRunId: string | null;
  onClose: () => void;
  onSelectRun: (suiteRunId: string | null) => void;
};

/**
 * One agent's quality, opened from its row: its figures and quality line, its
 * suite runs a page at a time, and for the run picked, what each case scored,
 * which checks held and what the judge said.
 */
export function AgentQualitySheet({
  agentId,
  suiteRunId,
  onClose,
  onSelectRun,
}: AgentQualitySheetProps) {
  const t = useT();
  const onOpenChange = useCallback(
    (open: boolean) => {
      if (!open) {
        onClose();
      }
    },
    [onClose],
  );

  return (
    <Sheet open={agentId !== null} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-3xl">
        {agentId ? (
          <AgentQualityBody agentId={agentId} suiteRunId={suiteRunId} onSelectRun={onSelectRun} />
        ) : (
          <SheetHeader className="border-border border-b px-5 py-4">
            <SheetTitle>{t("Agent quality")}</SheetTitle>
          </SheetHeader>
        )}
      </SheetContent>
    </Sheet>
  );
}

function AgentQualityBody({
  agentId,
  suiteRunId,
  onSelectRun,
}: {
  agentId: string;
  suiteRunId: string | null;
  onSelectRun: (suiteRunId: string | null) => void;
}) {
  const t = useT();
  const detail = useQuery({ ...queries.agentQuality.agent(agentId), staleTime: QUALITY_STALE_MS });

  return (
    <>
      <SheetHeader className="border-border border-b px-5 py-4">
        <SheetTitle>{detail.data?.agentName ?? t("Agent quality")}</SheetTitle>
        <SheetDescription>
          {t("How people rated its answers and how it scores against its golden set.")}
        </SheetDescription>
      </SheetHeader>
      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-4 px-5 py-4">
          {detail.isError ? (
            <Alert variant="destructive" size="sm">
              <CircleAlertIcon />
              <AlertDescription>{t("This agent's quality could not be loaded.")}</AlertDescription>
            </Alert>
          ) : detail.data ? (
            <AgentQualitySummary detail={detail.data} />
          ) : (
            <Skeleton className="h-24" aria-busy />
          )}
          <SuiteRunsPanel agentId={agentId} suiteRunId={suiteRunId} onSelectRun={onSelectRun} />
          {suiteRunId ? <SuiteCasesPanel suiteRunId={suiteRunId} /> : null}
          {detail.data?.ratingsVisible ? <WorstRatedPanel agentDefinitionId={agentId} /> : null}
        </div>
      </ScrollArea>
    </>
  );
}

function AgentQualitySummary({ detail }: { detail: AgentQualityDetail }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canRun } = usePermission(Resource.AgentEvalSuite, Operation.Create);
  const last = detail.lastSuiteRun;
  const values = sparklineValues(detail.qualityPoints);

  const run = useApiMutation<AgentSuiteRun, string>({
    mutationFn: (agentDefinitionId) => runAgentSuite(agentDefinitionId),
    onSuccess: async () => {
      toast.success(t("Suite run started"), {
        description: t("The agent answers its cases with every write simulated."),
      });
      await queryClient.invalidateQueries({ queryKey: queries.agentQuality._def });
    },
    resourceName: t("Suite run"),
  });

  return (
    <div className="flex flex-col gap-3">
      <KpiStrip aria-label={t("Agent quality figures")}>
        <KpiStripItem
          label={t("Satisfaction")}
          value={detail.ratingsVisible ? formatShare(detail.satisfaction) : "—"}
          sub={
            detail.ratingsVisible
              ? t("{0, plural, one {# rating} other {# ratings}}", detail.ratings)
              : t("Needs access to ratings")
          }
        />
        <KpiStripItem
          label={t("Quality score")}
          value={formatShare(last?.qualityScore ?? null)}
          sub={
            last?.baselineScore != null
              ? t("recent median {0}", formatShare(last.baselineScore))
              : t("no earlier runs to compare")
          }
          tone={last?.regression ? "danger" : undefined}
        />
        <KpiStripItem
          label={t("Active cases")}
          value={detail.activeCases}
          sub={t("in its golden set")}
        />
      </KpiStrip>

      <SectionPanel
        title={t("Quality over time")}
        hint={t("{0, plural, one {# scored run} other {# scored runs}}", values.length)}
        action={
          canRun ? (
            <Button
              size="xs"
              variant="outline"
              disabled={run.isPending || detail.activeCases === 0}
              onClick={() => run.mutate(detail.agentDefinitionId)}
            >
              <PlayIcon className="size-3.5" />
              {t("Run suite now")}
            </Button>
          ) : null
        }
      >
        {values.length === 0 ? (
          <SectionPanelQuiet>{t("No suite run has scored this agent yet.")}</SectionPanelQuiet>
        ) : (
          <div className="flex items-center gap-3 px-3 py-3">
            <Sparkline
              data={values}
              color={last?.regression ? "var(--chart-5)" : "var(--chart-1)"}
              width={320}
              height={48}
            />
          </div>
        )}
        {last ? (
          <div className="border-border flex flex-col gap-2 border-t px-3 py-2">
            <DescriptionList layout="inline">
              <DescriptionItem label={t("Last run")}>
                <SuiteRunStatusBadge status={last.status} />
              </DescriptionItem>
              <DescriptionItem label={t("Cost")}>{formatUsd(last.costUsd)}</DescriptionItem>
              <DescriptionItem label={t("What changed")}>{last.changeSummary}</DescriptionItem>
            </DescriptionList>
            {last.regression ? (
              <Alert variant="warning" size="sm">
                <TriangleAlertIcon />
                <AlertDescription>
                  {last.comments || t("The agent's score fell after it changed.")}
                </AlertDescription>
              </Alert>
            ) : last.comments ? (
              <p className="text-muted-foreground text-xs">{last.comments}</p>
            ) : null}
          </div>
        ) : null}
      </SectionPanel>
    </div>
  );
}

const suiteRunRowId = (run: AgentSuiteRun) => run.id;
const suiteCaseRowId = (evaluation: AgentSuiteRunCase) => evaluation.id;

function SuiteRunsPanel({
  agentId,
  suiteRunId,
  onSelectRun,
}: {
  agentId: string;
  suiteRunId: string | null;
  onSelectRun: (suiteRunId: string | null) => void;
}) {
  const t = useT();
  const { query, rows, pagination } = useQualityPages(`runs:${agentId}`, (page) =>
    queries.agentQuality.suiteRuns(agentId, page),
  );

  const columns = useMemo(
    () => [
      ...suiteRunColumns(t),
      {
        id: "open",
        header: () => <span className="sr-only">{t("Cases")}</span>,
        cell: ({ row }: { row: { original: AgentSuiteRun } }) => (
          <Button
            variant={row.original.id === suiteRunId ? "secondary" : "ghost"}
            size="xs"
            aria-pressed={row.original.id === suiteRunId}
            disabled={row.original.status === "Skipped"}
            onClick={() => onSelectRun(row.original.id === suiteRunId ? null : row.original.id)}
          >
            {t("Cases")}
          </Button>
        ),
      },
    ],
    [onSelectRun, suiteRunId, t],
  );

  return (
    <SectionPanel title={t("Suite runs")} count={pagination.totalCount ?? undefined}>
      <SectionTable
        label={t("Suite runs")}
        columns={columns}
        rows={rows}
        getRowId={suiteRunRowId}
        isLoading={query.isPending}
        isRefreshing={query.isPlaceholderData}
        error={query.isError ? t("Suite runs could not be loaded.") : null}
        onRetry={() => void query.refetch()}
        empty={t("This agent's suite has not run yet.")}
        pagination={pagination}
      />
    </SectionPanel>
  );
}

function CaseDetails({ evaluation }: { evaluation: AgentSuiteRunCase }) {
  const t = useT();
  const judge = readJudge(evaluation.judge);

  return (
    <div className="flex flex-col gap-2 px-3 py-2">
      <CaseScore checks={evaluation.checks} caseScore={evaluation.caseScore ?? null} />
      {judge.note ? (
        <DescriptionList layout="split">
          <DescriptionItem label={t("Judge's note")}>
            <span className="whitespace-pre-wrap">{judge.note}</span>
          </DescriptionItem>
        </DescriptionList>
      ) : null}
      {evaluation.errorMessage ? (
        <p className="text-muted-foreground text-xs">{evaluation.errorMessage}</p>
      ) : null}
      {evaluation.reply ? (
        <DescriptionList layout="split">
          <DescriptionItem label={t("Reply")}>
            <span className="line-clamp-6 whitespace-pre-wrap">{evaluation.reply}</span>
          </DescriptionItem>
        </DescriptionList>
      ) : null}
    </div>
  );
}

const renderCaseDetails = (evaluation: AgentSuiteRunCase) => (
  <CaseDetails evaluation={evaluation} />
);

function SuiteCasesPanel({ suiteRunId }: { suiteRunId: string }) {
  const t = useT();
  const columns = useMemo(() => suiteCaseColumns(t), [t]);
  const { query, rows, pagination } = useQualityPages(`cases:${suiteRunId}`, (page) =>
    queries.agentQuality.suiteRunCases(suiteRunId, page),
  );

  return (
    <SectionPanel title={t("Cases in this run")} count={pagination.totalCount ?? undefined}>
      <SectionTable
        label={t("Cases in this run")}
        columns={columns}
        rows={rows}
        getRowId={suiteCaseRowId}
        rowLabel={(evaluation) => t("Case {0}", evaluation.suiteOrdinal ?? "")}
        renderDetails={renderCaseDetails}
        isLoading={query.isPending}
        isRefreshing={query.isPlaceholderData}
        error={query.isError ? t("This run's cases could not be loaded.") : null}
        onRetry={() => void query.refetch()}
        empty={t("This run drew no cases.")}
        pagination={pagination}
      />
    </SectionPanel>
  );
}
