import { useT } from "@trenova/shared/i18n/use-t";
import { insightHref } from "@/components/insights/page-insights";
import { usePageInsights } from "@/components/insights/use-page-insights";
import { formatMetricValue } from "@/routes/home/_components/widgets/insight-presentation";
import type { Insight, InsightSeverity } from "@/types/insight";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { ExternalLinkIcon, SparklesIcon } from "lucide-react";
import { Link } from "react-router";
import { ModuleCard } from "./module-card";

const SEVERITY_DOT: Record<InsightSeverity, string> = {
  Critical: "bg-destructive",
  Warning: "bg-warning",
  Info: "bg-info",
};

function InsightRow({ insight, withDivider }: { insight: Insight; withDivider: boolean }) {
  const metric = insight.metrics?.[0];

  return (
    <div
      className={cn(
        "flex items-start justify-between gap-2 px-0.5 py-1.5",
        withDivider && "border-border border-t",
      )}
    >
      <div className="flex min-w-0 items-start gap-1.5">
        <span
          aria-hidden
          className={cn("mt-1 size-1.5 shrink-0 rounded-full", SEVERITY_DOT[insight.severity])}
        />
        <div className="min-w-0">
          <Link
            to={insightHref(insight)}
            className="line-clamp-2 text-[11px] leading-snug font-semibold hover:underline"
          >
            {insight.headline}
          </Link>
          {insight.subject !== "" && (
            <p className="text-muted-foreground truncate text-[10px]">{insight.subject}</p>
          )}
        </div>
      </div>
      {metric && (
        <span className="font-table text-muted-foreground shrink-0 text-[9.5px] tabular-nums">
          {formatMetricValue(metric)}
        </span>
      )}
    </div>
  );
}

function InsightSkeletonRow({ withDivider }: { withDivider: boolean }) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-2 px-0.5 py-2",
        withDivider && "border-border border-t",
      )}
    >
      <Skeleton className="h-3 w-36" />
      <Skeleton className="h-3 w-10" />
    </div>
  );
}

/**
 * The findings about service the customer feels, beside the loads they are
 * about. A dispatcher reading the exceptions inbox is the person who can do
 * something about a customer whose on-time rate is slipping.
 */
export function InsightsWatch({ enabled = true }: { enabled?: boolean }) {
  const t = useT();
  const { insights, isLoading, allowed } = usePageInsights("Shipments");

  let body: React.ReactNode;
  if (!allowed) {
    body = (
      <p className="text-muted-foreground px-4 py-6 text-center text-[10.5px]">
        {t("You do not have permission to read insights.")}
      </p>
    );
  } else if (enabled && isLoading) {
    body = (
      <>
        <InsightSkeletonRow withDivider={false} />
        <InsightSkeletonRow withDivider />
        <InsightSkeletonRow withDivider />
      </>
    );
  } else if (insights.length === 0) {
    body = (
      <div className="cc-fade-in flex flex-col items-center gap-2 px-4 py-6 text-center">
        <span className="bg-success/15 text-success inline-flex size-8 items-center justify-center rounded-full">
          <SparklesIcon className="size-4" />
        </span>
        <p className="text-[11.5px] font-medium">{t("Nothing needs attention right now")}</p>
        <p className="text-muted-foreground max-w-55 text-[10.5px] leading-snug">
          {t("Findings appear here when a customer's service moves in the wrong direction.")}
        </p>
      </div>
    );
  } else {
    body = (
      <>
        {insights.map((insight, index) => (
          <InsightRow key={insight.id} insight={insight} withDivider={index > 0} />
        ))}
      </>
    );
  }

  return (
    <ModuleCard
      id="insights"
      title={t("Insights")}
      count={allowed && !isLoading ? insights.length : undefined}
      countTone={insights.some((insight) => insight.severity === "Critical") ? "danger" : "muted"}
      rightSlot={
        <Button
          variant="ghost"
          size="icon-xxs"
          aria-label={t("All insights")}
          nativeButton={false}
          render={<Link to="/insights" />}
        >
          <ExternalLinkIcon className="size-2.5" />
        </Button>
      }
    >
      {body}
    </ModuleCard>
  );
}
