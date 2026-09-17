import { useT } from "@trenova/shared/i18n/use-t";
import { SeverityBadge } from "@/components/carrier-intelligence/severity-badge";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { CARRIER_INTEL_SEVERITY_RANK, carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import {
  resumeCarrierIntelMonitoring,
  type CarrierIntelFeedState,
  type CarrierIntelMonitoringStatus,
} from "@/lib/graphql/carrier-intel-settings";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { CirclePauseIcon, PlayIcon, PlugZapIcon } from "lucide-react";
import { useMemo, type ReactNode } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import type { MonitoringTab } from "./monitoring-tabs";

export type MonitoringSummaryProps = {
  status: CarrierIntelMonitoringStatus;
  canManage: boolean;
  onNavigate: (tab: MonitoringTab) => void;
};

type SummaryTileProps = {
  label: string;
  value: number;
  detail?: ReactNode;
  tone?: "critical" | "warning";
  onClick?: () => void;
  actionLabel?: string;
};

function SummaryTile({ label, value, detail, tone, onClick, actionLabel }: SummaryTileProps) {
  const content = (
    <>
      <span className="text-muted-foreground text-2xs font-medium uppercase">{label}</span>
      <span
        className={cn(
          "text-xl font-semibold tabular-nums",
          tone === "critical" && "text-destructive",
          tone === "warning" && "text-yellow-700 dark:text-yellow-400",
        )}
      >
        {value.toLocaleString()}
      </span>
      {detail ? <span className="text-muted-foreground text-xs">{detail}</span> : null}
    </>
  );

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        aria-label={actionLabel}
        className="bg-card hover:bg-accent focus-visible:ring-ring flex min-w-0 flex-col items-start gap-0.5 rounded-lg border p-3 text-left transition-colors outline-none focus-visible:ring-2"
      >
        {content}
      </button>
    );
  }

  return (
    <div className="bg-card flex min-w-0 flex-col items-start gap-0.5 rounded-lg border p-3">
      {content}
    </div>
  );
}

function FeedStateRow({ feed, now }: { feed: CarrierIntelFeedState; now: number }) {
  const t = useT();
  const labels = useCarrierIntelLabels();

  return (
    <li
      className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 px-3 py-2"
      data-feed-paused={feed.pausedReason ? "true" : "false"}
    >
      <div className="flex min-w-0 items-center gap-2">
        {feed.pausedReason ? (
          <Badge variant="warning" className="max-h-5">
            {t("Paused")}
          </Badge>
        ) : feed.lastError ? (
          <Badge variant="inactive" className="max-h-5">
            {t("Failing")}
          </Badge>
        ) : (
          <Badge variant="active" className="max-h-5">
            {t("Healthy")}
          </Badge>
        )}
        <span className="text-sm font-medium">{labels.feedType[feed.feedType]}</span>
        <span className="text-muted-foreground text-xs">
          {carrierIntelProviderLabel(feed.provider)}
        </span>
      </div>
      <div className="text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
        {feed.pausedReason ? (
          <span className="text-foreground">
            {feed.pausedAt
              ? t("{0} since {1}", feed.pausedReason, formatUnixDateTimeMedium(feed.pausedAt))
              : feed.pausedReason}
          </span>
        ) : feed.lastError ? (
          <span className="text-destructive max-w-96 truncate" title={feed.lastError}>
            {feed.lastError}
          </span>
        ) : null}
        <span title={feed.lastSuccessAt ? formatUnixDateTimeMedium(feed.lastSuccessAt) : undefined}>
          {feed.lastSuccessAt
            ? t("Last success {0}", formatRelativeTime(feed.lastSuccessAt - now))
            : t("No successful poll yet")}
        </span>
        {!feed.pausedReason && feed.nextPollAfter ? (
          <span title={formatUnixDateTimeMedium(feed.nextPollAfter)}>
            {t("Next poll {0}", formatRelativeTime(feed.nextPollAfter - now))}
          </span>
        ) : null}
        {feed.failureCount > 0 ? (
          <span className="tabular-nums">
            {t("{0, plural, one {# failure} other {# failures}}", feed.failureCount)}
          </span>
        ) : null}
      </div>
    </li>
  );
}

export function MonitoringSummary({ status, canManage, onNavigate }: MonitoringSummaryProps) {
  const t = useT();
  const now = useNowSeconds();
  const queryClient = useQueryClient();
  const { provider, enrollmentCounts, eventCounts, feeds, reviewQueueCount } = status;

  const severityCounts = useMemo(
    () =>
      eventCounts.bySeverity
        .filter((entry) => entry.count > 0)
        .sort(
          (a, b) =>
            CARRIER_INTEL_SEVERITY_RANK[a.severity] - CARRIER_INTEL_SEVERITY_RANK[b.severity],
        ),
    [eventCounts.bySeverity],
  );
  const pausedFeeds = feeds.filter((feed) => Boolean(feed.pausedReason));
  const criticalOpen =
    eventCounts.bySeverity.find((entry) => entry.severity === "Critical")?.count ?? 0;

  const resume = useApiMutation({
    mutationFn: resumeCarrierIntelMonitoring,
    resourceName: "Carrier monitoring",
    onSuccess: async () => {
      toast.success(t("Monitoring resumed"), {
        description: t("Paused feeds poll again on their next scheduled run."),
      });
      await queryClient.invalidateQueries({
        queryKey: queries.carrierIntelSettings.monitoringStatus().queryKey,
      });
    },
  });

  return (
    <section className="flex flex-col gap-3" aria-label={t("Monitoring status")}>
      {pausedFeeds.length > 0 ? (
        <Alert variant="warning">
          <CirclePauseIcon />
          <AlertTitle>{t("Carrier monitoring is paused")}</AlertTitle>
          <AlertDescription>
            {t(
              "{0, plural, one {# feed has} other {# feeds have}} stopped polling, so changes on monitored carriers are not being detected. Fix the cause in Integrations, then resume.",
              pausedFeeds.length,
            )}
          </AlertDescription>
          <AlertAction className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              nativeButton={false}
              render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
            >
              <PlugZapIcon className="size-3.5" />
              {t("Open integrations")}
            </Button>
            {canManage ? (
              <Button
                type="button"
                size="sm"
                isLoading={resume.isPending}
                loadingText={t("Resuming...")}
                onClick={() => resume.mutate()}
              >
                <PlayIcon className="size-3.5" />
                {t("Resume monitoring")}
              </Button>
            ) : null}
          </AlertAction>
        </Alert>
      ) : null}
      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
        <div className="bg-card col-span-2 flex min-w-0 flex-col gap-1 rounded-lg border p-3 md:col-span-1">
          <span className="text-muted-foreground text-2xs font-medium uppercase">
            {t("Provider")}
          </span>
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-base font-semibold">
              {provider.provider ? carrierIntelProviderLabel(provider.provider) : t("None")}
            </span>
            {provider.configured ? (
              <Badge variant="active" className="max-h-5">
                {t("Connected")}
              </Badge>
            ) : (
              <Badge variant="inactive" className="max-h-5">
                {t("Not connected")}
              </Badge>
            )}
          </div>
          <span className="text-muted-foreground text-xs">
            {provider.fallbackProvider
              ? t("Falls back to {0}", carrierIntelProviderLabel(provider.fallbackProvider))
              : t("No fallback provider")}
          </span>
        </div>
        <SummaryTile
          label={t("Open events")}
          value={eventCounts.open}
          tone={criticalOpen > 0 ? "critical" : eventCounts.open > 0 ? "warning" : undefined}
          onClick={() => onNavigate("inbox")}
          actionLabel={t("Open the event inbox")}
          detail={
            severityCounts.length > 0 ? (
              <span className="flex flex-wrap items-center gap-1">
                {severityCounts.map((entry) => (
                  <span key={entry.severity} className="inline-flex items-center gap-0.5">
                    <SeverityBadge severity={entry.severity} />
                    <span className="tabular-nums">{entry.count}</span>
                  </span>
                ))}
              </span>
            ) : (
              t("Nothing waiting")
            )
          }
        />
        <SummaryTile
          label={t("Acknowledged")}
          value={eventCounts.acknowledged}
          detail={t("Being worked, not yet resolved")}
          onClick={() => onNavigate("inbox")}
          actionLabel={t("Open the event inbox")}
        />
        <SummaryTile
          label={t("Review queue")}
          value={reviewQueueCount}
          tone={reviewQueueCount > 0 ? "warning" : undefined}
          detail={t("Carriers waiting for sign-off")}
          onClick={() => onNavigate("review")}
          actionLabel={t("Open the review queue")}
        />
        <SummaryTile
          label={t("Monitored carriers")}
          value={enrollmentCounts.active}
          tone={enrollmentCounts.failed > 0 ? "critical" : undefined}
          onClick={() => onNavigate("enrollments")}
          actionLabel={t("Open enrollments")}
          detail={t(
            "{0} wanted · {1} pending · {2} failed",
            enrollmentCounts.desired.toLocaleString(),
            enrollmentCounts.pending.toLocaleString(),
            enrollmentCounts.failed.toLocaleString(),
          )}
        />
      </div>
      {feeds.length > 0 ? (
        <ul className="bg-card divide-y rounded-lg border" aria-label={t("Monitoring feeds")}>
          {feeds.map((feed) => (
            <FeedStateRow key={`${feed.provider}-${feed.feedType}`} feed={feed} now={now} />
          ))}
        </ul>
      ) : (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-2 text-xs">
          {provider.configured
            ? t("No monitoring feed has run yet. Feeds start polling once carriers are enrolled.")
            : t("Feeds start polling once a provider is connected and carriers are enrolled.")}
        </p>
      )}
    </section>
  );
}
