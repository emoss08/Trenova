import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  CardShipmentsSkeleton,
  CardTimelineSkeleton,
  CardTrackSkeleton,
  RecordCardSkeleton,
  SettlementStatementSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { CARRIER_SETTLEMENT_SECTIONS } from "./carrier-settlement-sections";

/** The loads the card reads at a time; the next page is read on request. */
export const CARRIER_SETTLEMENT_SHIPMENT_ROWS = 3;
/** The activity entries the card reads at a time. */
export const CARRIER_SETTLEMENT_ACTIVITY_ROWS = 5;

const CELLS = 6;
const KEYS = 3;
const STATEMENT = [36, 42, 30] as const;
const ACTIVITY_ENTRIES = [64, 52, 58] as const;

/** The carrier's invoices while the settlement is read: the strip of moves and its keys. */
function ReconciliationSkeleton() {
  return (
    <div className="dk-cset-recon" aria-hidden>
      <div className="dk-cset-strip">
        {Array.from({ length: CELLS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <i key={index} className="dk-sk" />
        ))}
      </div>
      <dl className="dk-cset-keys">
        {Array.from({ length: KEYS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <div key={index}>
            <SkeletonBar w="60%" className="dk-rec-sk-label" />
            <SkeletonBar w="40%" className="dk-rec-sk-line" />
          </div>
        ))}
      </dl>
    </div>
  );
}

/**
 * The carrier settlement card while its code or its settlement is read: the
 * stub and the carrier over the loads it pays, its track, the carrier's
 * invoices, the costs, where the money goes, the loads and the activity, each
 * where the card draws it. A section the person may not read is left out here
 * as it is on the card.
 */
export function DeskCarrierSettlementSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(CARRIER_SETTLEMENT_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the settlement…")}>
      <CardHeadSkeleton
        kicker={[84, 88, 44]}
        stepper={stepper}
        lead={<i className="dk-sk dk-rec-stub" />}
        title={52}
        subtitle={48}
        figure
        meta={[80, 120]}
      />
      <CardSectionSkeleton title={96} aside={access.carrier ? 72 : undefined}>
        <CardTrackSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={104} aside={96}>
        <ReconciliationSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={40} aside={48}>
        <SettlementStatementSkeleton rows={STATEMENT} />
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={96} aside={56}>
          <CardShipmentsSkeleton layout="route" rows={CARRIER_SETTLEMENT_SHIPMENT_ROWS} />
        </CardSectionSkeleton>
      )}
      {access.activity && (
        <CardSectionSkeleton title={56} aside={64}>
          <CardTimelineSkeleton entries={ACTIVITY_ENTRIES} />
        </CardSectionSkeleton>
      )}
      <CardBarSkeleton actions={[96, 132]} />
    </RecordCardSkeleton>
  );
}
