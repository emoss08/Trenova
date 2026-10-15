import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardChecksSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const PARTNER_ROWS = [
  [22, 46],
  [18, 38],
  [26, 30],
  [20, 52],
] as const;
const PROFILES = 2;
const PROFILE_ROWS = [
  [24, 34],
  [18, 42],
] as const;
const SAMPLE_ROWS = [[40, 70]] as const;

/** Settings while the draft is read: a label, a value, the meter and the page tab, line by line. */
function ValueRowsSkeleton({ rows }: { rows: readonly (readonly [number, number])[] }) {
  return (
    <div className="dk-ecd-vs">
      {rows.map(([label, value], index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} className="dk-ecd-v">
          <SkeletonBar w={`${label * 4}px`} className="dk-rec-sk-label" />
          <SkeletonBar w={`${value}%`} className="dk-rec-sk-line" />
          <span />
          <i className="dk-sk dk-ecd-sure" />
          <i className="dk-sk dk-ecd-src" />
        </div>
      ))}
    </div>
  );
}

/** A document type while the draft is read: its stamp, its name and template, its settings. */
function ProfileSkeleton() {
  return (
    <div className="dk-ecd-prof">
      <div className="dk-ecd-prof-h">
        <i className="dk-sk dk-ecd-set" />
        <span className="dk-ecd-prof-n">
          <SkeletonBar w="180px" className="dk-rec-sk-line" />
          <SkeletonBar w="140px" className="dk-rec-sk-label" />
        </span>
      </div>
      <div className="dk-ecd-prof-m">
        <SkeletonBar w="64px" className="dk-rec-sk-fact" />
        <SkeletonBar w="72px" className="dk-rec-sk-fact" />
      </div>
      <ValueRowsSkeleton rows={PROFILE_ROWS} />
    </div>
  );
}

/** The outline of a drafted EDI connection, while its contents are read. */
export function ConnectionDraftSkeleton() {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the EDI connection draft…")}>
      <CardHeadSkeleton
        kicker={[124, 96, 44]}
        title={52}
        subtitle={38}
        figure
        meta={[96, 60, 84, 72]}
      />
      <CardSectionSkeleton title={96} aside={84}>
        <ValueRowsSkeleton rows={PARTNER_ROWS} />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={104}>
        {Array.from({ length: PROFILES }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <ProfileSkeleton key={index} />
        ))}
      </CardSectionSkeleton>
      <CardSectionSkeleton title={120}>
        <CardChecksSkeleton rows={SAMPLE_ROWS} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[116]} />
    </RecordCardSkeleton>
  );
}
