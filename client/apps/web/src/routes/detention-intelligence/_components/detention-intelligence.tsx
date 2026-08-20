import { CustomerMargin } from "./customer-margin";
import { DetentionLedger, DetentionLedgerSkeleton } from "./detention-ledger";
import { FacilityProfiles } from "./facility-profiles";
import { PanelError } from "./intelligence-panel";
import {
  useDetentionIntelligence,
  type DetentionWindowValue,
  detentionWindowDays,
} from "./use-detention-intelligence";
import { WaiverLeakage } from "./waiver-leakage";

/**
 * Detention intelligence: which facilities cost the most, which customers are
 * unprofitable once driver pay is netted off, and where discretionary revenue
 * is going.
 */
export function DetentionIntelligence({
  windowValue,
}: {
  windowValue: DetentionWindowValue;
}) {
  const { facilities, customers, waivers, rollup } = useDetentionIntelligence(
    detentionWindowDays(windowValue),
  );

  return (
    <div className="flex flex-col gap-4">
      {facilities.isError ? (
        <div className="border-border bg-card overflow-hidden rounded-lg border">
          <PanelError onRetry={() => void facilities.refetch()} />
        </div>
      ) : facilities.isLoading ? (
        <DetentionLedgerSkeleton />
      ) : (
        <DetentionLedger rollup={rollup} />
      )}

      <FacilityProfiles
        index={1}
        rows={facilities.data ?? []}
        isLoading={facilities.isLoading}
        isError={facilities.isError}
        onRetry={() => void facilities.refetch()}
      />

      <div className="grid min-w-0 gap-4 xl:grid-cols-2">
        <CustomerMargin
          index={2}
          rows={customers.data ?? []}
          isLoading={customers.isLoading}
          isError={customers.isError}
          onRetry={() => void customers.refetch()}
        />
        <WaiverLeakage
          index={3}
          rows={waivers.data ?? []}
          isLoading={waivers.isLoading}
          isError={waivers.isError}
          onRetry={() => void waivers.refetch()}
        />
      </div>
    </div>
  );
}
