import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardLedgerSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const FORMULA_LINES = [88, 64, 76] as const;
const VARIABLE_ROWS = [
  [26, 58],
  [34, 46],
  [22, 64],
] as const;
const PRICE_LINES = [42, 30] as const;

/**
 * The formula template card while its code or its template is read: the
 * kicker and name over the sample price and status, the formula in its code
 * block with the limits under it, the variables with their fields, and the
 * sample price as a ledger, each where the card draws it.
 */
export function DeskFormulaTemplateSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the formula template…")}>
      <CardHeadSkeleton
        kicker={[92, 22]}
        stepper={stepper}
        title={52}
        subtitle={68}
        figure
        meta={[104, 88]}
      />
      <CardSectionSkeleton title={56} aside={44}>
        <div className="dk-ft-code dk-ft-code-sk">
          {FORMULA_LINES.map((line, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <SkeletonBar key={index} w={line} className="dk-rec-sk-line" />
          ))}
        </div>
        <div className="dk-ft-rules">
          {[0, 1, 2].map((rule) => (
            <div key={rule}>
              <SkeletonBar w="46%" className="dk-rec-sk-label" />
              <SkeletonBar w="64%" className="dk-rec-sk-line" />
            </div>
          ))}
        </div>
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64} aside={36}>
        <CardRowsSkeleton rows={VARIABLE_ROWS} trailing={112} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={84} aside={140}>
        <CardLedgerSkeleton lines={PRICE_LINES} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[150]} />
    </RecordCardSkeleton>
  );
}
