import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardDocumentsSkeleton,
  CardHeadSkeleton,
  CardLedgerSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

/** Each stop's name and its window, as percentages of the rail's width. */
const STOP_ROWS = [
  [46, 58],
  [38, 64],
  [52, 54],
  [42, 60],
] as const;
/** Where the stops fall along the strip. */
const STRIP_DOTS = [0, 34, 68, 100] as const;
const COVERAGE_ROWS = [[36, 52]] as const;
const CHARGE_LINES = [34, 46, 28] as const;

/**
 * The shipment card while its code or its shipment is read: the PRO and lane
 * over the total and stage, the live strip with its stops along the line, the
 * stops down their rail, who covers the load, the charges as a ledger, and
 * the documents as tiles, each where the card draws it.
 */
export function DeskShipmentSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the shipment…")}>
      <CardHeadSkeleton
        kicker={[84, 64]}
        stepper={stepper}
        title={58}
        subtitle={42}
        figure
        meta={[112, 52, 64]}
      />
      <div className="dk-sh-nowsec">
        <div className="dk-sh-now dk-sh-now-sk">
          <div className="dk-sh-ends">
            <SkeletonBar w="96px" className="dk-rec-sk-line" />
            <SkeletonBar w="104px" className="dk-rec-sk-line" />
          </div>
          <div className="dk-sh-track">
            {STRIP_DOTS.map((left) => (
              <i key={left} className="dk-sk dk-sh-dot" style={{ left: `${left}%` }} />
            ))}
          </div>
          <div className="dk-sh-read">
            <SkeletonBar w="44%" className="dk-rec-sk-line" />
          </div>
        </div>
      </div>
      <CardSectionSkeleton title={44} aside={96}>
        <ol className="dk-sh-stops">
          {STOP_ROWS.map(([name, slot], index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <li key={index} className="dk-sh-stop">
              <i className="dk-sk dk-sh-node" />
              <span className="dk-sh-sb dk-sh-sk-sb">
                <SkeletonBar w={name} className="dk-rec-sk-line" />
                <SkeletonBar w={slot} className="dk-rec-sk-label" />
              </span>
            </li>
          ))}
        </ol>
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64}>
        <CardRowsSkeleton rows={COVERAGE_ROWS} trailing={72} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={56}>
        <CardLedgerSkeleton lines={CHARGE_LINES} />
      </CardSectionSkeleton>
      <CardDocumentsSkeleton />
      <CardBarSkeleton actions={[84, 128]} />
    </RecordCardSkeleton>
  );
}
