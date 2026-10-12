import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardMeterSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  CardTimelineSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { RATE_AGREEMENT_SECTIONS } from "./rate-agreement-sections";

const STEPS = 3;
const LIMIT_CELLS = 4;
const PARTY_ROWS = [[34, 22]] as const;
const PRICING_GROUPS = 2;
const ACTIVITY_ENTRIES = [58, 46, 52] as const;
const LANE_ROWS = [
  [30, 54],
  [26, 62],
  [34, 46],
] as const;

/**
 * The rate agreement card while its code or its agreement is read: the code
 * and kind over the name and the lanes it prices, the term's panel over the
 * approval rail, the party, the lanes as rows, and the limits, each where the
 * card draws it. A fact the person may not read is left out here as it is on
 * the card.
 */
export function DeskRateAgreementSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(RATE_AGREEMENT_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the rate agreement…")}>
      <CardHeadSkeleton
        kicker={[56, 52, 96, 18]}
        stepper={stepper}
        title={54}
        subtitle={70}
        figure
        meta={access.pricing ? [96, 132] : [96]}
      />
      <CardSectionSkeleton title={40}>
        <div className="dk-ra-now dk-ra-ended">
          <div className="dk-ra-ends">
            <div className="dk-ra-end">
              <SkeletonBar w="44px" className="dk-rec-sk-label" />
              <SkeletonBar w="96px" className="dk-rec-sk-line" />
            </div>
            <div className="dk-ra-end dk-to">
              <SkeletonBar w="32px" className="dk-rec-sk-label" />
              <SkeletonBar w="96px" className="dk-rec-sk-line" />
            </div>
          </div>
          <i className="dk-sk dk-ra-track-sk" />
          <div className="dk-ra-chips">
            <SkeletonBar w="120px" className="dk-rec-sk-pill" />
            <SkeletonBar w="96px" className="dk-rec-sk-pill" />
          </div>
        </div>
        <ol className="dk-ra-path">
          {Array.from({ length: STEPS }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <li key={index}>
              <i className="dk-sk dk-ra-node" />
              <div className="dk-ra-pb">
                <SkeletonBar w="34%" className="dk-rec-sk-line" />
                <SkeletonBar w="52%" className="dk-rec-sk-label" />
              </div>
            </li>
          ))}
        </ol>
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64}>
        <CardRowsSkeleton rows={PARTY_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={88}>
        <div className="dk-ra-mix">
          {Array.from({ length: PRICING_GROUPS }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <CardMeterSkeleton key={index} />
          ))}
        </div>
      </CardSectionSkeleton>
      <CardSectionSkeleton title={44} aside={72}>
        <CardRowsSkeleton rows={LANE_ROWS} trailing={104} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={44}>
        <div className="dk-ra-limits">
          {Array.from({ length: LIMIT_CELLS }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={index}>
              <SkeletonBar w="52%" className="dk-rec-sk-label" />
              <SkeletonBar w="68%" className="dk-rec-sk-line" />
            </div>
          ))}
        </div>
      </CardSectionSkeleton>
      {access.activity && (
        <CardSectionSkeleton title={56} aside={64}>
          <CardTimelineSkeleton entries={ACTIVITY_ENTRIES} />
        </CardSectionSkeleton>
      )}
      <CardBarSkeleton actions={[132]} />
    </RecordCardSkeleton>
  );
}
