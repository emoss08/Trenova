import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardFactGridSkeleton,
  CardHeadSkeleton,
  CardPathSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardStatsSkeleton,
  CardTimelineSkeleton,
  CardWhereSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { SERVICE_FAILURE_SECTIONS } from "./service-failure-sections";

const LOAD_ROWS = [
  [26, 40],
  [34, 22],
] as const;
const FAILURE_ROWS = [
  [22, 58],
  [22, 52],
] as const;
const REVIEW_STEPS = 3;
const ACTIVITY_ENTRIES = [58, 46, 52] as const;
const FACT_CELLS = 8;

/**
 * The service failure card while its code or its failure is read: its number
 * and kind over the stop it happened at and how late it was, the clock, the
 * stop on its map, the load, the cause, the review down a rail, the load's
 * other failures, its activity and its details, each where the card draws it.
 * A section the person may not read is left out here as it is on the card.
 */
export function DeskServiceFailureSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(SERVICE_FAILURE_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the service failure…")}>
      <CardHeadSkeleton
        kicker={[72, 72, 56]}
        stepper={stepper}
        title={56}
        subtitle={36}
        figure
        meta={[64, 96, 88]}
      />
      <CardSectionSkeleton title={96}>
        <CardStatsSkeleton cells={4} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={44}>
        <CardWhereSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={36}>
        <CardRowsSkeleton rows={LOAD_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={40}>
        <div className="dk-svf-cause">
          <SkeletonBar w="40%" className="dk-rec-sk-line" />
          <SkeletonBar w="72%" className="dk-rec-sk-label" />
        </div>
      </CardSectionSkeleton>
      <CardSectionSkeleton title={48}>
        <CardPathSkeleton steps={REVIEW_STEPS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={140} aside={64}>
        <CardRowsSkeleton rows={FAILURE_ROWS} trailing={56} />
      </CardSectionSkeleton>
      {access.activity && (
        <CardSectionSkeleton title={56} aside={64}>
          <CardTimelineSkeleton entries={ACTIVITY_ENTRIES} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={48}>
        <CardFactGridSkeleton cells={FACT_CELLS} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[148]} />
    </RecordCardSkeleton>
  );
}
