import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardDocumentsSkeleton,
  CardFactGridSkeleton,
  CardHeadSkeleton,
  CardMeterSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
} from "../../skeletons/record-card-skeleton";
import { WORKER_SECTIONS } from "./worker-sections";

const STANDING_ROWS = [[40, 56]] as const;
const QUALIFICATION_ROWS = [
  [24, 40],
  [32, 44],
  [28, 36],
] as const;
const CREDENTIAL_ROWS = [[36, 40]] as const;
const SHIPMENT_ROWS = [
  [22, 56],
  [26, 48],
  [20, 60],
] as const;
const FACT_CELLS = 8;

/** The four clocks' outline, laid out as the card lays them. */
export function HosClocksSkeleton() {
  return (
    <div className="dk-wk-clocks">
      <CardMeterSkeleton />
      <CardMeterSkeleton />
      <CardMeterSkeleton />
      <CardMeterSkeleton />
    </div>
  );
}

/**
 * The worker card while its code or its worker is read: the kind of worker
 * over their name and license, the drive time left, their four clocks as
 * meters, their standing and qualifications as checks, their shipments as rows
 * and their details as a grid, each where the card draws it. A section the
 * person may not read is left out here as it is on the card.
 */
export function DeskWorkerSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(WORKER_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the worker…")}>
      <CardHeadSkeleton
        kicker={[52, 64, 48]}
        stepper={stepper}
        title={56}
        subtitle={60}
        figure
        meta={[112, 84]}
      />
      <CardSectionSkeleton title={104} aside={64}>
        <HosClocksSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={56} aside={84}>
        <CardChecksSkeleton rows={STANDING_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={96} aside={72}>
        <CardChecksSkeleton rows={QUALIFICATION_ROWS} />
      </CardSectionSkeleton>
      {access.credentials && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardChecksSkeleton rows={CREDENTIAL_ROWS} />
        </CardSectionSkeleton>
      )}
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
