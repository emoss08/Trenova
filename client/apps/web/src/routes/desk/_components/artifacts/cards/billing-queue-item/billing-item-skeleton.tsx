import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardDocumentsSkeleton,
  CardBoxSkeleton,
  CardChecksSkeleton,
  CardHeadSkeleton,
  CardLedgerSkeleton,
  CardSectionSkeleton,
  CardTimelineSkeleton,
  RecordCardSkeleton,
} from "../../skeletons/record-card-skeleton";

const CHECKS = [
  [34, 58],
  [28, 44],
  [40, 62],
  [30, 50],
  [36, 40],
] as const;
const CHARGES = [38, 46, 30] as const;
const ACTIVITY = [58, 46, 64] as const;

/**
 * The billing item while its code or its item is read: the item's numbers over
 * the bill-to and the amount, the five checks, the charges against the rate
 * con, the lane, what has happened to it, the shipment's documents, and the
 * bar of its actions, each where the card draws it.
 */
export function BillingItemSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the billing item…")}>
      <CardHeadSkeleton
        kicker={[96, 84]}
        stepper={stepper}
        title={48}
        subtitle={34}
        figure
        meta={[118, 84]}
      />
      <CardSectionSkeleton title={150} aside={72}>
        <CardChecksSkeleton rows={CHECKS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={60} aside={96}>
        <CardLedgerSkeleton lines={CHARGES} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64} aside={88}>
        <CardBoxSkeleton height={52} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={56}>
        <CardTimelineSkeleton entries={ACTIVITY} />
      </CardSectionSkeleton>
      <CardDocumentsSkeleton />
      <CardBarSkeleton actions={[64, 88]} />
    </RecordCardSkeleton>
  );
}
