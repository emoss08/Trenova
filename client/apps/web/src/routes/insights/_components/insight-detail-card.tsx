import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { Insight, InsightMetric } from "@/types/insight";
import {
  formatMetricChange,
  formatMetricValue,
  insightBody,
  isChangeAdverse,
  isStale,
} from "@/routes/home/_components/widgets/insight-presentation";
import {
  ArrowRightIcon,
  ClockAlertIcon,
  LightbulbIcon,
  RotateCcwIcon,
  SparklesIcon,
  XIcon,
} from "lucide-react";
import { Link } from "react-router";
import { CATEGORY_LABELS, SEVERITY_TONE } from "./insight-labels";

/**
 * One finding in full.
 *
 * The widget shows a headline and two numbers because a home tile is scanned.
 * This is the version for someone who has come to look: every metric the
 * detector computed, what it was compared against, the recommendation, the
 * period the numbers describe, and where the wording came from.
 */
export function InsightDetailCard({
  insight,
  now,
  onChanged,
  onOpen,
}: {
  insight: Insight;
  now: number;
  onChanged: () => void;
  onOpen: () => void;
}) {
  const t = useT();
  const body = insightBody(insight);
  const metrics = insight.metrics ?? [];
  const stale = isStale(insight, now);

  const dismissMutation = useApiMutation({
    mutationFn: () => apiService.insightService.dismiss(insight.id),
    onSuccess: onChanged,
    resourceName: "Insight",
  });

  const restoreMutation = useApiMutation({
    mutationFn: () => apiService.insightService.restore(insight.id),
    onSuccess: onChanged,
    resourceName: "Insight",
  });

  return (
    <div className="border-border bg-card hover:border-border/80 rounded-lg border p-4 transition-colors">
      <div className="flex items-start justify-between gap-3">
        {/* The headline is the affordance rather than the whole card: the card
            already holds links into the records, and a click target wrapping
            those would swallow them. */}
        <button type="button" onClick={onOpen} className="min-w-0 flex-1 cursor-pointer text-left">
          <div className="mb-1.5 flex flex-wrap items-center gap-1.5">
            <Badge variant={SEVERITY_TONE[insight.severity]}>{insight.severity}</Badge>
            <Badge variant="neutral" appearance="outline">{t(CATEGORY_LABELS[insight.category])}</Badge>
            {insight.status !== "Active" && (
              <Badge variant="neutral">{statusLabel(insight, t)}</Badge>
            )}
            {stale && (
              <Badge variant="neutral" appearance="outline" className="gap-1">
                <ClockAlertIcon className="size-2.5" />
                {t("Numbers may be out of date")}
              </Badge>
            )}
          </div>
          <h2 className="text-sm leading-snug font-semibold">{insight.headline}</h2>
          {insight.subject !== "" && (
            <p className="text-muted-foreground text-xs">{insight.subject}</p>
          )}
        </button>

        {insight.status === "Active" ? (
          <Button
            variant="outline"
            size="xs"
            onClick={() => dismissMutation.mutate(undefined)}
            disabled={dismissMutation.isPending}
          >
            <XIcon className="size-3" />
            {t("Dismiss")}
          </Button>
        ) : insight.status === "Dismissed" ? (
          <Button
            variant="outline"
            size="xs"
            onClick={() => restoreMutation.mutate(undefined)}
            disabled={restoreMutation.isPending}
          >
            <RotateCcwIcon className="size-3" />
            {t("Restore")}
          </Button>
        ) : null}
      </div>

      {metrics.length > 0 && (
        <div className="border-border/60 mt-3 grid grid-cols-2 gap-4 border-t pt-3 sm:grid-cols-3">
          {metrics.map((metric) => (
            <MetricBlock key={metric.key} metric={metric} />
          ))}
        </div>
      )}

      {body.text !== "" && (
        <p className="text-muted-foreground mt-3 text-xs leading-relaxed">
          {body.text}
          {body.generated && <GeneratedMark />}
        </p>
      )}

      {insight.recommendation !== "" && (
        <Alert className="mt-3">
          <LightbulbIcon className="size-4" />
          <AlertDescription>{insight.recommendation}</AlertDescription>
        </Alert>
      )}

      <div className="text-2xs text-muted-foreground mt-3 flex flex-wrap items-center gap-x-4 gap-y-1">
        {(insight.links ?? []).map((link) => (
          <Link
            key={link.path}
            to={link.path}
            className="hover:text-foreground inline-flex items-center gap-1 transition-colors"
          >
            {link.label}
            {link.count > 0 && <span className="tabular-nums">({link.count})</span>}
            <ArrowRightIcon className="size-2.5" />
          </Link>
        ))}
      </div>

      <Provenance insight={insight} />
    </div>
  );
}

function MetricBlock({ metric }: { metric: InsightMetric }) {
  const change = formatMetricChange(metric);
  const adverse = isChangeAdverse(metric);

  return (
    <div className="flex flex-col">
      <div className="flex items-baseline gap-1.5">
        <span className="font-table text-base tabular-nums">{formatMetricValue(metric)}</span>
        {change && (
          <span
            className={cn("text-2xs tabular-nums", adverse ? "text-destructive" : "text-success")}
          >
            {change}
          </span>
        )}
      </div>
      <span className="text-2xs text-muted-foreground">{metric.label}</span>
      {metric.baseline && metric.baselineLabel !== "" && (
        <span className="text-2xs text-muted-foreground/70">
          {formatMetricValue({ ...metric, value: metric.baseline })} · {metric.baselineLabel}
        </span>
      )}
    </div>
  );
}

/**
 * Where these numbers came from and when.
 *
 * A figure without its window invites someone to read a month's total as a
 * day's, and a card that names the model only when one was used is the honest
 * way to say which of these sentences a machine wrote.
 */
function Provenance({ insight }: { insight: Insight }) {
  const t = useT();

  return (
    <div className="border-border/60 text-2xs text-muted-foreground/70 mt-3 flex flex-wrap gap-x-4 gap-y-1 border-t pt-2">
      <span>
        {t("Measured")} {generateDateTimeStringFromUnixTimestamp(insight.windowStart)} –{" "}
        {generateDateTimeStringFromUnixTimestamp(insight.windowEnd)}
      </span>
      <span>
        {t("Found")} {generateDateTimeStringFromUnixTimestamp(insight.detectedAt)}
      </span>
      {insight.narrated && insight.modelIdentifier !== "" && (
        <span>
          {t("Explained by")} {insight.modelIdentifier}
        </span>
      )}
      {insight.dismissedAt && (
        <span>
          {t("Dismissed")} {generateDateTimeStringFromUnixTimestamp(insight.dismissedAt)}
          {insight.dismissReason !== "" && ` · ${insight.dismissReason}`}
        </span>
      )}
    </div>
  );
}

function GeneratedMark() {
  const t = useT();

  return (
    <span
      className="text-muted-foreground/60 ml-1 inline-flex translate-y-px align-middle"
      title={t(
        "This explanation was written by AI. The figures above were computed from your records.",
      )}
    >
      <SparklesIcon className="size-2.5" />
    </span>
  );
}

function statusLabel(insight: Insight, t: (key: string) => string): string {
  switch (insight.status) {
    case "Dismissed":
      return t("Dismissed");
    case "Resolved":
      return t("No longer found");
    default:
      return t("Replaced by newer numbers");
  }
}
