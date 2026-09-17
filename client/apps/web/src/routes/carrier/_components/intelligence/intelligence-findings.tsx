import { useT } from "@trenova/shared/i18n/use-t";
import { FindingList } from "@/components/carrier-intelligence/finding-list";
import { GrantOverrideDialog } from "@/components/carrier-intelligence/grant-override-dialog";
import { RevokeOverrideDialog } from "@/components/carrier-intelligence/revoke-override-dialog";
import { canOverrideFinding } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTEL_OVERRIDES_KEY,
  fetchCarrierIntelOverrides,
  type CarrierIntelFinding,
  type CarrierIntelOverride,
} from "@/lib/graphql/carrier-intelligence";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { ShieldOffIcon, ShieldPlusIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";

export type IntelligenceFindingsProps = {
  carrierId: string;
  findings: readonly CarrierIntelFinding[];
  ruleLabels: Readonly<Record<string, string>>;
  canApprove: boolean;
  onChanged: () => void;
};

function OverrideHistory({
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

  if (overrides.length === 0) {
    return null;
  }

  return (
    <section aria-label={t("Overrides")} className="flex flex-col gap-2">
      <h4 className="text-sm font-medium">{t("Overrides")}</h4>
      <ul className="bg-card divide-y rounded-lg border">
        {overrides.map((override) => (
          <li
            key={override.id}
            className="flex flex-col gap-1 px-3 py-2 sm:flex-row sm:items-start sm:justify-between"
          >
            <div className="flex min-w-0 flex-col gap-0.5">
              <div className="flex flex-wrap items-center gap-1.5">
                <span className="text-sm font-medium">
                  {ruleLabels[override.ruleCode] ?? override.ruleCode}
                </span>
                {override.active ? (
                  <Badge variant="teal" className="max-h-5">
                    {t("Active")}
                  </Badge>
                ) : override.revokedAt ? (
                  <Badge variant="secondary" className="max-h-5">
                    {t("Revoked")}
                  </Badge>
                ) : (
                  <Badge variant="outline" className="max-h-5">
                    {t("Expired")}
                  </Badge>
                )}
              </div>
              <p className="text-sm">{override.reason}</p>
              <p className="text-muted-foreground text-xs">
                {t(
                  "Granted {0} · expires {1}",
                  formatUnixDateTimeMedium(override.grantedAt),
                  formatUnixDateTimeMedium(override.expiresAt),
                )}
                {override.revokedAt
                  ? ` · ${t("revoked {0}", formatUnixDateTimeMedium(override.revokedAt))}`
                  : null}
                {override.revokeReason ? ` · ${override.revokeReason}` : null}
              </p>
            </div>
            {canApprove && override.active ? (
              <Button type="button" size="xs" variant="outline" onClick={() => onRevoke(override)}>
                <ShieldOffIcon />
                {t("Revoke")}
              </Button>
            ) : null}
          </li>
        ))}
      </ul>
    </section>
  );
}

export function IntelligenceFindings({
  carrierId,
  findings,
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
          <Button type="button" size="xs" variant="outline" onClick={() => setGranting(finding)}>
            <ShieldPlusIcon />
            {t("Grant override")}
          </Button>
        );
      }
      const override = finding.overrideId ? overridesById.get(finding.overrideId) : undefined;
      if (finding.overridden && override?.active) {
        return (
          <Button type="button" size="xs" variant="outline" onClick={() => setRevoking(override)}>
            <ShieldOffIcon />
            {t("Revoke override")}
          </Button>
        );
      }
      return null;
    },
    [canApprove, overridesById, t],
  );

  return (
    <div className="flex flex-col gap-4">
      <FindingList findings={findings} ruleLabels={ruleLabels} renderActions={renderActions} />
      {overridesQuery.isPending ? (
        <Skeleton className="h-16 w-full" />
      ) : overridesQuery.isError ? (
        <p className="text-destructive text-sm">
          {t("Overrides could not be loaded. {0}", overridesQuery.error.message)}
        </p>
      ) : (
        <OverrideHistory
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
