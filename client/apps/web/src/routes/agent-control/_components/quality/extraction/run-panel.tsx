import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  cancelExtractionEvalRun,
  EXTRACTION_ACCURACY_KEY,
  EXTRACTION_EVAL_RESULT_LIST_KEY,
  EXTRACTION_EVAL_RUN_DETAIL_KEY,
  EXTRACTION_EVAL_RUN_LIST_KEY,
  fetchExtractionEvalResults,
  fetchExtractionEvalRun,
  type ExtractionEvalRunRow,
} from "@/lib/graphql/extraction-eval";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleStopIcon } from "lucide-react";
import { toast } from "sonner";
import { formatShare, formatUsd } from "../quality-model";
import { RunStatusBadge } from "./extraction-badges";
import { ACTIVE_RUN_POLL_MS, accuracyTone, isRunActive } from "./extraction-model";
import { FieldAccuracyTable } from "./field-accuracy-table";
import { RunResults } from "./run-results";

/**
 * One evaluation run: how the model scored overall and per field, what it
 * cost, and each case's outcome. A run in flight is re-read until it ends.
 */
export function RunPanel({ open, onOpenChange, row }: DataTablePanelProps<ExtractionEvalRunRow>) {
  const t = useT();
  const queryClient = useQueryClient();
  const id = row?.id;
  const { allowed: canUpdate } = usePermission(Resource.AgentEvalSuite, Operation.Update);

  const detail = useQuery({
    queryKey: [EXTRACTION_EVAL_RUN_DETAIL_KEY, id],
    queryFn: ({ signal }) => fetchExtractionEvalRun(id ?? "", { signal }),
    enabled: open && id !== undefined,
    refetchInterval: (query) =>
      query.state.data && isRunActive(query.state.data.status) ? ACTIVE_RUN_POLL_MS : false,
  });
  const run = detail.data;
  const active = run ? isRunActive(run.status) : false;

  const results = useQuery({
    queryKey: [EXTRACTION_EVAL_RESULT_LIST_KEY, id],
    queryFn: ({ signal }) => fetchExtractionEvalResults(id ?? "", { signal }),
    enabled: open && id !== undefined,
    refetchInterval: active ? ACTIVE_RUN_POLL_MS : false,
  });

  const cancel = useApiMutation({
    mutationFn: () => cancelExtractionEvalRun(id ?? ""),
    onSuccess: async () => {
      toast.success(t("Evaluation canceled"), {
        description: t("Cases already scored are kept; the rest are skipped."),
      });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_RUN_DETAIL_KEY, id] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_RESULT_LIST_KEY, id] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_RUN_LIST_KEY] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_ACCURACY_KEY] }),
      ]);
    },
    resourceName: t("Evaluation run"),
  });

  const done = run ? run.casesCompleted + run.casesFailed + run.casesSkipped : 0;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={run ? t("{0} evaluation", run.providerName) : t("Evaluation run")}
      subtitle={run ? <RunStatusBadge value={run.status} t={t} /> : undefined}
      size="xl"
      headerActions={
        run && active && canUpdate ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            isLoading={cancel.isPending}
            onClick={() => cancel.mutate(undefined)}
          >
            <CircleStopIcon />
            {t("Cancel run")}
          </Button>
        ) : undefined
      }
    >
      {detail.isLoading || !run ? (
        <ComponentLoader />
      ) : (
        <div className="flex flex-col gap-4">
          {run.stopReason ? (
            <Alert variant={run.status === "BudgetStopped" ? "warning" : "info"} size="sm">
              <AlertDescription>{run.stopReason}</AlertDescription>
            </Alert>
          ) : null}
          {run.failureMessage ? (
            <Alert variant="destructive" size="sm">
              <AlertDescription>{run.failureMessage}</AlertDescription>
            </Alert>
          ) : null}
          <KpiStrip aria-label={t("Evaluation run figures")}>
            <KpiStripItem
              label={t("Field accuracy")}
              value={run.scoredCount > 0 ? formatShare(run.accuracy) : "—"}
              sub={t("{0, plural, one {# field scored} other {# fields scored}}", run.scoredCount)}
              tone={accuracyTone(run.accuracy, run.scoredCount)}
            />
            <KpiStripItem
              label={t("Cases")}
              value={t("{0} of {1}", done, run.casesTotal)}
              sub={t("{0} failed, {1} skipped", run.casesFailed, run.casesSkipped)}
              tone={run.casesFailed > 0 ? "warning" : undefined}
            />
            <KpiStripItem
              label={t("Model")}
              value={run.servedModel || run.providerModel || "—"}
              sub={run.providerName}
            />
            <KpiStripItem
              label={t("Avg latency")}
              value={run.avgLatencyMs > 0 ? t("{0} ms", run.avgLatencyMs) : "—"}
              sub={t("{0} in, {1} out tokens", run.inputTokens, run.outputTokens)}
            />
            <KpiStripItem
              label={t("Cost")}
              value={formatUsd(run.costUsd)}
              sub={t("from the evaluation budget")}
            />
          </KpiStrip>
          <SectionPanel title={t("Accuracy by field")} hint={t("worst first")}>
            {run.fieldAccuracy.length > 0 ? (
              <FieldAccuracyTable fields={run.fieldAccuracy} />
            ) : (
              <SectionPanelQuiet>
                {active ? t("Scores appear as cases finish.") : t("No field was scored.")}
              </SectionPanelQuiet>
            )}
          </SectionPanel>
          <SectionPanel title={t("Cases")} count={results.data?.length}>
            {results.isLoading ? <ComponentLoader /> : <RunResults results={results.data ?? []} />}
          </SectionPanel>
        </div>
      )}
    </DataTablePanelContainer>
  );
}
