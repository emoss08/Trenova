import { useT } from "@trenova/shared/i18n/use-t";
import { FreshnessIndicator } from "@/components/carrier-intelligence/freshness-indicator";
import { ReviewStateBadge } from "@/components/carrier-intelligence/review-state-badge";
import { RiskLevelBadge } from "@/components/carrier-intelligence/risk-level-badge";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import type { CarrierIntelControl } from "@/lib/graphql/carrier-intel-settings";
import {
  setCarrierMonitoring,
  type CarrierIntelSnapshot,
  type CarrierMonitoringEnrollment,
} from "@/lib/graphql/carrier-intelligence";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { BadgeCheckIcon, BellRingIcon, RadarIcon, ScanSearchIcon } from "lucide-react";
import { useId } from "react";
import { toast } from "sonner";

export type IntelligenceHeaderProps = {
  carrierId: string;
  snapshot: CarrierIntelSnapshot | null;
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
    <div className="bg-card flex flex-col gap-3 rounded-lg border p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1.5">
          <div className="flex flex-wrap items-center gap-1.5">
            <RadarIcon className="text-muted-foreground size-4" aria-hidden />
            <h3 className="text-sm font-semibold">{t("Carrier intelligence")}</h3>
            <span className="text-muted-foreground text-xs">
              {t("via {0}", carrierIntelProviderLabel(snapshot?.provider ?? provider))}
            </span>
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            <RiskLevelBadge level={snapshot?.riskLevel ?? null} />
            {snapshot ? (
              <ReviewStateBadge state={snapshot.reviewState} reviewedAt={snapshot.reviewedAt} />
            ) : null}
            {snapshot && snapshot.blockingCodes.length > 0 ? (
              <Badge variant="inactive" className="max-h-5 tabular-nums">
                {t(
                  "{0, plural, one {# blocker} other {# blockers}}",
                  snapshot.blockingCodes.length,
                )}
              </Badge>
            ) : null}
            <Badge
              variant={openEventCount > 0 ? "warning" : "outline"}
              className="max-h-5 tabular-nums"
            >
              <BellRingIcon aria-hidden />
              {t("{0, plural, one {# open event} other {# open events}}", openEventCount)}
            </Badge>
          </div>
          {snapshot ? (
            <FreshnessIndicator
              fetchedAt={snapshot.fetchedAt}
              confirmedAt={snapshot.confirmedAt}
              effectiveAsOf={snapshot.effectiveAsOf}
              sourceAsOf={snapshot.sourceAsOf}
              staleAfterHours={control?.preTenderMaxAgeHours}
              expiredAfterHours={control?.hardMaxAgeHours}
            />
          ) : null}
          {snapshot?.reviewNote ? (
            <p className="text-muted-foreground text-xs">
              {t("Review note: {0}", snapshot.reviewNote)}
            </p>
          ) : null}
        </div>
        {canUpdate ? (
          <div className="flex flex-wrap items-center gap-1.5">
            {snapshot && snapshot.reviewState === "NeedsReview" ? (
              <Button type="button" size="sm" variant="outline" onClick={onMarkReviewed}>
                <BadgeCheckIcon />
                {t("Mark reviewed")}
              </Button>
            ) : null}
            <Button type="button" size="sm" onClick={onVet}>
              <ScanSearchIcon />
              {snapshot ? t("Vet now") : t("Vet carrier")}
            </Button>
          </div>
        ) : null}
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <Label htmlFor={monitoringId}>{t("Continuous monitoring")}</Label>
          <span className="text-muted-foreground text-xs">
            {enrollment
              ? t(
                  "{0} · {1}",
                  enrolled ? t("Enrolled") : t("Not enrolled"),
                  labels.vendorState[enrollment.vendorState],
                )
              : t("Not enrolled")}
            {enrollment?.lastSyncedAt
              ? ` · ${t("last synced {0}", formatUnixDateTimeMedium(enrollment.lastSyncedAt))}`
              : null}
          </span>
          {enrollment?.vendorState === "Failed" && enrollment.lastError ? (
            <span className="text-destructive text-xs">
              {t(
                "Enrollment failed after {0, plural, one {# attempt} other {# attempts}}: {1}",
                enrollment.failureCount,
                enrollment.lastError,
              )}
            </span>
          ) : null}
        </div>
        <Switch
          id={monitoringId}
          checked={enrolled}
          disabled={!canUpdate || monitoring.isPending}
          onCheckedChange={(checked) => monitoring.mutate(checked)}
        />
      </div>
    </div>
  );
}
