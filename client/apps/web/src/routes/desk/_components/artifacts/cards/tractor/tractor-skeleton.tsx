import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardDocumentsSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardShipmentsSkeleton,
  CardSectionSkeleton,
  CardWhereSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { TRACTOR_SECTIONS } from "./tractor-sections";

const CHECK_ROWS = [
  [34, 46],
  [28, 40],
] as const;
const DIALS = 5;
const SEATS = 2;
const SHIPMENT_ROWS = 3;
const INSPECTION_ROWS = [
  [36, 54],
  [32, 48],
  [38, 52],
] as const;
const SPEC_ROWS = 8;

/** The dashboard strip under the map while the position is read. */
function DashSkeleton() {
  return (
    <div className="dk-tr-dash">
      {Array.from({ length: DIALS }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} className="dk-tr-dial">
          <SkeletonBar w="36px" className="dk-rec-sk-line" />
          <SkeletonBar w="52px" className="dk-rec-sk-label" />
        </div>
      ))}
    </div>
  );
}

/**
 * The tractor card while its code or its tractor is read: the code and type
 * over the name, VIN and odometer, where it is beside its dial, what falls due
 * as checks, its drivers, shipments and inspections as rows, and its details
 * as a grid, each where the card draws it. A section the person may not read
 * is left out here as it is on the card.
 */
export function DeskTractorSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(TRACTOR_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the tractor…")}>
      <CardHeadSkeleton
        kicker={[48, 40, 64]}
        stepper={stepper}
        title={52}
        subtitle={44}
        figure
        meta={[118, 84]}
      />
      <CardSectionSkeleton title={72} aside={96}>
        <CardWhereSkeleton />
        <DashSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={32} aside={72}>
        <CardChecksSkeleton rows={CHECK_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={52}>
        <div className="dk-tr-seats">
          {Array.from({ length: SEATS }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={index} className="dk-rec-row dk-tr-seat">
              <i className="dk-sk dk-rec-av" />
              <span className="dk-rec-main">
                <SkeletonBar w="60%" className="dk-rec-sk-line" />
                <SkeletonBar w="40%" className="dk-rec-sk-label" />
              </span>
            </div>
          ))}
        </div>
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardShipmentsSkeleton layout="board" rows={SHIPMENT_ROWS} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={76}>
        <CardRowsSkeleton rows={INSPECTION_ROWS} trailing={72} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={48}>
        <div className="dk-tr-spec">
          {Array.from({ length: SPEC_ROWS }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={index}>
              <SkeletonBar w="84px" className="dk-rec-sk-label" />
              <i aria-hidden />
              <SkeletonBar w="96px" className="dk-rec-sk-line" />
            </div>
          ))}
        </div>
      </CardSectionSkeleton>
      <CardDocumentsSkeleton />
      <CardBarSkeleton actions={[116]} />
    </RecordCardSkeleton>
  );
}
