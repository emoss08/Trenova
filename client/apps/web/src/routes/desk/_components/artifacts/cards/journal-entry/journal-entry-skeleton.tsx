import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardEquationSkeleton,
  CardHeadSkeleton,
  CardJournalSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardTrackSkeleton,
  RecordCardSkeleton,
} from "../../skeletons/record-card-skeleton";

const LINES = [
  [44, "dr"],
  [36, "cr"],
  [40, "cr"],
] as const;

/**
 * The journal entry card while its code or its entry is read: the JE stub
 * beside the entry's name and its total, the track with who moved it, the
 * lines written into the book, the equation, and the record it came from,
 * each where the card draws it.
 */
export function DeskJournalEntrySkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the journal entry…")}>
      <CardHeadSkeleton
        kicker={[96, 64, 112]}
        stepper={stepper}
        title={48}
        subtitle={30}
        figure
        meta={[148, 120]}
        lead={<i className="dk-sk dk-rec-stub" />}
      />
      <CardSectionSkeleton title={96}>
        <CardTrackSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={88} aside={120}>
        <CardJournalSkeleton lines={LINES} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={152}>
        <CardEquationSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={120}>
        <CardRowsSkeleton rows={[[24, 52]]} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[112]} />
    </RecordCardSkeleton>
  );
}
