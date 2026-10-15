import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  CardShipmentsSkeleton,
  CardTrackSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { DETENTION_OCCURRENCE_SECTIONS } from "./detention-occurrence-sections";

const RECEIPT = [58, 46, 52, 40, 62] as const;
const FACTORS = [64, 56, 60, 52] as const;

/** The clock while the charge is read: the ruler, its moments, and the three figures beneath. */
function ClockSkeleton() {
  return (
    <div className="dk-dtn-clock" aria-hidden>
      <i className="dk-sk dk-dtn-rule" />
      <ol className="dk-dtn-ticks">
        {[0, 34, 66, 100].map((left) => (
          <li
            key={left}
            className={left === 0 ? "dk-start" : left === 100 ? "dk-end" : undefined}
            style={{ left: `${left}%` }}
          >
            <SkeletonBar w="64px" className="dk-rec-sk-label" />
          </li>
        ))}
      </ol>
      <dl className="dk-dtn-sum">
        {[56, 48, 72].map((width) => (
          <div key={width}>
            <SkeletonBar w={`${width}px`} className="dk-rec-sk-label" />
            <SkeletonBar w="44px" className="dk-rec-sk-line" />
          </div>
        ))}
      </dl>
    </div>
  );
}

/** The receipt while the charge is read: a line a step. */
function ReceiptSkeleton() {
  return (
    <ol className="dk-dtn-rcpt" aria-hidden>
      {RECEIPT.map((width, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <li key={index}>
          <SkeletonBar w="64px" className="dk-rec-sk-line" />
          <SkeletonBar w={`${width}%`} className="dk-rec-sk-line" />
          <SkeletonBar w="48px" className="dk-rec-sk-line" />
        </li>
      ))}
    </ol>
  );
}

/** Whether it holds up while the charge is read: the score and its factors. */
function ProofSkeleton() {
  return (
    <div className="dk-dtn-proof" aria-hidden>
      <div className="dk-dtn-score">
        <SkeletonBar w="40px" className="dk-rec-sk-figure" />
        <SkeletonBar w="80px" className="dk-rec-sk-label" />
      </div>
      <ul className="dk-dtn-factors">
        {FACTORS.map((width, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <li key={index}>
            <SkeletonBar w={`${width}%`} className="dk-rec-sk-line" />
            <SkeletonBar w="32px" className="dk-rec-sk-label" />
          </li>
        ))}
      </ul>
    </div>
  );
}

/**
 * The detention card while its code or its charge is read: the stub and the
 * facility over the charge, its track, the clock, the receipt, whether it
 * holds up, the shipment, and the bar, each where the card draws it. A section
 * the person may not read is left out here as it is on the card.
 */
export function DeskDetentionOccurrenceSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(DETENTION_OCCURRENCE_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the detention charge…")}>
      <CardHeadSkeleton
        kicker={[64, 96, 120]}
        stepper={stepper}
        lead={<i className="dk-sk dk-rec-stub" />}
        title={48}
        subtitle={72}
        figure
        meta={[152, 120]}
      />
      <CardSectionSkeleton title={96} aside={access.shipments ? 88 : undefined}>
        <CardTrackSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64} aside={96}>
        <ClockSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={136} aside={144}>
        <ReceiptSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={96} aside={128}>
        <ProofSkeleton />
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={72}>
          <CardShipmentsSkeleton layout="route" rows={1} />
        </CardSectionSkeleton>
      )}
      <CardBarSkeleton actions={[104, 136]} />
    </RecordCardSkeleton>
  );
}
