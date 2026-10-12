import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardShipmentsSkeleton,
  CardWhereSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { LOCATION_SECTIONS } from "./location-sections";

const POLICY_ROWS = [[34, 58]] as const;
const SHIPMENT_ROWS = 3;
const AMENITIES = [112, 96, 104, 72] as const;
const FILE_ROWS = 5;
const FAILURE_ROWS = [
  [24, 62],
  [20, 54],
] as const;

/** The dwell ruler's outline: the bar with its pins, the stops past free time, and the money line. */
export function DwellSkeleton() {
  return (
    <div className="dk-lo-dwell">
      <i className="dk-sk dk-lo-ruler-sk" />
      <i className="dk-sk dk-lo-breach-sk" />
      <SkeletonBar w="58%" className="dk-rec-sk-line dk-lo-sk-money" />
    </div>
  );
}

/** The amenity chips under the map while the location is read. */
function AmenitiesSkeleton() {
  return (
    <div className="dk-lo-amen">
      {AMENITIES.map((width, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <SkeletonBar key={index} w={`${width}px`} className="dk-rec-sk-pill" />
      ))}
    </div>
  );
}

/** The address card while the location is read. */
function FileSkeleton() {
  return (
    <div className="dk-lo-file">
      <div className="dk-lo-file-sk">
        <SkeletonBar w="70%" className="dk-rec-sk-line" />
        <SkeletonBar w="52%" className="dk-rec-sk-label" />
      </div>
      <div className="dk-lo-file-sk">
        {Array.from({ length: FILE_ROWS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <SkeletonBar key={index} w="88%" className="dk-rec-sk-label" />
        ))}
      </div>
    </div>
  );
}

/**
 * The location card while its code or its location is read: the code and
 * category over the name and address, the average dwell, the map's square
 * beside the address and fence, the dwell figures, the policies, shipments and
 * service failures as rows and the details as a grid, each where the card
 * draws it. A section the person may not read is left out here as it is on the
 * card.
 */
export function DeskLocationSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(LOCATION_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the location…")}>
      <CardHeadSkeleton
        kicker={[52, 72, 64]}
        stepper={stepper}
        title={60}
        subtitle={72}
        figure={access.detention}
        meta={[96]}
      />
      <CardSectionSkeleton title={72}>
        <CardWhereSkeleton />
        <AmenitiesSkeleton />
      </CardSectionSkeleton>
      {access.detention && (
        <CardSectionSkeleton title={120}>
          <DwellSkeleton />
        </CardSectionSkeleton>
      )}
      {access.detention && (
        <CardSectionSkeleton title={112}>
          <CardRowsSkeleton rows={POLICY_ROWS} trailing={72} />
        </CardSectionSkeleton>
      )}
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardShipmentsSkeleton layout="table" rows={SHIPMENT_ROWS} />
        </CardSectionSkeleton>
      )}
      {access.failures && (
        <CardSectionSkeleton title={104} aside={64}>
          <CardRowsSkeleton rows={FAILURE_ROWS} trailing={64} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={48}>
        <FileSkeleton />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[116]} />
    </RecordCardSkeleton>
  );
}
