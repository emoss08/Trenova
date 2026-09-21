import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { Insight, InsightMetric, InsightSeverity } from "@/types/insight";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ArrowRightIcon, ClockAlertIcon, LightbulbIcon, XIcon } from "lucide-react";
import { Link } from "react-router";
import { WidgetCount, WidgetEmpty, WidgetShell, WidgetSkeleton } from "../widget-shell";
import type { WidgetProps } from "../widget-registry";
import {
  formatMetricChange,
  formatMetricValue,
  insightBody,
  isChangeAdverse,
  isStale,
  primaryMetrics,
  sortInsights,
} from "./insight-presentation";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";

const INSIGHT_LIMIT = 6;

/**
 * Findings computed from the organization's own records.
 *
 * Every number on this tile came from a query, not a model. Where a model was
 * involved it wrote the sentence underneath, and the card says so, because a
 * panel headed "insights" invites more trust than a table and the reader should
 * know which parts earned it.
 */
export function AIInsightsWidget({ widget }: WidgetProps) {
  const t = useT();

  const insightsQuery = useQuery(queries.insight.active(INSIGHT_LIMIT));
  const insights = sortInsights(insightsQuery.data?.results ?? []);

  // The clock is read once at mount rather than on every render: staleness is
  // measured against a horizon well over a day out, so a fresher reading would
  // not change any card, and reading it during render makes the tile's output
  // depend on when React happened to re-run it. Every card on the tile is judged
  // against the same instant, so two cards cannot disagree about whether the
  // last refresh was recent.
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  const criticalCount = insights.filter((entry) => entry.severity === "Critical").length;

  return (
    <WidgetShell
      title={widget.title || t("Operational Insights")}
      icon={LightbulbIcon}
      badge={criticalCount > 0 ? <WidgetCount value={criticalCount} tone="danger" /> : null}
      href="/insights"
      hrefLabel={t("All insights")}
    >
      {insightsQuery.isLoading ? (
        <WidgetSkeleton rows={3} />
      ) : insights.length === 0 ? (
        <WidgetEmpty icon={LightbulbIcon}>
          {t(
            "Nothing needs your attention. Findings appear here when service, billing, or compliance moves in the wrong direction.",
          )}
        </WidgetEmpty>
      ) : (
        <div className="flex flex-col gap-2">
          {insights.map((entry) => (
            <InsightCard key={entry.id} insight={entry} now={now} />
          ))}
        </div>
      )}
    </WidgetShell>
  );
}

function InsightCard({ insight, now }: { insight: Insight; now: number }) {
  const t = useT();
  const queryClient = useQueryClient();
  const body = insightBody(insight);
  const metrics = primaryMetrics(insight);
  const stale = isStale(insight, now);

  const dismissMutation = useApiMutation({
    mutationFn: () => apiService.insightService.dismiss(insight.id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queries.insight.active(INSIGHT_LIMIT).queryKey,
      });
    },
    resourceName: "Insight",
  });

  return (
    <div className="border-border/60 bg-card/50 group/insight rounded-md border p-2.5">
      <div className="flex items-start gap-2">
        <SeverityDot severity={insight.severity} />
        <Link
          to={`/insights?selected=${encodeURIComponent(insight.id)}`}
          className="min-w-0 flex-1"
        >
          <p className="text-xs leading-snug font-medium hover:underline">{insight.headline}</p>
          {insight.subject !== "" && (
            <p className="text-2xs text-muted-foreground mt-0.5 truncate">{insight.subject}</p>
          )}
        </Link>
        <Button
          variant="ghost"
          size="xs"
          className="size-5 shrink-0 p-0 opacity-0 transition-opacity group-hover/insight:opacity-100 focus-visible:opacity-100"
          onClick={() => dismissMutation.mutate(undefined)}
          disabled={dismissMutation.isPending}
          aria-label={t("Dismiss this insight")}
        >
          <XIcon className="size-3" />
        </Button>
      </div>

      {metrics.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1">
          {metrics.map((metric) => (
            <MetricReadout key={metric.key} metric={metric} />
          ))}
        </div>
      )}

      {body.text !== "" && (
        <p className="text-2xs text-muted-foreground mt-2 leading-relaxed">
          {body.text}
          {body.generated && <GeneratedMark />}
        </p>
      )}

      <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1">
        {(insight.links ?? []).map((link) => (
          <Link
            key={link.path}
            to={link.path}
            className="text-2xs text-muted-foreground hover:text-foreground inline-flex items-center gap-1 transition-colors"
          >
            {link.label}
            {link.count > 0 && <span className="tabular-nums">({link.count})</span>}
            <ArrowRightIcon className="size-2.5" />
          </Link>
        ))}
        {stale && <StaleMark />}
      </div>
    </div>
  );
}

/**
 * The numbers, shown larger than the prose around them.
 *
 * This is the part of the card that is certain, so it gets the visual weight.
 * A change is coloured by the metric's own direction rather than its sign:
 * rising revenue and rising detention are not the same news.
 */
function MetricReadout({ metric }: { metric: InsightMetric }) {
  const change = formatMetricChange(metric);
  const adverse = isChangeAdverse(metric);

  return (
    <div className="flex flex-col">
      <div className="flex items-baseline gap-1.5">
        <span className="font-table text-sm tabular-nums">{formatMetricValue(metric)}</span>
        {change && (
          <span
            className={cn("text-2xs tabular-nums", adverse ? "text-destructive" : "text-success")}
          >
            {change}
          </span>
        )}
      </div>
      <span className="text-2xs text-muted-foreground truncate">
        {metric.label}
        {metric.baseline && metric.baselineLabel !== "" && (
          <span className="opacity-70"> · vs {metric.baselineLabel}</span>
        )}
      </span>
    </div>
  );
}

/**
 * Marks the sentence a model wrote.
 *
 * The distinction matters precisely because the numbers beside it did not come
 * from one. Hiding it would let generated prose borrow the credibility of the
 * computed figures it sits next to.
 */
function GeneratedMark() {
  const t = useT();

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span className="text-muted-foreground/60 ml-1 inline-flex translate-y-px align-middle">
            <AssistMark className="size-2.5" />
          </span>
        }
      />
      <TooltipContent>
        {t(
          "This explanation was written by AI. The figures above were computed from your records.",
        )}
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * Says the numbers are old rather than hiding the card.
 *
 * A refresh that has not run is a different thing from a problem that has gone
 * away, and dropping the card would quietly report the second when the first is
 * true.
 */
function StaleMark() {
  const t = useT();

  return (
    <Badge variant="neutral" appearance="outline" className="gap-1 text-3xs">
      <ClockAlertIcon className="size-2.5" />
      {t("Numbers may be out of date")}
    </Badge>
  );
}

function SeverityDot({ severity }: { severity: InsightSeverity }) {
  const toneClass = {
    Critical: "bg-destructive",
    Warning: "bg-warning",
    Info: "bg-muted-foreground/40",
  }[severity];

  return <span className={cn("mt-1 size-1.5 shrink-0 rounded-full", toneClass)} />;
}
