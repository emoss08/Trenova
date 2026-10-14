import type { AIAnalyticsSeries, AIAnalyticsTaskRow } from "@/lib/graphql/ai-analytics";
import { routeAITask } from "@/lib/graphql/ai-analytics";
import { aiAnalytics } from "@/lib/queries/ai-analytics";
import { aiProvider } from "@/lib/queries/ai-provider";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { AiAnalyticsCompareBy } from "@trenova/graphql/generated/graphql";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { memo, useState, type CSSProperties } from "react";
import { toast } from "sonner";
import { Ic } from "../kit/ic";
import { ConfirmDialog } from "../kit/modal";
import { completionIntensity, percent, type SeriesColors } from "./analytics-model";
import { AnalyticsCard } from "./card";
import { HeatHead } from "./heat-head";
import { highlightClass, useHighlight } from "./highlight";

type TaskFitCardProps = {
  tasks: readonly AIAnalyticsTaskRow[];
  series: readonly AIAnalyticsSeries[];
  colors: SeriesColors;
  compareBy: AiAnalyticsCompareBy;
  taskLabels: ReadonlyMap<string, string>;
  canRoute: boolean;
};

/**
 * Completed runs per task and series, counting only the series a task was routed to: where it
 * goes now, which does it best, and the provider to route it to when another is clearly better.
 */
export const TaskFitCard = memo(function TaskFitCard({
  tasks,
  series,
  colors,
  compareBy,
  taskLabels,
  canRoute,
}: TaskFitCardProps) {
  const t = useT();
  const { hot } = useHighlight();
  const queryClient = useQueryClient();
  const [confirming, setConfirming] = useState<AIAnalyticsTaskRow | null>(null);
  const labelOf = (row: AIAnalyticsTaskRow) => t(taskLabels.get(row.task) ?? row.task);
  const route = useMutation({
    mutationFn: (row: AIAnalyticsTaskRow) =>
      routeAITask(row.task, row.suggestion?.provider.id ?? ""),
    onSuccess: async (_, row) => {
      toast.success(t("{0} now goes to {1}", labelOf(row), row.suggestion?.provider.name ?? ""));
      setConfirming(null);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: aiAnalytics._def }),
        queryClient.invalidateQueries({ queryKey: aiProvider._def }),
      ]);
    },
    onError: (error) =>
      toast.error(error instanceof Error ? error.message : t("The task could not be routed")),
  });
  const keys = new Set(series.map((item) => item.key));
  const rows = tasks.filter((row) =>
    row.cells.some((cell) => keys.has(cell.seriesKey) && cell.runs > 0),
  );

  return (
    <AnalyticsCard
      title={
        compareBy === "Provider"
          ? t("Which provider fits which task")
          : t("Which model fits which task")
      }
      description={
        compareBy === "Provider"
          ? t("Completed runs per task, counting only the tasks each provider took.")
          : t("Completed runs per task, counting only the tasks each model was routed to.")
      }
      className="an-tf"
      right={
        <div className="an-lgd sm">
          <span className="an-rtk">
            <i />
            {t("Goes here now")}
          </span>
          <span>
            <Ic n="check" s={10} w={2.6} />
            {t("Best for the task")}
          </span>
        </div>
      }
    >
      {rows.length === 0 ? (
        <p className="an-nil">{t("No routed task ran in this scope.")}</p>
      ) : (
        <div className="an-hmw">
          <div
            className="an-hm fit"
            role="table"
            aria-label={t("Completed runs by task and series")}
            style={{ "--c": series.length } as CSSProperties}
          >
            <span className="an-hh" role="columnheader" />
            <HeatHead series={series} colors={colors} />
            {rows.map((row) => (
              <div key={row.task} role="row" className="an-hr">
                <span className="an-tn" role="rowheader" title={labelOf(row)}>
                  <b>{labelOf(row)}</b>
                  {row.suggestion ? (
                    <button
                      type="button"
                      className="an-rto ui-focus-ring"
                      disabled={!canRoute}
                      title={
                        canRoute
                          ? t("Route {0} to {1}", labelOf(row), row.suggestion.provider.name)
                          : t("Changing routing needs the right to edit providers")
                      }
                      onClick={() => setConfirming(row)}
                    >
                      <Ic n="arrowR" s={10} />
                      {t("Use {0}", row.suggestion.provider.shortName)}
                      <em className="mono t-k">+{Math.round(row.suggestion.gainPoints)}</em>
                    </button>
                  ) : (
                    row.current && <span>{t("Goes to {0}", row.current.name)}</span>
                  )}
                </span>
                {series.map((item) => {
                  const cell = row.cells.find((candidate) => candidate.seriesKey === item.key);
                  if (!cell || cell.runs === 0) {
                    return (
                      <span
                        key={item.key}
                        role="cell"
                        className={cn("an-c z", highlightClass(hot, item.key))}
                        aria-label={t("Not routed here")}
                      >
                        —
                      </span>
                    );
                  }
                  const current = row.currentSeriesKey === item.key;
                  const best = row.bestSeriesKey === item.key;
                  return (
                    <span
                      key={item.key}
                      role="cell"
                      className={cn(
                        "an-c ok",
                        cell.few && "few",
                        current && "rt",
                        best && "is-best",
                        highlightClass(hot, item.key),
                      )}
                      style={
                        {
                          "--i": cell.few ? 0 : completionIntensity(cell.completedRate),
                        } as CSSProperties
                      }
                      title={
                        current
                          ? t(
                              "{0} of {1} runs completed · routed here now",
                              formatNumber(cell.completed),
                              formatNumber(cell.runs),
                            )
                          : t(
                              "{0} of {1} runs completed",
                              formatNumber(cell.completed),
                              formatNumber(cell.runs),
                            )
                      }
                    >
                      {best && <Ic n="check" s={10} w={2.6} />}
                      <span className="mono">
                        {cell.few ? t("few") : percent(cell.completedRate)}
                      </span>
                    </span>
                  );
                })}
              </div>
            ))}
          </div>
        </div>
      )}
      {confirming?.suggestion && (
        <ConfirmDialog
          open
          onClose={() => setConfirming(null)}
          title={t(
            "Route {0} to {1}?",
            labelOf(confirming).toLowerCase(),
            confirming.suggestion.provider.name,
          )}
          description={
            confirming.current
              ? t(
                  "{0} becomes the first provider to take it, and {1} stops taking it. Every other task stays where it goes.",
                  confirming.suggestion.provider.name,
                  confirming.current.name,
                )
              : t(
                  "{0} becomes the first provider to take it. Every other task stays where it goes.",
                  confirming.suggestion.provider.name,
                )
          }
          confirmLabel={t("Route it")}
          busy={route.isPending}
          onConfirm={() => route.mutate(confirming)}
        />
      )}
    </AnalyticsCard>
  );
});
