import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardFactGridSkeleton,
  CardHeadSkeleton,
  CardMeterSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardStatsSkeleton,
  RecordCardSkeleton,
} from "../../skeletons/record-card-skeleton";
import { CARRIER_SECTIONS } from "./carrier-sections";

const ELIGIBILITY_ROWS = [[44, 30]] as const;
const INSURANCE_ROWS = [
  [32, 58],
  [28, 54],
] as const;
const AUTHORITY_CELLS = 4;
const BASIC_METERS = 4;
const SHIPMENT_ROWS = [
  [22, 56],
  [26, 48],
  [20, 60],
] as const;
const SETTLEMENT_ROWS = [
  [24, 58],
  [22, 52],
] as const;
const CONTACT_ROWS = [[30, 62]] as const;
const FACT_CELLS = 8;

/** The intelligence section's outline: the authority strip over the BASICs' meters. */
export function CarrierIntelSkeleton() {
  return (
    <>
      <CardStatsSkeleton cells={AUTHORITY_CELLS} />
      <div className="dk-ca-basics">
        {Array.from({ length: BASIC_METERS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <CardMeterSkeleton key={index} />
        ))}
      </div>
    </>
  );
}

/**
 * The carrier card while its code or its carrier is read: its identifiers over
 * the name and DBA, the risk figure, whether it can be tendered as checks, its
 * insurance as checks, its authority and BASICs, its shipments, settlements and
 * contacts as rows and its details as a grid, each where the card draws it. A
 * section the person may not read is left out here as it is on the card.
 */
export function DeskCarrierSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(CARRIER_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the carrier…")}>
      <CardHeadSkeleton
        kicker={[48, 72, 64, 56]}
        stepper={stepper}
        title={56}
        subtitle={40}
        figure
        meta={[88, 120]}
      />
      {access.eligibility && (
        <CardSectionSkeleton title={120} aside={84}>
          <CardChecksSkeleton rows={ELIGIBILITY_ROWS} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={64} aside={72}>
        <CardChecksSkeleton rows={INSURANCE_ROWS} />
      </CardSectionSkeleton>
      {access.intelligence && (
        <CardSectionSkeleton title={112} aside={96}>
          <CarrierIntelSkeleton />
        </CardSectionSkeleton>
      )}
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardRowsSkeleton rows={SHIPMENT_ROWS} trailing={88} />
        </CardSectionSkeleton>
      )}
      {access.settlements && (
        <CardSectionSkeleton title={84} aside={72}>
          <CardRowsSkeleton rows={SETTLEMENT_ROWS} trailing={80} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={64}>
        <CardRowsSkeleton rows={CONTACT_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={48}>
        <CardFactGridSkeleton cells={FACT_CELLS} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[108]} />
    </RecordCardSkeleton>
  );
}
