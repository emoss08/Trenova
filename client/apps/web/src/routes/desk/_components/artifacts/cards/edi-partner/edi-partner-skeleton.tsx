import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const LANE_ROWS = [2, 1] as const;
const FIGURES = 4;
const DAYS = 14;
const DAY_HEIGHTS = [72, 70, 76, 18, 12, 74, 78, 72, 80, 76, 20, 12, 74, 46];
const MESSAGE_ROWS = [
  [34, 52],
  [30, 46],
  [36, 50],
] as const;

function times(count: number) {
  return Array.from({ length: count }, (_, index) => index);
}

/** The port's outline before the partner's ways are known. */
function PortSkeleton() {
  return <i className="dk-sk dk-edp-port dk-edp-port-sk" />;
}

/** The exchange's outline: both ends and a lane each way with its sets. */
export function ExchangeSkeleton() {
  return (
    <div className="dk-edp-x">
      <div className="dk-edp-end dk-us">
        <SkeletonBar w="54%" className="dk-rec-sk-line" />
        <SkeletonBar w="72%" className="dk-rec-sk-label" />
      </div>
      <div className="dk-edp-lanes">
        {LANE_ROWS.map((rows, lane) => (
          <div key={lane} className="dk-edp-lane">
            <div className="dk-edp-wire" />
            <ul>
              {times(rows).map((index) => (
                <li key={index}>
                  <i className="dk-sk dk-edp-stamp" />
                  <span className="dk-what">
                    <SkeletonBar w="38%" className="dk-rec-sk-line" />
                    <SkeletonBar w="56%" className="dk-rec-sk-label" />
                  </span>
                  <SkeletonBar w="36px" className="dk-rec-sk-label" />
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <div className="dk-edp-end dk-them">
        <SkeletonBar w="64%" className="dk-rec-sk-line" />
        <SkeletonBar w="80%" className="dk-rec-sk-label" />
      </div>
    </div>
  );
}

/** The traffic's outline: the figures, a pair of columns a day, and the acknowledgments' ruler. */
export function PulseSkeleton() {
  return (
    <div className="dk-edp-pulse">
      <dl className="dk-edp-figs">
        {times(FIGURES).map((index) => (
          <div key={index}>
            <SkeletonBar w="44%" className="dk-rec-sk-label" />
            <SkeletonBar w="52%" className="dk-rec-sk-line" />
          </div>
        ))}
      </dl>
      <div className="dk-edp-cols">
        {times(DAYS).map((index) => (
          <div key={index} className="dk-edp-day">
            <span>
              <i className="dk-sk dk-o" style={{ height: `${DAY_HEIGHTS[index]}%` }} />
              <i className="dk-sk dk-i" style={{ height: `${DAY_HEIGHTS[index] - 4}%` }} />
            </span>
          </div>
        ))}
      </div>
      <div className="dk-edp-ack">
        <SkeletonBar w="46%" className="dk-rec-sk-line" />
        <i className="dk-sk dk-edp-ackrule" />
      </div>
    </div>
  );
}

/** The transport's outline: its method and name, both ends, and its settings. */
function TransportSkeleton() {
  return (
    <div className="dk-edp-patch">
      <div className="dk-edp-patch-h">
        <SkeletonBar w="34px" className="dk-rec-sk-pill" />
        <SkeletonBar w="28%" className="dk-rec-sk-line" />
      </div>
      <i className="dk-sk dk-edp-ends dk-edp-ends-sk" />
      <CardRowsSkeleton
        rows={[
          [22, 48],
          [20, 40],
        ]}
      />
    </div>
  );
}

/**
 * The EDI partner card while its code or its partner is read: the partner's
 * code and kind over its name with the port before it, the delivered share,
 * what the two sides trade as two lanes between their ends, the window's
 * traffic as a column a day with the acknowledgments' ruler, the line we reach
 * them over, the latest messages and who to call, each where the card draws it.
 */
export function DeskEdiPartnerSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the EDI partner…")}>
      <CardHeadSkeleton
        kicker={[52, 92, 120]}
        stepper={stepper}
        title={34}
        subtitle={52}
        figure
        meta={[72, 112, 36]}
        lead={<PortSkeleton />}
      />
      <CardSectionSkeleton title={96} aside={64}>
        <ExchangeSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={88} aside={72}>
        <PulseSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={120} aside={96}>
        <TransportSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={104} aside={64}>
        <CardRowsSkeleton rows={MESSAGE_ROWS} trailing={72} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={76}>
        <div className="dk-edp-who">
          <i className="dk-sk dk-rec-av dk-md" />
          <span className="dk-rec-main">
            <SkeletonBar w="26%" className="dk-rec-sk-line" />
            <SkeletonBar w="40%" className="dk-rec-sk-label" />
          </span>
        </div>
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[124]} />
    </RecordCardSkeleton>
  );
}
