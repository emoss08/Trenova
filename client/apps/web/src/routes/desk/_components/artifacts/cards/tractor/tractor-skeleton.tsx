import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardDocumentsSkeleton,
  CardFactGridSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardWhereSkeleton,
  RecordCardSkeleton,
} from "../../skeletons/record-card-skeleton";
import { TRACTOR_SECTIONS } from "./tractor-sections";

const CHECK_ROWS = [
  [34, 46],
  [28, 40],
] as const;
const DRIVER_ROWS = [[30, 22]] as const;
const SHIPMENT_ROWS = [
  [22, 56],
  [26, 48],
  [20, 60],
] as const;
const INSPECTION_ROWS = [
  [36, 54],
  [32, 48],
  [38, 52],
] as const;
const FACT_CELLS = 8;

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
      </CardSectionSkeleton>
      <CardSectionSkeleton title={32} aside={72}>
        <CardChecksSkeleton rows={CHECK_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={52}>
        <CardRowsSkeleton rows={DRIVER_ROWS} />
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardRowsSkeleton rows={SHIPMENT_ROWS} trailing={88} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={76}>
        <CardRowsSkeleton rows={INSPECTION_ROWS} trailing={72} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={48}>
        <CardFactGridSkeleton cells={FACT_CELLS} />
      </CardSectionSkeleton>
      <CardDocumentsSkeleton />
      <CardBarSkeleton actions={[116]} />
    </RecordCardSkeleton>
  );
}
