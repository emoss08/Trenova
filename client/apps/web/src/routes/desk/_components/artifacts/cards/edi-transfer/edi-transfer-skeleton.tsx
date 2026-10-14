import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const GATES = 4;
const STOPS = 2;
const FACTS = 4;
const LEDGER = [62, 48] as const;
const MAPPINGS = [70, 56, 64] as const;
const LINK_ROWS = [[30, 44]] as const;

/** The decision's gates while the tender is read: a segment and a line under each. */
function GatesSkeleton() {
  return (
    <ol className="dk-tdr-gates">
      {Array.from({ length: GATES }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <li key={index} className="dk-tdr-gate dk-ahead">
          <span className="dk-tdr-bar" />
          <SkeletonBar w="56%" className="dk-rec-sk-line" />
          <SkeletonBar w="72%" className="dk-rec-sk-label" />
        </li>
      ))}
    </ol>
  );
}

/** The route while the tender is read: a pin on the rail and the stop's lines. */
function RouteSkeleton() {
  return (
    <ol className="dk-tdr-route">
      {Array.from({ length: STOPS }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <li key={index} className="dk-tdr-stop dk-pick">
          <i className="dk-tdr-pin" />
          <SkeletonBar w="30%" className="dk-rec-sk-label" />
          <SkeletonBar w="54%" className="dk-rec-sk-line" />
          <SkeletonBar w="40%" className="dk-rec-sk-label" />
        </li>
      ))}
    </ol>
  );
}

/** Freight and rate while the tender is read: the facts beside the ledger. */
function FreightSkeleton() {
  return (
    <div className="dk-tdr-freight">
      <div className="dk-tdr-load">
        <div className="dk-tdr-facts">
          {Array.from({ length: FACTS }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={index}>
              <SkeletonBar w="44%" className="dk-rec-sk-label" />
              <SkeletonBar w="70%" className="dk-rec-sk-line" />
            </div>
          ))}
        </div>
      </div>
      <div className="dk-tdr-rate">
        {LEDGER.map((width, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <SkeletonBar key={index} w={`${width}%`} className="dk-rec-sk-line" />
        ))}
      </div>
    </div>
  );
}

/** The mappings while they are read: the meter and a line for each code. */
function MappingsSkeleton() {
  return (
    <div className="dk-tdr-maps" aria-busy>
      <div className="dk-tdr-meter" />
      <div className="dk-tdr-mlist">
        {MAPPINGS.map((width, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <SkeletonBar key={index} w={`${width}%`} className="dk-rec-sk-line" />
        ))}
      </div>
    </div>
  );
}

/**
 * The EDI transfer card while its code or its tender is read: the ticket and
 * the bill of lading over the partner, the decision's gates, the route, the
 * freight and its rate, the mappings and the records it names, each where the
 * card draws it.
 */
export function DeskEdiTransferSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the EDI transfer…")}>
      <CardHeadSkeleton
        kicker={[84, 48, 40]}
        stepper={stepper}
        lead={<i className="dk-sk dk-tdr-ticket" />}
        title={36}
        subtitle={30}
        meta={[96, 52, 72]}
      />
      <CardSectionSkeleton title={56}>
        <GatesSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={40} aside={48}>
        <RouteSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={104}>
        <FreightSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={64} aside={80}>
        <MappingsSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={104}>
        <CardRowsSkeleton rows={LINK_ROWS} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[110]} />
    </RecordCardSkeleton>
  );
}
