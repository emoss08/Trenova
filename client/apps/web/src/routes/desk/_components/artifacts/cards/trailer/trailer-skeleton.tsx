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
import { TRAILER_SECTIONS } from "./trailer-sections";

const CHECK_ROWS = [
  [36, 48],
  [28, 40],
] as const;
const SHIPMENT_ROWS = [
  [22, 56],
  [26, 48],
  [20, 60],
] as const;
const FACT_CELLS = 8;

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
      <CardSectionSkeleton title={32} aside={72}>
        <CardChecksSkeleton rows={CHECK_ROWS} />
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardRowsSkeleton rows={SHIPMENT_ROWS} trailing={88} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={48}>
        <CardFactGridSkeleton cells={FACT_CELLS} />
      </CardSectionSkeleton>
      <CardDocumentsSkeleton />
      <CardBarSkeleton actions={[108]} />
    </RecordCardSkeleton>
  );
}
