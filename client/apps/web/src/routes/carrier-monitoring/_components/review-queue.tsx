import { useT } from "@trenova/shared/i18n/use-t";
import { FindingList } from "@/components/carrier-intelligence/finding-list";
import { FreshnessIndicator } from "@/components/carrier-intelligence/freshness-indicator";
import { MarkReviewedDialog } from "@/components/carrier-intelligence/mark-reviewed-dialog";
import { RiskLevelBadge } from "@/components/carrier-intelligence/risk-level-badge";
import { ReviewStateBadge } from "@/components/carrier-intelligence/review-state-badge";
import { useCarrierIntelRuleLabels } from "@/components/carrier-intelligence/use-carrier-intel-rule-labels";
import { carrierPanelPath } from "@/lib/carrier-links";
import { carrierIntelProviderLabel, groupFindings } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTELLIGENCE_KEY,
  CARRIER_INTEL_REVIEW_QUEUE_KEY,
  fetchCarrierIntelReviewQueue,
  type CarrierIntelSnapshotSummary,
} from "@/lib/graphql/carrier-intelligence";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { ClipboardCheckIcon, ExternalLinkIcon, RefreshCwIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { Link } from "react-router";

export const REVIEW_QUEUE_LIMIT = 100;

export type ReviewQueueProps = {
  canUpdate: boolean;
};

function ReviewQueueSketch() {
  return (
    <div className="flex flex-col gap-2">
      {[70, 45, 55].map((share) => (
        <div key={share} className="flex flex-col gap-2 rounded-lg border p-3">
          <div className="flex items-center gap-2">
            <GhostLine className="w-32" />
            <GhostLine className="w-16" />
          </div>
          <GhostBar share={share} className="w-full" />
        </div>
      ))}
    </div>
  );
}

type ReviewQueueItemProps = {
  snapshot: CarrierIntelSnapshotSummary;
  ruleLabels: Readonly<Record<string, string>>;
  canUpdate: boolean;
  onMarkReviewed: (snapshot: CarrierIntelSnapshotSummary) => void;
};

function ReviewQueueItem({
  snapshot,
  ruleLabels,
  canUpdate,
  onMarkReviewed,
}: ReviewQueueItemProps) {
  const t = useT();
  const findings = useMemo(() => {
    const grouped = groupFindings(snapshot.findings);
    return [...grouped.blockers, ...grouped.advisories];
  }, [snapshot.findings]);
  const blockerCount = snapshot.blockingCodes.length;

  return (
    <li
      className="bg-card flex flex-col gap-3 rounded-lg border p-3"
      data-testid="review-queue-item"
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-1.5">
            {snapshot.carrierId ? (
              <Link
                to={carrierPanelPath(snapshot.carrierId, "intelligence")}
                className="truncate text-sm font-medium hover:underline"
              >
                {t("USDOT {0}", snapshot.dotNumber)}
              </Link>
            ) : (
              <span className="text-sm font-medium">{t("USDOT {0}", snapshot.dotNumber)}</span>
            )}
            <RiskLevelBadge level={snapshot.riskLevel} />
            <ReviewStateBadge state={snapshot.reviewState} reviewedAt={snapshot.reviewedAt} />
            {blockerCount > 0 ? (
              <Badge variant="inactive" className="max-h-5 tabular-nums">
                {t("{0, plural, one {# blocker} other {# blockers}}", blockerCount)}
              </Badge>
            ) : null}
          </div>
          <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
            <span>{carrierIntelProviderLabel(snapshot.provider)}</span>
            {snapshot.docketNumber ? <span>{t("MC {0}", snapshot.docketNumber)}</span> : null}
            <FreshnessIndicator
              fetchedAt={snapshot.fetchedAt}
              confirmedAt={snapshot.confirmedAt}
              effectiveAsOf={snapshot.effectiveAsOf}
            />
          </div>
        </div>
        <div className="flex items-center gap-1.5">
          {snapshot.carrierId ? (
            <Button
              size="sm"
              variant="ghost"
              nativeButton={false}
              render={<Link to={carrierPanelPath(snapshot.carrierId, "intelligence")} />}
            >
              <ExternalLinkIcon className="size-3.5" />
              {t("Open carrier")}
            </Button>
          ) : null}
          {canUpdate && snapshot.carrierId ? (
            <Button type="button" size="sm" onClick={() => onMarkReviewed(snapshot)}>
              <ClipboardCheckIcon className="size-3.5" />
              {t("Mark reviewed")}
            </Button>
          ) : null}
        </div>
      </div>
      <FindingList
        findings={findings}
        ruleLabels={ruleLabels}
        emptyMessage={t(
          "No blocking or advisory findings. The review was requested by a change on the carrier.",
        )}
      />
    </li>
  );
}

export function ReviewQueue({ canUpdate }: ReviewQueueProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const ruleLabels = useCarrierIntelRuleLabels(true);
  const [reviewing, setReviewing] = useState<CarrierIntelSnapshotSummary | null>(null);

  const queueQuery = useQuery({
    queryKey: [CARRIER_INTEL_REVIEW_QUEUE_KEY, REVIEW_QUEUE_LIMIT],
    queryFn: ({ signal }) => fetchCarrierIntelReviewQueue(REVIEW_QUEUE_LIMIT, { signal }),
  });

  const handleReviewed = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: [CARRIER_INTEL_REVIEW_QUEUE_KEY] });
    void queryClient.invalidateQueries({ queryKey: [CARRIER_INTELLIGENCE_KEY] });
    void queryClient.invalidateQueries({ queryKey: ["carrier-list"] });
    void queryClient.invalidateQueries({
      queryKey: queries.carrierIntelSettings.monitoringStatus().queryKey,
    });
  }, [queryClient]);

  if (queueQuery.isPending) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  if (queueQuery.isError) {
    return (
      <div className="text-destructive flex items-center justify-between gap-2 rounded-lg border border-dashed p-3 text-sm">
        <span>
          {t(
            "The review queue could not be loaded. {0}",
            graphQLErrorMessage(queueQuery.error, t("Try again in a moment.")),
          )}
        </span>
        <Button type="button" size="xs" variant="outline" onClick={() => void queueQuery.refetch()}>
          <RefreshCwIcon />
          {t("Retry")}
        </Button>
      </div>
    );
  }

  const snapshots = queueQuery.data;

  if (snapshots.length === 0) {
    return (
      <EmptySheet
        title={t("The review queue is clear")}
        description={t(
          "Carriers land here when a vetting or a monitored change needs someone to look at it and sign off.",
        )}
        sketch={<ReviewQueueSketch />}
      />
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-muted-foreground text-xs">
          {snapshots.length >= REVIEW_QUEUE_LIMIT
            ? t("Showing the first {0} carriers waiting for review.", REVIEW_QUEUE_LIMIT)
            : t(
                "{0, plural, one {# carrier is} other {# carriers are}} waiting for review.",
                snapshots.length,
              )}
        </p>
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          aria-label={t("Refresh review queue")}
          isLoading={queueQuery.isRefetching}
          onClick={() => void queueQuery.refetch()}
        >
          <RefreshCwIcon />
        </Button>
      </div>
      <ul className="flex flex-col gap-3">
        {snapshots.map((snapshot) => (
          <ReviewQueueItem
            key={snapshot.id}
            snapshot={snapshot}
            ruleLabels={ruleLabels}
            canUpdate={canUpdate}
            onMarkReviewed={setReviewing}
          />
        ))}
      </ul>
      {reviewing?.carrierId ? (
        <MarkReviewedDialog
          carrierId={reviewing.carrierId}
          blockingCount={reviewing.blockingCodes.length}
          open
          onOpenChange={(open) => {
            if (!open) {
              setReviewing(null);
            }
          }}
          onReviewed={handleReviewed}
        />
      ) : null}
    </div>
  );
}
