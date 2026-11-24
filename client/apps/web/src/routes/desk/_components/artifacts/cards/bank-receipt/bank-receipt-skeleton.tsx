import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardStatsSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { BANK_RECEIPT_SECTIONS } from "./bank-receipt-sections";

const CHECK_ROWS = [[44, 72]] as const;
const PAYMENT_ROWS = [
  [36, 48],
  [28, 20],
] as const;
const QUEUE_ROWS = [[24, 44]] as const;
const CANDIDATE_ROWS = [
  [40, 64],
  [32, 56],
] as const;

/** The bank's own line while the receipt is read: its day, its words and its amount. */
export function BankLineSkeleton() {
  return (
    <div className="grid grid-cols-[auto_1fr_auto] items-center gap-3 rounded-md border px-3 py-2">
      <SkeletonBar w={44} className="dk-rec-sk-label" />
      <SkeletonBar w="60%" className="dk-rec-sk-label" />
      <SkeletonBar w={64} className="dk-rec-sk-line" />
    </div>
  );
}

/**
 * The bank receipt card while its code or its receipt is read: its reference
 * and amount over where it stands, the bank's line, why it is or is not
 * applied, the payment it was matched to and its queue entry, each where the
 * card draws it. A section the person may not read is left out here as it is
 * on the card.
 */
export function DeskBankReceiptSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(BANK_RECEIPT_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the bank receipt…")}>
      <CardHeadSkeleton
        kicker={[72, 88]}
        stepper={stepper}
        title={48}
        subtitle={64}
        figure
        meta={[56, 96]}
      />
      <CardSectionSkeleton title={88}>
        <BankLineSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={96}>
        <CardChecksSkeleton rows={CHECK_ROWS} />
      </CardSectionSkeleton>
      {(access.payment || access.invoices) && (
        <CardSectionSkeleton title={104}>
          <CardRowsSkeleton rows={CANDIDATE_ROWS} trailing={72} />
        </CardSectionSkeleton>
      )}
      {access.payment && (
        <CardSectionSkeleton title={112}>
          <CardStatsSkeleton cells={3} />
          <CardRowsSkeleton rows={PAYMENT_ROWS} trailing={64} />
        </CardSectionSkeleton>
      )}
      {access.workItem && (
        <CardSectionSkeleton title={136}>
          <CardRowsSkeleton rows={QUEUE_ROWS} trailing={56} />
        </CardSectionSkeleton>
      )}
      <CardBarSkeleton actions={[140]} />
    </RecordCardSkeleton>
  );
}
