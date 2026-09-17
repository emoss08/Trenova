import { IntelSnapshotHeader } from "@/components/carrier-intelligence/intel-snapshot-header";
import { RelativeTime } from "@/components/carrier-intelligence/relative-time";
import { StatusDot } from "@/components/carrier-intelligence/status-dot";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import type { CarrierIntelControl } from "@/lib/graphql/carrier-intel-settings";
import {
  setCarrierMonitoring,
  type CarrierIntelSnapshot,
  type CarrierMonitoringEnrollment,
} from "@/lib/graphql/carrier-intelligence";
import { Button } from "@trenova/shared/components/ui/button";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId } from "react";
import { toast } from "sonner";

export type IntelligenceHeaderProps = {
  carrierId: string;
  snapshot: CarrierIntelSnapshot;
  enrollment: CarrierMonitoringEnrollment | null;
  openEventCount: number;
  provider: string | null;
  control: CarrierIntelControl | null;
  canUpdate: boolean;
  onVet: () => void;
  onMarkReviewed: () => void;
  onChanged: () => void;
};

export function IntelligenceHeader({
  carrierId,
  snapshot,
  enrollment,
  openEventCount,
  provider,
  control,
  canUpdate,
  onVet,
  onMarkReviewed,
  onChanged,
}: IntelligenceHeaderProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const monitoringId = useId();
  const enrolled = enrollment?.desiredState === "Enrolled";
  const failed = enrollment?.vendorState === "Failed";

  const monitoring = useApiMutation<number, boolean>({
    resourceName: "Carrier monitoring",
    mutationFn: (enabled) => setCarrierMonitoring([carrierId], enabled),
    onSuccess: (_count, enabled) => {
      toast.success(enabled ? t("Monitoring enabled") : t("Monitoring disabled"), {
        description: enabled
          ? t(
              "Changes at {0} will be picked up automatically.",
              carrierIntelProviderLabel(provider),
            )
          : t("This carrier is no longer watched for changes."),
      });
      onChanged();
    },
  });

  return (
    <IntelSnapshotHeader
      label={t("Carrier intelligence summary")}
      riskLevel={snapshot.riskLevel}
      reviewState={snapshot.reviewState}
      reviewedAt={snapshot.reviewedAt}
      blockingCount={snapshot.blockingCodes.length}
      openEventCount={openEventCount}
      freshness={{
        effectiveAsOf: snapshot.effectiveAsOf,
        fetchedAt: snapshot.fetchedAt,
        confirmedAt: snapshot.confirmedAt,
        depth: snapshot.depth,
        depthFetchedAt: snapshot.depthFetchedAt,
        fetchedDepth: snapshot.fetchedDepth,
        sourceAsOf: snapshot.sourceAsOf,
        staleAfterHours: control?.preTenderMaxAgeHours,
        expiredAfterHours: control?.hardMaxAgeHours,
      }}
      meta={t(
        "via {0} · {1}",
        carrierIntelProviderLabel(snapshot.provider ?? provider),
        labels.depth[snapshot.depth],
      )}
      note={snapshot.reviewNote ? t("Review note: {0}", snapshot.reviewNote) : null}
      actions={
        canUpdate ? (
          <>
            {snapshot.reviewState === "NeedsReview" ? (
              <Button type="button" size="sm" variant="ghost" onClick={onMarkReviewed}>
                {t("Mark reviewed")}
              </Button>
            ) : null}
            <Button type="button" size="sm" variant="outline" onClick={onVet}>
              {t("Vet now")}
            </Button>
          </>
        ) : null
      }
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <Switch
          id={monitoringId}
          size="sm"
          checked={enrolled}
          disabled={!canUpdate || monitoring.isPending}
          onCheckedChange={(checked) => monitoring.mutate(checked)}
        />
        <Label htmlFor={monitoringId} className="text-xs font-normal">
          {t("Continuous monitoring")}
        </Label>
        <span className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-xs">
          {enrollment ? (
            <>
              <span aria-hidden>·</span>
              <span className="inline-flex items-center gap-1.5">
                {failed ? <StatusDot tone="critical" /> : null}
                {labels.vendorState[enrollment.vendorState]}
              </span>
              {enrollment.lastSyncedAt ? (
                <>
                  <span aria-hidden>·</span>
                  <span className="inline-flex items-center gap-1">
                    {t("synced")} <RelativeTime timestamp={enrollment.lastSyncedAt} />
                  </span>
                </>
              ) : null}
            </>
          ) : null}
        </span>
        {failed && enrollment?.lastError ? (
          <p className="text-muted-foreground w-full text-xs">
            {t(
              "Enrollment failed after {0, plural, one {# attempt} other {# attempts}}: {1}",
              enrollment.failureCount,
              enrollment.lastError,
            )}
          </p>
        ) : null}
      </div>
    </IntelSnapshotHeader>
  );
}
