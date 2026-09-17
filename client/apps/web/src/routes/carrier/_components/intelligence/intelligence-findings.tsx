import { DecisionSummary } from "@/components/carrier-intelligence/decision-summary";
import { GrantOverrideDialog } from "@/components/carrier-intelligence/grant-override-dialog";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { RevokeOverrideDialog } from "@/components/carrier-intelligence/revoke-override-dialog";
import { StatusDot, type StatusTone } from "@/components/carrier-intelligence/status-dot";
import {
  canOverrideFinding,
  carrierIntelOverrideState,
  type CarrierIntelOverrideState,
} from "@/lib/carrier-intelligence";
import {
  CARRIER_INTEL_OVERRIDES_KEY,
  fetchCarrierIntelOverrides,
  type CarrierIntelFinding,
  type CarrierIntelOverride,
} from "@/lib/graphql/carrier-intelligence";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useCallback, useMemo, useState } from "react";

export type IntelligenceFindingsProps = {
  carrierId: string;
  findings: readonly CarrierIntelFinding[];
  notFound: boolean;
  provider: string | null;
  ruleLabels: Readonly<Record<string, string>>;
  canApprove: boolean;
  onChanged: () => void;
};

const OVERRIDE_TONE: Record<CarrierIntelOverrideState, StatusTone> = {
  active: "success",
  revoked: "neutral",
  expired: "neutral",
};

function OverrideRows({
  overrides,
  ruleLabels,
  canApprove,
  onRevoke,
}: {
  overrides: readonly CarrierIntelOverride[];
  ruleLabels: Readonly<Record<string, string>>;
  canApprove: boolean;
  onRevoke: (override: CarrierIntelOverride) => void;
}) {
  const t = useT();
  const stateLabels: Record<CarrierIntelOverrideState, string> = {
    active: t("Active"),
    revoked: t("Revoked"),
    expired: t("Expired"),
  };

  if (overrides.length === 0) {
    return null;
  }

  return (
    <section aria-label={t("Overrides")} className="flex flex-col">
      <h3 className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
        {t("Overrides")}
        <span className="tabular-nums">{overrides.length}</span>
      </h3>
      <ul className="divide-border divide-y">
        {overrides.map((override) => {
          const state = carrierIntelOverrideState(override);
          return (
            <li
              key={override.id}
              className="flex items-start gap-2.5 py-2"
              data-override-state={state}
            >
              <span className="flex h-5 shrink-0 items-center">
                <StatusDot tone={OVERRIDE_TONE[state]} />
              </span>
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="text-sm">
                  {ruleLabels[override.ruleCode] ?? override.ruleCode}
                  <span className="text-muted-foreground"> · {stateLabels[state]}</span>
                </span>
                <span className="text-muted-foreground text-xs">
                  {[
                    override.reason,
                    t(
                      "granted {0}, expires {1}",
                      formatUnixDateMedium(override.grantedAt),
                      formatUnixDateMedium(override.expiresAt),
                    ),
                    override.revokedAt
                      ? t("revoked {0}", formatUnixDateMedium(override.revokedAt))
                      : null,
                    override.revokeReason,
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
              </div>
              {canApprove && override.active ? (
                <Button type="button" size="xs" variant="ghost" onClick={() => onRevoke(override)}>
                  {t("Revoke")}
                </Button>
              ) : null}
            </li>
          );
        })}
      </ul>
    </section>
  );
}

export function IntelligenceFindings({
  carrierId,
  findings,
  notFound,
  provider,
  ruleLabels,
  canApprove,
  onChanged,
}: IntelligenceFindingsProps) {
  const t = useT();
  const [granting, setGranting] = useState<CarrierIntelFinding | null>(null);
  const [revoking, setRevoking] = useState<CarrierIntelOverride | null>(null);

  const overridesQuery = useQuery({
    queryKey: [CARRIER_INTEL_OVERRIDES_KEY, carrierId],
    queryFn: ({ signal }) => fetchCarrierIntelOverrides(carrierId, { signal }),
  });

  const overridesById = useMemo(
    () => new Map((overridesQuery.data ?? []).map((override) => [override.id, override])),
    [overridesQuery.data],
  );

  const { refetch: refetchOverrides } = overridesQuery;
  const handleChanged = useCallback(() => {
    void refetchOverrides();
    onChanged();
  }, [onChanged, refetchOverrides]);

  const renderActions = useCallback(
    (finding: CarrierIntelFinding) => {
      if (!canApprove) {
        return null;
      }
      if (canOverrideFinding(finding)) {
        return (
          <Button type="button" size="xs" variant="ghost" onClick={() => setGranting(finding)}>
            {t("Grant override")}
          </Button>
        );
      }
      const override = finding.overrideId ? overridesById.get(finding.overrideId) : undefined;
      if (finding.overridden && override?.active) {
        return (
          <Button type="button" size="xs" variant="ghost" onClick={() => setRevoking(override)}>
            {t("Revoke override")}
          </Button>
        );
      }
      return null;
    },
    [canApprove, overridesById, t],
  );

  return (
    <div className="flex flex-col gap-6">
      <DecisionSummary
        findings={findings}
        ruleLabels={ruleLabels}
        notFound={notFound}
        provider={provider}
        grouped
        renderActions={renderActions}
      />
      {overridesQuery.isPending ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-9 w-full" />
        </div>
      ) : overridesQuery.isError ? (
        <IntelInlineError
          error={overridesQuery.error}
          title={t("Overrides could not be loaded")}
          onRetry={() => void refetchOverrides()}
        />
      ) : (
        <OverrideRows
          overrides={overridesQuery.data}
          ruleLabels={ruleLabels}
          canApprove={canApprove}
          onRevoke={setRevoking}
        />
      )}
      <GrantOverrideDialog
        carrierId={carrierId}
        finding={granting}
        ruleLabel={granting ? ruleLabels[granting.code] : undefined}
        open={granting !== null}
        onOpenChange={(open) => {
          if (!open) {
            setGranting(null);
          }
        }}
        onGranted={handleChanged}
      />
      <RevokeOverrideDialog
        override={revoking}
        ruleLabel={revoking ? ruleLabels[revoking.ruleCode] : undefined}
        open={revoking !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRevoking(null);
          }
        }}
        onRevoked={handleChanged}
      />
    </div>
  );
}
