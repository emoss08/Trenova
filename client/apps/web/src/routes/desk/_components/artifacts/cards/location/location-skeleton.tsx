import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardFactGridSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardStatsSkeleton,
  CardWhereSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { LOCATION_SECTIONS } from "./location-sections";

const DWELL_CELLS = 4;
const POLICY_ROWS = [[34, 58]] as const;
const SHIPMENT_ROWS = [
  [22, 56],
  [26, 48],
  [20, 60],
] as const;
const FAILURE_ROWS = [
  [24, 62],
  [20, 54],
] as const;
const FACT_CELLS = 6;

/** The dwell figures' outline: the strip of four and the line of money under it. */
export function DwellSkeleton() {
  return (
    <>
      <CardStatsSkeleton cells={DWELL_CELLS} />
      <SkeletonBar w="58%" className="dk-rec-sk-line dk-lo-sk-money" />
    </>
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
        meta={[64, 132]}
      />
      <CardSectionSkeleton title={72}>
        <CardWhereSkeleton />
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
          <CardRowsSkeleton rows={SHIPMENT_ROWS} trailing={88} />
        </CardSectionSkeleton>
      )}
      {access.failures && (
        <CardSectionSkeleton title={104} aside={64}>
          <CardRowsSkeleton rows={FAILURE_ROWS} trailing={64} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={48}>
        <CardFactGridSkeleton cells={FACT_CELLS} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[116]} />
    </RecordCardSkeleton>
  );
}
