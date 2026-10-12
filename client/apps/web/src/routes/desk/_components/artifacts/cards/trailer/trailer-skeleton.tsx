import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardDocumentsSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  CardShipmentsSkeleton,
  CardWhereSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { TRAILER_SECTIONS } from "./trailer-sections";

const LEGEND_ROWS = 2;
const SHIPMENT_ROWS = 3;
const PLATE_CELLS = 8;

/** The service calendar's axis and legend while the trailer is read. */
function CalendarSkeleton() {
  return (
    <div className="dk-tl-cal">
      <i className="dk-sk dk-tl-axis-sk" />
      <ul className="dk-tl-legend">
        {Array.from({ length: LEGEND_ROWS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <li key={index}>
            <i className="dk-sk dk-rec-sk-mark" />
            <span>
              <SkeletonBar w="60%" className="dk-rec-sk-line" />
              <SkeletonBar w="44%" className="dk-rec-sk-label" />
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** The data plate while the trailer is read. */
function PlateSkeleton() {
  return (
    <div className="dk-tl-plate">
      {Array.from({ length: PLATE_CELLS }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index}>
          <SkeletonBar w="44%" className="dk-rec-sk-label" />
          <SkeletonBar w="64%" className="dk-rec-sk-line" />
        </div>
      ))}
    </div>
  );
}

/**
 * The trailer card while its code or its trailer is read: the code and type
 * over the name, VIN and maximum load, where its last move left it beside the
 * map's square, what falls due as checks, its shipments as rows and its
 * details as a grid, each where the card draws it. A section the person may
 * not read is left out here as it is on the card.
 */
export function DeskTrailerSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(TRAILER_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the trailer…")}>
      <CardHeadSkeleton
        kicker={[48, 40, 64]}
        stepper={stepper}
        title={48}
        subtitle={44}
        figure
        meta={[96, 84]}
      />
      <CardSectionSkeleton title={72}>
        <CardWhereSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={112} aside={72}>
        <CalendarSkeleton />
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardShipmentsSkeleton layout="rows" rows={SHIPMENT_ROWS} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={72}>
        <PlateSkeleton />
      </CardSectionSkeleton>
      <CardDocumentsSkeleton />
      <CardBarSkeleton actions={[108]} />
    </RecordCardSkeleton>
  );
}
