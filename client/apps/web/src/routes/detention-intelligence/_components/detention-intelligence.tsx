import { useT } from "@trenova/shared/i18n/use-t";
import { CustomerMargin } from "./customer-margin";
import { DetentionIntelligenceEmpty } from "./detention-intelligence-empty";
import { DetentionLedger, DetentionLedgerSkeleton } from "./detention-ledger";
import { FacilityProfiles } from "./facility-profiles";
import { PanelError } from "./intelligence-panel";
import {
  DETENTION_WINDOW_OPTIONS,
  useDetentionIntelligence,
  type DetentionWindowValue,
  detentionWindowDays,
} from "./use-detention-intelligence";
import { WaiverLeakage } from "./waiver-leakage";

const WIDEST_WINDOW = DETENTION_WINDOW_OPTIONS[DETENTION_WINDOW_OPTIONS.length - 1];

/**
 * Detention intelligence: which facilities cost the most, which customers are
 * unprofitable once driver pay is netted off, and where discretionary revenue
 * is going.
 */
export function DetentionIntelligence({
  windowValue,
  onWiden,
}: {
  windowValue: DetentionWindowValue;
  /** Moves the page's window to its widest; absent once it is already there. */
  onWiden?: () => void;
}) {
  const t = useT();

  const days = detentionWindowDays(windowValue);
  const { facilities, customers, waivers, rollup } = useDetentionIntelligence(days);

  // Facilities cover every settled stop exactly once, so an empty facility
  // list means the whole window is empty, not just one panel of it.
  if (!facilities.isLoading && !facilities.isError && (facilities.data ?? []).length === 0) {
    return (
      <DetentionIntelligenceEmpty
        title={t("No detention in this window")}
        description={`No stop settled detention at any facility in the last ${days} days. Look back further, or check that the detention engine is switched on for this organization.`}
        onWiden={windowValue === WIDEST_WINDOW.value ? undefined : onWiden}
        widenLabel={`Look back ${WIDEST_WINDOW.days} days`}
      />
    );
  }

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
