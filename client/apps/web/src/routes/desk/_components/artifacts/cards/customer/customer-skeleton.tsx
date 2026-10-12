import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardMeterSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { CUSTOMER_SECTIONS } from "./customer-sections";

const AGING_CELLS = 5;
const STAT_CELLS = 4;
const TERM_CELLS = 4;
const SHIPMENT_ROWS = [
  [22, 56],
  [26, 48],
  [20, 60],
] as const;
const AGREEMENT_ROWS = [
  [38, 24],
  [30, 20],
] as const;

/**
 * The customer card while its code or its customer is read: the code and
 * place over the name, address and what is open, the aging strip with its
 * buckets and the payment figures, the credit meter over the billing terms,
 * and the shipments and agreements as rows, each where the card draws it. A
 * section the person may not read is left out here as it is on the card.
 */
export function DeskCustomerSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(CUSTOMER_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the customer…")}>
      <CardHeadSkeleton
        kicker={[44, 88]}
        stepper={stepper}
        title={48}
        subtitle={72}
        figure={access.receivables}
        meta={[92, 116]}
      />
      {access.receivables && (
        <CardSectionSkeleton title={84} aside={64}>
          <i className="dk-sk dk-cu-age dk-cu-age-sk" />
          <div className="dk-cu-buckets">
            {Array.from({ length: AGING_CELLS }, (_, index) => (
              // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
              <div key={index}>
                <SkeletonBar w="52%" className="dk-rec-sk-label" />
                <SkeletonBar w="72%" className="dk-rec-sk-line" />
              </div>
            ))}
          </div>
          <div className="dk-cu-stats">
            {Array.from({ length: STAT_CELLS }, (_, index) => (
              // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
              <div key={index}>
                <SkeletonBar w="46%" className="dk-rec-sk-label" />
                <SkeletonBar w="64%" className="dk-rec-sk-line" />
              </div>
            ))}
          </div>
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={52}>
        <CardMeterSkeleton />
        <div className="dk-cu-terms">
          {Array.from({ length: TERM_CELLS }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={index}>
              <SkeletonBar w="48%" className="dk-rec-sk-label" />
              <SkeletonBar w="70%" className="dk-rec-sk-line" />
            </div>
          ))}
        </div>
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardRowsSkeleton rows={SHIPMENT_ROWS} trailing={88} />
        </CardSectionSkeleton>
      )}
      {access.agreements && (
        <CardSectionSkeleton title={104}>
          <CardRowsSkeleton rows={AGREEMENT_ROWS} trailing={96} />
        </CardSectionSkeleton>
      )}
      <CardBarSkeleton actions={[128]} />
    </RecordCardSkeleton>
  );
}
