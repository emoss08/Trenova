import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardBoxSkeleton,
  CardChecksSkeleton,
  CardDocumentsSkeleton,
  CardHeadSkeleton,
  CardPathSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
} from "../../skeletons/record-card-skeleton";

const CHECKS = [
  [44, 36],
  [32, 58],
  [40, 50],
] as const;

/**
 * The fuel purchase card while its code or its purchase is read: the fuel's
 * stub beside the vendor and the total, the receipt beside the IFTA panel, the
 * checks, how it was filed, and the card it was bought on, each where the card
 * draws it.
 */
export function DeskFuelPurchaseSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the fuel purchase…")} className="dk-fpu">
      <CardHeadSkeleton
        kicker={[84, 88, 120]}
        stepper={stepper}
        title={44}
        subtitle={52}
        figure
        meta={[152, 44, 72]}
        lead={<i className="dk-sk dk-rec-stub" />}
      />
      <section className="dk-rec-sec dk-fpu-hero">
        <CardBoxSkeleton height={276} />
        <CardBoxSkeleton height={224} />
      </section>
      <CardSectionSkeleton title={56} aside={84}>
        <CardChecksSkeleton rows={CHECKS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={112} aside={88}>
        <CardPathSkeleton steps={2} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64}>
        <CardRowsSkeleton rows={[[44, 30]]} trailing={56} />
      </CardSectionSkeleton>
      <CardDocumentsSkeleton />
      <CardBarSkeleton actions={[124]} />
    </RecordCardSkeleton>
  );
}
