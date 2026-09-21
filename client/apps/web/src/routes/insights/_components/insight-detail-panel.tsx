import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { Insight, InsightMetric } from "@/types/insight";
import {
  formatMetricValue,
  insightBody,
  isChangeAdverse,
  formatMetricChange,
} from "@/routes/home/_components/widgets/insight-presentation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowRightIcon,
  LightbulbIcon,
  RotateCcwIcon,
  SearchCheckIcon,
  TrendingDownIcon,
  TrendingUpIcon,
  XIcon,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { CATEGORY_LABELS, SEVERITY_TONE } from "./insight-labels";
import {
  buildTrend,
  normalizePoints,
  primaryTrendMetricKey,
  type MetricTrend,
} from "./insight-trend";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";

/**
 * One finding, opened.
 *
 * The card in the list already shows the numbers. What this adds is the three
 * things a card cannot hold: how the finding has moved across refreshes, what
 * the rule actually looks for and what it declines to report, and somewhere to
 * say why you are dismissing it.
 */
export function InsightDetailPanel({
  insightId,
  onClose,
}: {
  insightId: string;
  onClose: () => void;
}) {
  const detailQuery = useQuery(queries.insight.detail(insightId));
  const detail = detailQuery.data;

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-xl">
        {detailQuery.isLoading || !detail ? (
          <div className="flex flex-col gap-4 p-6">
            <Skeleton className="h-8 w-3/4" />
            <Skeleton className="h-24" />
            <Skeleton className="h-32" />
          </div>
        ) : (
          <DetailBody
            insight={detail.insight}
            history={detail.history ?? []}
            explanation={detail.explanation}
            onClose={onClose}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

function DetailBody({
  insight,
  history,
  explanation,
  onClose,
}: {
  insight: Insight;
  history: Insight[];
  explanation: { measures: string; threshold: string; excludes: string };
  onClose: () => void;
}) {
  const t = useT();
  const body = insightBody(insight);
  const metricKey = primaryTrendMetricKey(insight);
  const trend = metricKey ? buildTrend(insight, history, metricKey) : null;

  return (
    <>
      <SheetHeader>
        <div className="mb-1.5 flex flex-wrap items-center gap-1.5">
          <Badge variant={SEVERITY_TONE[insight.severity]}>{insight.severity}</Badge>
          <Badge variant="neutral" appearance="outline">
            {t(CATEGORY_LABELS[insight.category])}
          </Badge>
          {insight.status !== "Active" && <Badge variant="neutral">{insight.status}</Badge>}
        </div>
        <SheetTitle className="leading-snug">{insight.headline}</SheetTitle>
        {insight.subject !== "" && <SheetDescription>{insight.subject}</SheetDescription>}
      </SheetHeader>

      <div className="flex flex-col gap-5 p-6 pt-0">
        <section>
          <SectionLabel>{t("What was measured")}</SectionLabel>
          <div className="grid grid-cols-2 gap-4">
            {(insight.metrics ?? []).map((metric) => (
              <MetricBlock key={metric.key} metric={metric} />
            ))}
          </div>
          <p className="text-2xs text-muted-foreground/70 mt-2">
            {generateDateTimeStringFromUnixTimestamp(insight.windowStart)} –{" "}
            {generateDateTimeStringFromUnixTimestamp(insight.windowEnd)}
          </p>
        </section>

        {trend && <TrendSection trend={trend} />}

        {body.text !== "" && (
          <section>
            <SectionLabel>
              {t("Explanation")}
              {body.generated && (
                <span className="text-muted-foreground/60 ml-1.5 inline-flex items-center gap-1 normal-case">
                  <AssistMark className="size-2.5" />
                  {t("written by AI")}
                </span>
              )}
            </SectionLabel>
            <p className="text-muted-foreground text-xs leading-relaxed">{body.text}</p>
          </section>
        )}

        {insight.recommendation !== "" && (
          <Alert>
            <LightbulbIcon className="size-4" />
            <AlertDescription>{insight.recommendation}</AlertDescription>
          </Alert>
        )}

        {(insight.links ?? []).length > 0 && (
          <section>
            <SectionLabel>{t("The records behind this")}</SectionLabel>
            <div className="flex flex-col gap-1.5">
              {(insight.links ?? []).map((link) => (
                <Link
                  key={link.path}
                  to={link.path}
                  className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 text-xs transition-colors"
                >
                  {link.label}
                  {link.count > 0 && <span className="tabular-nums">({link.count})</span>}
                  <ArrowRightIcon className="size-3" />
                </Link>
              ))}
            </div>
          </section>
        )}

        <RuleSection explanation={explanation} />

        <DecisionSection insight={insight} onDone={onClose} />
      </div>
    </>
  );
}

/**
 * How the finding has moved.
 *
 * Drawn as a bare sparkline with its endpoints labelled rather than a full
 * chart: the question is "which way and by how much", and axes would take four
 * times the room to answer it no better.
 */
function TrendSection({ trend }: { trend: MetricTrend }) {
  const t = useT();

  if (trend.points.length < 2) {
    return (
      <section>
        <SectionLabel>{t("Since last time")}</SectionLabel>
        <p className="text-2xs text-muted-foreground">
          {t("This is the first time this finding has been made, so there is nothing to compare.")}
        </p>
      </section>
    );
  }

  const normalized = normalizePoints(trend.points);
  const first = trend.points[0];
  const last = trend.points[trend.points.length - 1];

  return (
    <section>
      <SectionLabel>{t("Since last time")}</SectionLabel>
      <div className="flex items-center gap-3">
        <Sparkline values={normalized} worsening={trend.direction === "worsening"} />
        <div className="flex flex-col">
          <span
            className={cn(
              "inline-flex items-center gap-1 text-xs",
              trend.direction === "worsening" ? "text-destructive" : "text-success",
            )}
          >
            {trend.direction === "worsening" ? (
              <TrendingUpIcon className="size-3" />
            ) : (
              <TrendingDownIcon className="size-3" />
            )}
            {trend.direction === "worsening"
              ? t("Getting worse")
              : trend.direction === "improving"
                ? t("Improving")
                : t("Holding steady")}
          </span>
          <span className="text-2xs text-muted-foreground">
            {formatMetricValue({ ...trend.current, value: String(first?.value ?? 0) })} →{" "}
            {formatMetricValue({ ...trend.current, value: String(last?.value ?? 0) })}
          </span>
          <span className="text-2xs text-muted-foreground/70">
            {t("across {0} refreshes", String(trend.points.length))}
          </span>
        </div>
      </div>
    </section>
  );
}

function Sparkline({ values, worsening }: { values: number[]; worsening: boolean }) {
  const width = 96;
  const height = 28;
  const step = values.length > 1 ? width / (values.length - 1) : 0;

  const path = values
    .map((value, index) => {
      // The SVG origin is top-left, so a high value is a low y.
      const y = height - value * (height - 4) - 2;

      return `${index === 0 ? "M" : "L"}${index * step},${y}`;
    })
    .join(" ");

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className="shrink-0"
      role="presentation"
    >
      <path
        d={path}
        fill="none"
        strokeWidth={1.5}
        className={worsening ? "stroke-destructive" : "stroke-success"}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/**
 * What the rule looks for, and what it does not.
 *
 * The exclusions are the part that makes a finding auditable. A reader who knows
 * the on-time rule ignores customers under twelve deliveries can tell "no
 * finding" apart from "no problem", which are very different answers.
 */
function RuleSection({
  explanation,
}: {
  explanation: { measures: string; threshold: string; excludes: string };
}) {
  const t = useT();

  if (explanation.measures === "") {
    return null;
  }

  return (
    <section className="border-border/60 bg-muted/30 rounded-md border p-3">
      <SectionLabel>
        <SearchCheckIcon className="mr-1 inline size-3" />
        {t("Why you are seeing this")}
      </SectionLabel>
      <dl className="flex flex-col gap-2 text-xs">
        <div>
          <dt className="text-muted-foreground/70 text-2xs">{t("Measures")}</dt>
          <dd className="text-muted-foreground">{explanation.measures}</dd>
        </div>
        {explanation.threshold !== "" && (
          <div>
            <dt className="text-muted-foreground/70 text-2xs">{t("Reported when")}</dt>
            <dd className="text-muted-foreground">{explanation.threshold}</dd>
          </div>
        )}
        {explanation.excludes !== "" && (
          <div>
            <dt className="text-muted-foreground/70 text-2xs">{t("Does not report")}</dt>
            <dd className="text-muted-foreground">{explanation.excludes}</dd>
          </div>
        )}
      </dl>
    </section>
  );
}

/**
 * Dismissing, with somewhere to say why.
 *
 * The widget's dismiss sends no reason because there is nowhere on a tile to
 * type one. Here there is, and "why did we ignore this" is the question asked
 * three months later. It stays optional: making it mandatory would only teach
 * people to type a full stop.
 */
function DecisionSection({ insight, onDone }: { insight: Insight; onDone: () => void }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState("");

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: queries.insight._def });
    onDone();
  };

  const dismissMutation = useApiMutation({
    mutationFn: () => apiService.insightService.dismiss(insight.id, reason.trim()),
    onSuccess: invalidate,
    resourceName: "Insight",
  });

  const restoreMutation = useApiMutation({
    mutationFn: () => apiService.insightService.restore(insight.id),
    onSuccess: invalidate,
    resourceName: "Insight",
  });

  if (insight.status === "Dismissed") {
    return (
      <section className="border-border/60 border-t pt-4">
        <p className="text-2xs text-muted-foreground mb-2">
          {t("Dismissed")} {generateDateTimeStringFromUnixTimestamp(insight.dismissedAt ?? 0)}
          {insight.dismissReason !== "" && ` · ${insight.dismissReason}`}
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => restoreMutation.mutate(undefined)}
          disabled={restoreMutation.isPending}
        >
          <RotateCcwIcon className="size-3.5" />
          {t("Restore this finding")}
        </Button>
      </section>
    );
  }

  if (insight.status !== "Active") {
    return null;
  }

  return (
    <section className="border-border/60 flex flex-col gap-2 border-t pt-4">
      <SectionLabel>{t("Not worth acting on?")}</SectionLabel>
      <Textarea
        value={reason}
        onChange={(event) => setReason(event.target.value)}
        placeholder={t("Optional: why is this being ignored?")}
        rows={2}
        className="resize-none text-xs"
      />
      <div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => dismissMutation.mutate(undefined)}
          disabled={dismissMutation.isPending}
        >
          <XIcon className="size-3.5" />
          {t("Dismiss for 30 days")}
        </Button>
      </div>
      <p className="text-2xs text-muted-foreground/70">
        {t("It will come back if the condition is still there in a month.")}
      </p>
    </section>
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

function SectionLabel({ children }: { children: React.ReactNode }) {
  return <p className="cc-label text-muted-foreground mb-1.5">{children}</p>;
}
