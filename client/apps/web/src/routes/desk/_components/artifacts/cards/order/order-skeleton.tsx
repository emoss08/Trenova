import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardFactGridSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardStatsSkeleton,
  CardTimelineSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { ORDER_LIFECYCLE } from "./order-card";
import { ORDER_SECTIONS } from "./order-sections";

const SHIPMENT_ROWS = [
  [22, 56],
  [26, 48],
] as const;
const CHARGE_ROWS = [
  [30, 40],
  [26, 36],
] as const;
const ACTIVITY_ENTRIES = [58, 46, 52] as const;
const FACT_CELLS = 6;

/** The order's way, step by step, while it is read. */
export function LifecycleSkeleton() {
  return (
    <ol className="dk-or-way" aria-hidden>
      {ORDER_LIFECYCLE.map((step) => (
        <li key={step}>
          <i className="dk-sk" />
          <SkeletonBar w="64%" className="dk-rec-sk-label" />
        </li>
      ))}
    </ol>
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
          <CardRowsSkeleton rows={SHIPMENT_ROWS} trailing={88} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={60}>
        <CardStatsSkeleton cells={3} className="dk-or-sums" />
        <CardRowsSkeleton rows={CHARGE_ROWS} trailing={72} />
      </CardSectionSkeleton>
      {access.activity && (
        <CardSectionSkeleton title={56} aside={64}>
          <CardTimelineSkeleton entries={ACTIVITY_ENTRIES} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={48}>
        <CardFactGridSkeleton cells={FACT_CELLS} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[100]} />
    </RecordCardSkeleton>
  );
}
