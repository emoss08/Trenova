import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  CardShipmentsSkeleton,
  CardTimelineSkeleton,
  CardTrackSkeleton,
  RecordCardSkeleton,
  SettlementStatementSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { DRIVER_SETTLEMENT_SECTIONS } from "./driver-settlement-sections";

/** The loads the card reads at a time; the next page is read on request. */
export const DRIVER_SETTLEMENT_SHIPMENT_ROWS = 3;
/** The activity entries the card reads at a time. */
export const DRIVER_SETTLEMENT_ACTIVITY_ROWS = 5;

const FALL = [88, 24, 18, 72] as const;
const STATEMENT = [40, 34, 30, 44] as const;
const ACTIVITY_ENTRIES = [64, 52, 58] as const;

/** How it adds up while the settlement is read: a lane for each step, the net under a rule. */
function WaterfallSkeleton() {
  return (
    <ol className="dk-pay-fall" aria-hidden>
      {FALL.map((width, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <li key={index} className={index === FALL.length - 1 ? "dk-total" : "dk-in"}>
          <span className="dk-what">
            <SkeletonBar w="72px" className="dk-rec-sk-line" />
          </span>
          <span className="dk-lane">
            <i className="dk-bar dk-sk" style={{ left: 0, width: `${width}%` }} />
          </span>
          <span className="dk-amt">
            <SkeletonBar w="64px" className="dk-rec-sk-line" />
          </span>
        </li>
      ))}
    </ol>
  );
}

/**
 * The worker settlement card while its code or its settlement is read: the
 * stub and the worker over the loads and miles it pays, its track, how it
 * adds up, the statement, the loads it pays and its activity, each where the
 * card draws it. A section the person may not read is left out here as it is
 * on the card.
 */
export function DeskDriverSettlementSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(DRIVER_SETTLEMENT_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the settlement…")}>
      <CardHeadSkeleton
        kicker={[84, 88, 56]}
        stepper={stepper}
        lead={<i className="dk-sk dk-rec-stub" />}
        title={44}
        subtitle={64}
        figure
        meta={[104, 88]}
      />
      <CardSectionSkeleton title={96} aside={access.worker ? 72 : undefined}>
        <CardTrackSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={88}>
        <WaterfallSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={32} aside={48}>
        <SettlementStatementSkeleton rows={STATEMENT} />
      </CardSectionSkeleton>
      {access.shipments && (
        <CardSectionSkeleton title={96} aside={56}>
          <CardShipmentsSkeleton layout="route" rows={DRIVER_SETTLEMENT_SHIPMENT_ROWS} />
        </CardSectionSkeleton>
      )}
      {access.activity && (
        <CardSectionSkeleton title={56} aside={64}>
          <CardTimelineSkeleton entries={ACTIVITY_ENTRIES} />
        </CardSectionSkeleton>
      )}
      <CardBarSkeleton actions={[96, 132]} />
    </RecordCardSkeleton>
  );
}
