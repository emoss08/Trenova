import { useT } from "@trenova/shared/i18n/use-t";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardFactGridSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { CARRIER_SECTIONS } from "./carrier-sections";

const FINDING_ROWS = [[44, 30]] as const;
const COVER_TILES = 2;
const AUTHORITY_PILLS = 4;
const BASIC_COLUMNS = 7;
const SHIPMENT_ROWS = [
  [22, 56],
  [26, 48],
  [20, 60],
] as const;
const SETTLEMENT_ROWS = [
  [24, 58],
  [22, 52],
] as const;
const CONTACT_ROWS = 1;
const FACT_CELLS = 8;

function times(count: number) {
  return Array.from({ length: count }, (_, index) => index);
}

/** The verdict banner's outline while whether the carrier can be tendered is read. */
export function VerdictSkeleton() {
  return (
    <div className="dk-ca-verdict dk-ca-verdict-sk">
      <div className="dk-ca-verdict-h">
        <i className="dk-sk dk-ca-verdict-i" />
        <SkeletonBar w={120} className="dk-rec-sk-line" />
      </div>
      <CardChecksSkeleton rows={FINDING_ROWS} />
    </div>
  );
}

/** The coverage tiles' outline while the carrier is read. */
function CoverageSkeleton() {
  return (
    <ul className="dk-ca-cover">
      {times(COVER_TILES).map((index) => (
        <li key={index}>
          <SkeletonBar w="52%" className="dk-rec-sk-label" />
          <SkeletonBar w="40%" className="dk-ca-cover-sk" />
          <SkeletonBar w="64%" className="dk-rec-sk-label" />
        </li>
      ))}
    </ul>
  );
}

/** The intelligence section's outline: the authority pills over the BASICs' columns. */
export function CarrierIntelSkeleton() {
  return (
    <>
      <div className="dk-ca-auth">
        {times(AUTHORITY_PILLS).map((index) => (
          <i key={index} className="dk-sk dk-ca-auth-sk" />
        ))}
      </div>
      <div className="dk-ca-chart">
        {times(BASIC_COLUMNS).map((index) => (
          <div key={index} className="dk-ca-col">
            <i className="dk-sk dk-ca-col-bar" />
            <SkeletonBar w="60%" className="dk-rec-sk-label" />
          </div>
        ))}
      </div>
    </>
  );
}

/** The contacts' outline: a person each, their initials beside their name and line. */
function ContactsSkeleton() {
  return (
    <ul className="dk-ca-contacts">
      {times(CONTACT_ROWS).map((index) => (
        <li key={index} className="dk-ca-person">
          <i className="dk-sk dk-rec-av dk-md" />
          <span className="dk-rec-main">
            <SkeletonBar w="34%" className="dk-rec-sk-line" />
            <SkeletonBar w="58%" className="dk-rec-sk-label" />
          </span>
        </li>
      ))}
    </ul>
  );
}

/**
 * The carrier card while its code or its carrier is read: its identifiers over
 * the name and DBA, the risk figure, whether it can be tendered as a verdict,
 * its insurance as coverage tiles, its authority as pills over the BASICs'
 * columns, its shipments and settlements as rows, its contacts with their
 * initials and its details as a grid, each where the card draws it. A
 * section the person may not read is left out here as it is on the card.
 */
export function DeskCarrierSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  const access = useCardAccess(CARRIER_SECTIONS);
  return (
    <RecordCardSkeleton label={t("Loading the carrier…")}>
      <CardHeadSkeleton
        kicker={[48, 72, 64, 56]}
        stepper={stepper}
        title={56}
        subtitle={40}
        figure
        meta={[88, 120]}
      />
      {access.eligibility && (
        <CardSectionSkeleton title={120} aside={84}>
          <VerdictSkeleton />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={64} aside={72}>
        <CoverageSkeleton />
      </CardSectionSkeleton>
      {access.intelligence && (
        <CardSectionSkeleton title={112} aside={96}>
          <CarrierIntelSkeleton />
        </CardSectionSkeleton>
      )}
      {access.shipments && (
        <CardSectionSkeleton title={112} aside={72}>
          <CardRowsSkeleton rows={SHIPMENT_ROWS} trailing={88} />
        </CardSectionSkeleton>
      )}
      {access.settlements && (
        <CardSectionSkeleton title={84} aside={72}>
          <CardRowsSkeleton rows={SETTLEMENT_ROWS} trailing={80} />
        </CardSectionSkeleton>
      )}
      <CardSectionSkeleton title={64}>
        <ContactsSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={48}>
        <CardFactGridSkeleton cells={FACT_CELLS} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[108]} />
    </RecordCardSkeleton>
  );
}
