import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  CardStatsSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const PAIRS = 3;

/** The outline of a dispatch plan while its loads are read back. */
export function DispatchPlanSkeleton() {
  const t = useT();

  return (
    <RecordCardSkeleton label={t("Loading the dispatch plan…")} className="dk-dpl">
      <CardHeadSkeleton kicker={[72, 48, 64]} title={52} subtitle={68} figure meta={[120]} />
      <CardStatsSkeleton cells={4} className="dk-dpl-tally" />
      <CardSectionSkeleton title={40} aside={160}>
        <ol className="dk-dpl-ledger">
          {Array.from({ length: PAIRS }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <li key={index} className="dk-dpl-pair">
              <div className="dk-dpl-load">
                <SkeletonBar w={62} />
                <SkeletonBar w={78} />
                <SkeletonBar w={56} />
              </div>
              <div className="dk-dpl-link" />
              <div className="dk-dpl-drv">
                <SkeletonBar w={70} />
              </div>
            </li>
          ))}
        </ol>
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[120, 140]} />
    </RecordCardSkeleton>
  );
}
