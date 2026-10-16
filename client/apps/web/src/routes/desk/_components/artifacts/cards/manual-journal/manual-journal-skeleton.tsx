import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardEquationSkeleton,
  CardHeadSkeleton,
  CardJournalSkeleton,
  CardSectionSkeleton,
  CardTrackSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const LINES = [
  [40, "dr"],
  [36, "dr"],
  [44, "cr"],
] as const;

/** The requester's words while the journal is read: two lines and the name under them. */
function WhySkeleton() {
  return (
    <div className="dk-mj-why" aria-hidden>
      <SkeletonBar w="92%" className="dk-rec-sk-line" />
      <SkeletonBar w="64%" className="dk-rec-sk-line" />
      <SkeletonBar w="22%" className="dk-rec-sk-label" />
    </div>
  );
}

/** Where it will post while the journal is read: one calendar leaf and its line. */
function PostingSkeleton() {
  return (
    <div className="dk-mj-post" aria-hidden>
      <div className="dk-mj-cal">
        <i className="dk-sk dk-mj-leaf" />
        <span className="dk-what">
          <SkeletonBar w="40%" className="dk-rec-sk-line" />
          <SkeletonBar w="56%" className="dk-rec-sk-label" />
        </span>
      </div>
    </div>
  );
}

/**
 * The manual journal card while its code or its journal is read: the MJ stub
 * beside what it books and its total, the track with who signed it, the
 * requester's reason, the lines written into the book, the equation, and
 * where it will post, each where the card draws it.
 */
export function DeskManualJournalSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the manual journal…")}>
      <CardHeadSkeleton
        kicker={[88, 96, 32]}
        stepper={stepper}
        title={52}
        subtitle={34}
        figure
        meta={[148, 120]}
        lead={<i className="dk-sk dk-rec-stub" />}
      />
      <CardSectionSkeleton title={96}>
        <CardTrackSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={32}>
        <WhySkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={88} aside={120}>
        <CardJournalSkeleton lines={LINES} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={152}>
        <CardEquationSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={120}>
        <PostingSkeleton />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[96, 120]} />
    </RecordCardSkeleton>
  );
}
