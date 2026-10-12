import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardDocumentsSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  CardShipmentsSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { WORKER_SECTIONS } from "./worker-sections";

const STANDING_ROWS = [[40, 56]] as const;
const QUALIFICATION_TILES = 4;
const CREDENTIAL_TILES = 2;
const CLOCKS = 4;
const RECORD_ROWS = 6;
const SHIPMENT_ROWS = 3;
/** The four clocks' rings, laid out as the card lays them. */
export function HosClocksSkeleton() {
  return (
    <div className="dk-wk-rings">
      {Array.from({ length: CLOCKS }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} className="dk-wk-ring">
          <i className="dk-sk dk-wk-ring-sk" />
          <SkeletonBar w="44%" className="dk-rec-sk-line" />
          <SkeletonBar w="36%" className="dk-rec-sk-label" />
        </div>
      ))}
    </div>
  );
}

/** The credential wallet's cards while they are read. */
export function WalletSkeleton({ tiles }: { tiles: number }) {
  return (
    <ul className="dk-wk-wallet">
      {Array.from({ length: tiles }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <li key={index}>
          <i className="dk-sk dk-wk-wallet-i" />
          <SkeletonBar w="54%" className="dk-rec-sk-line" />
          <SkeletonBar w="70%" className="dk-rec-sk-label" />
        </li>
      ))}
    </ul>
  );
}

/** The ways to reach the worker and their record while they are read. */
function ReachSkeleton() {
  return (
    <div className="dk-wk-reach">
      <div className="dk-wk-ways">
        <i className="dk-sk dk-wk-way-sk" />
        <i className="dk-sk dk-wk-way-sk" />
      </div>
      <div className="dk-wk-record">
        {Array.from({ length: RECORD_ROWS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <div key={index}>
            <SkeletonBar w="40%" className="dk-rec-sk-label" />
            <SkeletonBar w="52%" className="dk-rec-sk-line" />
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * The worker card while its code or its worker is read: the kind of worker
 * over their name and license, the drive time left, their four clocks as
 * meters, their standing and qualifications as checks, their shipments as rows
 * and their details as a grid, each where the card draws it. A section the
 * person may not read is left out here as it is on the card.
 */
export function DeskWorkerSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(WORKER_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the worker…")}>
      <CardHeadSkeleton
        kicker={[52, 64, 48]}
        lead
        stepper={stepper}
        title={56}
        subtitle={60}
        figure
        meta={[112, 84]}
      />
      <CardSectionSkeleton title={104} aside={64}>
        <HosClocksSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={56} aside={84}>
        <CardChecksSkeleton rows={STANDING_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={96} aside={72}>
        <WalletSkeleton tiles={QUALIFICATION_TILES} />
      </CardSectionSkeleton>
      {access.credentials && (
        <CardSectionSkeleton title={112} aside={72}>
          <WalletSkeleton tiles={CREDENTIAL_TILES} />
        </CardSectionSkeleton>
      )}
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardShipmentsSkeleton layout="table" rows={SHIPMENT_ROWS} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={104}>
        <ReachSkeleton />
      </CardSectionSkeleton>
      <CardDocumentsSkeleton />
      <CardBarSkeleton actions={[108]} />
    </RecordCardSkeleton>
  );
}
