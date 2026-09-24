import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { Sparkline } from "@/components/kpi/sparkline";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  runAgentSuite,
  type AgentQualityDetail,
  type AgentQualityRow,
  type AgentSuiteRun,
} from "@/lib/graphql/agent-quality";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleAlertIcon, PlayIcon, TriangleAlertIcon } from "lucide-react";
import { toast } from "sonner";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { SuiteRunStatusBadge } from "./quality-columns";
import { QUALITY_STALE_MS, formatShare, formatUsd, sparklineValues } from "./quality-model";

/**
 * One agent's quality, opened from its row: its figures and quality line, how
 * its last suite run went, and the way to its runs and the answers people
 * rated down, each a table of its own under Quality.
 */
export function AgentQualityPanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<AgentQualityRow>) {
  const t = useT();

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.name ?? t("Agent quality")}
      description={t("How people rated its answers and how it scores against its golden set.")}
      size="lg"
    >
      {row ? <AgentQualityBody agentId={row.agentDefinitionId} /> : null}
    </DataTablePanelContainer>
  );
}

function AgentQualityBody({ agentId }: { agentId: string }) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const { allowed: canReadRatings } = usePermission(Resource.AgentFeedback, Operation.Read);
  const detail = useQuery({ ...queries.agentQuality.agent(agentId), staleTime: QUALITY_STALE_MS });

  return (
    <div className="flex flex-col gap-4">
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
      <div className="flex flex-wrap gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => navigate({ tab: "quality", view: "runs", qualityAgent: agentId })}
        >
          {t("Its suite runs")}
        </Button>
        {canReadRatings ? (
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate({ tab: "quality", view: "ratings", qualityAgent: agentId })}
          >
            {t("Its worst-rated answers")}
          </Button>
        ) : null}
      </div>
    </div>
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
