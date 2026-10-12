import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  CardShipmentsSkeleton,
  CardTimelineSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { ORDER_SECTIONS } from "./order-sections";

const SHIPMENT_ROWS = 2;
const CHARGE_ROWS = 2;
const ACTIVITY_ENTRIES = [58, 46, 52] as const;
const REFERENCES = [88, 72, 72, 56, 72, 80] as const;

/** The order's track while it is read: one bar where the segments will stand, and its line. */
export function LifecycleSkeleton() {
  return (
    <div className="dk-or-track" aria-hidden>
      <i className="dk-sk dk-or-track-sk" />
      <SkeletonBar w="34%" className="dk-rec-sk-label" />
    </div>
  );
}

/** The charge ledger while it is read: its bar, its lines on their leaders, and its total. */
function LedgerSkeleton() {
  return (
    <div className="dk-or-ledger">
      <div className="dk-or-billed">
        <i className="dk-sk dk-or-billbar" />
        <SkeletonBar w="52%" className="dk-rec-sk-label" />
      </div>
      <ul>
        {Array.from({ length: CHARGE_ROWS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <li key={index}>
            <SkeletonBar w="100%" className="dk-rec-sk-line" />
            <SkeletonBar w="34%" className="dk-rec-sk-label" />
          </li>
        ))}
      </ul>
      <div className="dk-or-total">
        <SkeletonBar w="56px" className="dk-rec-sk-label" />
        <SkeletonBar w="72px" className="dk-rec-sk-line" />
      </div>
    </div>
  );
}

/**
 * The order card while its code or its order is read: its number over the
 * customer, the total, its way from draft to closed, its legs and charges as
 * rows, its activity down a line and its details as a grid, each where the
 * card draws it. A section the person may not read is left out here as it is
 * on the card.
 */
export function DeskOrderSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(ORDER_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the order…")}>
      <CardHeadSkeleton
        kicker={[56, 64, 56]}
        stepper={stepper}
        title={40}
        subtitle={52}
        figure
        meta={[72, 96]}
      />
      <CardSectionSkeleton title={96}>
        <LifecycleSkeleton />
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={64} aside={72}>
          <CardShipmentsSkeleton layout="route" rows={SHIPMENT_ROWS} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={60}>
        <LedgerSkeleton />
      </CardSectionSkeleton>
      {access.activity && (
        <CardSectionSkeleton title={56} aside={64}>
          <CardTimelineSkeleton entries={ACTIVITY_ENTRIES} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={48}>
        <div className="dk-or-refs">
          {REFERENCES.map((width, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={index}>
              <SkeletonBar w="40px" className="dk-rec-sk-label" />
              <SkeletonBar w={`${width}px`} className="dk-rec-sk-line" />
            </div>
          ))}
        </div>
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[100]} />
    </RecordCardSkeleton>
  );
}
