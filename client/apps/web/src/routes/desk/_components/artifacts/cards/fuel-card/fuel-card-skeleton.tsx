import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardBoxSkeleton,
  CardChecksSkeleton,
  CardFactGridSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardStatsSkeleton,
  RecordCardSkeleton,
} from "../../skeletons/record-card-skeleton";

/**
 * The fuel card card while its code or its card is read: the network's card
 * in small beside the card's name and this month's spend, the card face beside
 * who holds it and the month's figures, the latest purchases, where it fuels,
 * the feed and the card's details, each where the card draws it.
 */
export function DeskFuelCardSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the fuel card…")} className="dk-fcd">
      <CardHeadSkeleton
        kicker={[64, 60, 96]}
        stepper={stepper}
        title={52}
        subtitle={40}
        figure
        meta={[148, 112]}
        lead={<i className="dk-sk dk-fu-lead h-8.5 w-13 rounded-md" />}
      />
      <section className="dk-rec-sec dk-fcd-hero">
        <i className="dk-sk dk-fcd-face" />
        <div className="dk-fcd-side">
          <CardRowsSkeleton
            rows={[
              [36, 22],
              [28, 40],
            ]}
          />
          <CardStatsSkeleton cells={4} />
        </div>
      </section>
      <CardSectionSkeleton title={120} aside={64}>
        <CardRowsSkeleton
          rows={[
            [40, 28],
            [36, 30],
            [44, 26],
          ]}
          trailing={64}
        />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={96} aside={104}>
        <CardBoxSkeleton height={72} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64}>
        <CardChecksSkeleton rows={[[40, 56]]} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={56}>
        <CardFactGridSkeleton cells={4} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[104]} />
    </RecordCardSkeleton>
  );
}
