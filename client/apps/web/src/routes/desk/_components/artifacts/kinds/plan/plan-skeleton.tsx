import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardTimelineSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const STEP_TITLES = [62, 54, 70] as const;
const RECORD_ROWS = [
  [36, 52],
  [30, 40],
] as const;

/** The track under the plan's name while it is read: as many segments as steps drawn below. */
function TrackSkeleton({ segments }: { segments: number }) {
  return (
    <div className="dk-pln-track">
      {Array.from({ length: segments }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <span key={index} className="dk-sk dk-pln-sk-seg" />
      ))}
    </div>
  );
}

/** The outline of a plan while its steps, its preview and its decision are read. */
export function PlanSkeleton() {
  const t = useT();

  return (
    <RecordCardSkeleton label={t("Loading the plan…")} className="dk-pln">
      <CardHeadSkeleton
        kicker={[32, 52, 84]}
        title={58}
        subtitle={72}
        figure
        under={<TrackSkeleton segments={STEP_TITLES.length} />}
        meta={[132, 96]}
      />
      <CardSectionSkeleton title={44} aside={180}>
        <div className="dk-pln-steps">
          {STEP_TITLES.map((title, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={index} className="dk-pln-sk-step">
              <i className="dk-sk dk-pln-sk-node" />
              <div>
                <SkeletonBar w={`${title}%`} className="dk-rec-sk-line" />
                <SkeletonBar w="34%" className="dk-rec-sk-label" />
                <i className="dk-sk dk-pln-sk-box" />
              </div>
            </div>
          ))}
        </div>
      </CardSectionSkeleton>
      <CardSectionSkeleton title={112} aside={56}>
        <CardRowsSkeleton rows={RECORD_ROWS} trailing={40} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64} aside={120}>
        <CardTimelineSkeleton entries={[64, 48]} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[76, 104]} />
    </RecordCardSkeleton>
  );
}
