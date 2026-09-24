import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { CircleAlertIcon } from "lucide-react";
import { QUALITY_STALE_MS, formatShare, formatUsd } from "./quality-model";

/**
 * The organization's AI quality in five figures: what people thought of the
 * answers, how many they rated, how the agents score against their golden
 * sets, how many regressed, and what evaluation has spent this month.
 */
export function QualityFigures() {
  const t = useT();
  const overview = useQuery({ ...queries.agentQuality.overview(), staleTime: QUALITY_STALE_MS });

  if (overview.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <CircleAlertIcon />
        <AlertDescription>
          {t("How well agents are doing could not be loaded. Try again shortly.")}
        </AlertDescription>
      </Alert>
    );
  }

  if (!overview.data) {
    return <Skeleton className="h-16" aria-busy />;
  }

  const figures = overview.data;
  const hidden = t("Needs access to ratings");

  return (
    <KpiStrip aria-label={t("AI quality figures")}>
      <KpiStripItem
        label={t("Satisfaction")}
        value={figures.ratingsVisible ? formatShare(figures.satisfaction) : "—"}
        sub={
          figures.ratingsVisible
            ? t("{0, plural, one {last # day} other {last # days}}", figures.windowDays)
            : hidden
        }
      />
      <KpiStripItem
        label={t("Ratings")}
        value={figures.ratingsVisible ? figures.ratings : "—"}
        sub={figures.ratingsVisible ? t("thumbs up and down") : hidden}
      />
      <KpiStripItem
        label={t("Quality score")}
        value={formatShare(figures.qualityScore)}
        sub={t("{0, plural, one {# agent scored} other {# agents scored}}", figures.agentsScored)}
      />
      <KpiStripItem
        label={t("Regressions")}
        value={figures.regressions}
        sub={t(
          "{0, plural, one {# agent still regressed} other {# agents still regressed}}",
          figures.openRegressions,
        )}
        tone={figures.openRegressions > 0 ? "danger" : undefined}
      />
      <KpiStripItem
        label={t("Eval spend this month")}
        value={formatUsd(figures.evalSpendMonthUsd)}
        sub={t("of {0}", formatUsd(figures.monthlyBudgetUsd))}
        tone={
          Number(figures.evalSpendMonthUsd) >= Number(figures.monthlyBudgetUsd)
            ? "warning"
            : undefined
        }
      />
    </KpiStrip>
  );
}
