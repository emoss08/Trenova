import { useT } from "@trenova/shared/i18n/use-t";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import {
  formatMetricChange,
  formatMetricValue,
  isChangeAdverse,
} from "@/routes/home/_components/widgets/insight-presentation";
import type { Insight, InsightSeverity, InsightSurface } from "@/types/insight";
import { CheckCircle2Icon, SparklesIcon } from "lucide-react";
import { m } from "motion/react";
import { Link } from "react-router";
import { PAGE_INSIGHTS_LIMIT, usePageInsights } from "./use-page-insights";

const SEVERITY_STYLES: Record<InsightSeverity, string> = {
  Critical: "bg-danger/15 text-danger-foreground",
  Warning: "bg-warning/15 text-warning-foreground",
  Info: "bg-info/15 text-info-foreground",
};

const SEVERITY_DOT: Record<InsightSeverity, string> = {
  Critical: "bg-danger",
  Warning: "bg-warning",
  Info: "bg-info",
};

export function insightHref(insight: Insight): string {
  return `/insights?selected=${encodeURIComponent(insight.id)}`;
}

/**
 * The findings that belong on this page, in the same card the page's other
 * panels wear.
 *
 * It renders nothing at all for a reader who may not see insights, and a quiet
 * "nothing needs attention" for one who may, because a dashboard that only
 * shows the panel when there is bad news teaches people to dread it.
 */
export function PageInsightsCard({
  surface,
  title,
  className,
}: {
  surface: InsightSurface;
  title?: string;
  className?: string;
}) {
  const t = useT();
  const { insights, isLoading, allowed } = usePageInsights(surface);

  if (!allowed) {
    return null;
  }

  return (
    <Card className={cn("gap-0 p-0", className)}>
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="flex items-center gap-1.5 text-sm font-medium">
          <SparklesIcon className="text-muted-foreground size-3.5" />
          {title ?? t("Insights")}
          {insights.length > 0 ? (
            <span className="bg-muted text-muted-foreground ml-1 rounded-full px-1.5 py-0.5 text-xs font-medium tabular-nums">
              {insights.length}
            </span>
          ) : null}
        </CardTitle>
        <Link
          to="/insights"
          className="text-muted-foreground hover:text-foreground text-xs hover:underline"
        >
          {t("All insights")}
        </Link>
      </CardHeader>
      <CardContent className="p-2">
        {isLoading ? (
          <div className="space-y-2 p-2">
            {Array.from({ length: 3 }).map((_, index) => (
              <Skeleton key={index} className="h-10 w-full" />
            ))}
          </div>
        ) : insights.length === 0 ? (
          <div className="text-muted-foreground flex h-32 flex-col items-center justify-center gap-2 text-sm">
            <CheckCircle2Icon className="size-5 text-success-foreground" />
            {t("Nothing needs attention right now")}
          </div>
        ) : (
          <PageInsightRows insights={insights} />
        )}
      </CardContent>
    </Card>
  );
}

/**
 * The rows without the card, for a host that supplies its own chrome — a
 * command-center panel, say. Each row is one finding: how loud it is, what it
 * says, and the one number that says why.
 */
export function PageInsightRows({ insights }: { insights: readonly Insight[] }) {
  return (
    <div className="divide-y">
      {insights.map((insight, index) => (
        <m.div
          key={insight.id}
          initial={{ opacity: 0, x: -6 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ duration: 0.25, delay: index * 0.03, ease: "easeOut" }}
        >
          <InsightRow insight={insight} />
        </m.div>
      ))}
    </div>
  );
}

function InsightRow({ insight }: { insight: Insight }) {
  const metric = insight.metrics?.[0];
  const change = metric ? formatMetricChange(metric) : null;
  const adverse = metric ? isChangeAdverse(metric) : false;

  return (
    <Link
      to={insightHref(insight)}
      className="hover:bg-muted/50 flex items-start gap-3 rounded-md px-2 py-2 transition-colors"
    >
      <span
        className={cn(
          "mt-0.5 inline-flex w-16 shrink-0 justify-center rounded-full px-1.5 py-0.5 text-xs font-medium",
          SEVERITY_STYLES[insight.severity],
        )}
      >
        {insight.severity}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline justify-between gap-2">
          <span className="line-clamp-2 text-xs leading-snug font-medium">{insight.headline}</span>
          {metric && (
            <span className="shrink-0 text-right text-xs font-semibold tabular-nums">
              {formatMetricValue(metric)}
              {change && (
                <span
                  className={cn(
                    "ml-1 text-xs font-medium",
                    adverse ? "text-destructive" : "text-success",
                  )}
                >
                  {change}
                </span>
              )}
            </span>
          )}
        </div>
        <div className="text-muted-foreground mt-0.5 flex items-center gap-1.5 text-xs">
          {insight.subject !== "" && <span className="truncate">{insight.subject}</span>}
          {metric && (
            <span className="truncate">
              {insight.subject !== "" ? "· " : ""}
              {metric.label}
            </span>
          )}
          {insight.narrated && <SparklesIcon className="size-2.5 shrink-0 opacity-60" />}
        </div>
      </div>
    </Link>
  );
}

/**
 * One line of findings for a page that has no room for a card.
 *
 * The dispatch console is a fixed-height workspace where every pane is
 * working space, so its findings arrive as chips on a single row and the row
 * disappears entirely when there are none. The full wording is in the tooltip
 * and one click away.
 */
export function PageInsightsStrip({
  surface,
  className,
}: {
  surface: InsightSurface;
  className?: string;
}) {
  const t = useT();
  const { insights, allowed } = usePageInsights(surface, PAGE_INSIGHTS_LIMIT);

  if (!allowed || insights.length === 0) {
    return null;
  }

  return (
    <div
      className={cn("flex min-w-0 items-center gap-1.5 overflow-x-auto", className)}
      aria-label={t("Insights for this page")}
    >
      <span className="text-muted-foreground inline-flex shrink-0 items-center gap-1 text-xs font-medium">
        <SparklesIcon className="size-3" />
        {t("Insights")}
      </span>
      {insights.map((insight) => (
        <InsightChip key={insight.id} insight={insight} />
      ))}
      <Link
        to="/insights"
        className="text-muted-foreground hover:text-foreground shrink-0 text-xs hover:underline"
      >
        {t("All insights")}
      </Link>
    </div>
  );
}

function InsightChip({ insight }: { insight: Insight }) {
  const metric = insight.metrics?.[0];

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Link
            to={insightHref(insight)}
            className="bg-muted/40 hover:bg-muted flex max-w-72 shrink-0 items-center gap-1.5 rounded-md border px-2 py-1 text-xs transition-colors"
          >
            <span
              aria-hidden
              className={cn("size-1.5 shrink-0 rounded-full", SEVERITY_DOT[insight.severity])}
            />
            <span className="truncate">
              {insight.subject !== "" ? insight.subject : insight.headline}
            </span>
            {metric && (
              <span className="shrink-0 font-semibold tabular-nums">
                {formatMetricValue(metric)}
              </span>
            )}
          </Link>
        }
      />
      <TooltipContent className="max-w-sm">{insight.headline}</TooltipContent>
    </Tooltip>
  );
}
