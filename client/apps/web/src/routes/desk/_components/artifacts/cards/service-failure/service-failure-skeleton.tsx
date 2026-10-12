import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardPathSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
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
const CLOCK_EVENTS = 3;
const EDI_CODES = 3;

/** The clock's track while the failure is read: its stretches, and the moments between them. */
function ClockSkeleton() {
  return (
    <ol className="dk-svf-clock">
      {Array.from({ length: CLOCK_EVENTS }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <ClockStepSkeleton key={index} gap={index > 0} />
      ))}
    </ol>
  );
}

function ClockStepSkeleton({ gap }: { gap: boolean }) {
  return (
    <>
      {gap && (
        <li className="dk-svf-gap" style={{ flexGrow: 1 }}>
          <SkeletonBar w={72} className="dk-rec-sk-label dk-svf-span" />
          <i className="dk-sk" />
        </li>
      )}
      <li className="dk-svf-ev">
        <i />
        <span className="dk-svf-at">
          <SkeletonBar w={52} className="dk-rec-sk-line" />
          <SkeletonBar w={64} className="dk-rec-sk-label" />
        </span>
      </li>
    </>
  );
}

/** The 214's code tiles while the failure is read. */
function EdiSkeleton() {
  return (
    <div className="dk-svf-edi">
      {Array.from({ length: EDI_CODES }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index}>
          <SkeletonBar w="52%" className="dk-rec-sk-label" />
          <SkeletonBar w={36} className="dk-svf-edi-sk" />
          <SkeletonBar w="70%" className="dk-rec-sk-label" />
        </div>
      ))}
    </div>
  );
}

/**
 * The service failure card while its code or its failure is read: its number
 * and kind over the stop it happened at and how late it was, the clock along
 * its track, the stop on its map, the load, the cause, the review down a rail,
 * the load's other failures, its activity and the 214's codes, each where the
 * card draws it.
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
        <ClockSkeleton />
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
      <CardSectionSkeleton title={96} aside={112}>
        <EdiSkeleton />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[148]} />
    </RecordCardSkeleton>
  );
}
