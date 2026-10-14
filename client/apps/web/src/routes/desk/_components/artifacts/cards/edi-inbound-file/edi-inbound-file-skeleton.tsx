import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const STAGES = 3;
const ENTRIES = [
  [44, 60],
  [40, 52],
] as const;
const FOOT_FIELDS = 3;

/** The intake while the file is read: three chevrons and a line under each. */
function IntakeSkeleton() {
  return (
    <ol className="dk-eif-intake">
      {Array.from({ length: STAGES }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <li key={index} className="dk-eif-stage dk-ahead">
          <span className="dk-eif-chev">
            <SkeletonBar w="56px" className="dk-rec-sk-label" />
          </span>
          <SkeletonBar w="64%" className="dk-rec-sk-label" />
        </li>
      ))}
    </ol>
  );
}

/** The routing label while the file is read: from, to, the interchange, and the label's foot. */
function LabelSkeleton() {
  return (
    <div className="dk-eif-label">
      <div className="dk-eif-addr">
        {[0, 1].map((side) => (
          <div key={side} className="dk-eif-party">
            <SkeletonBar w="32px" className="dk-rec-sk-label" />
            <SkeletonBar w="110px" className="dk-rec-sk-line" />
            <SkeletonBar w="84px" className="dk-rec-sk-label" />
          </div>
        ))}
        <div className="dk-eif-icn">
          <SkeletonBar w="64px" className="dk-rec-sk-label" />
          <SkeletonBar w="92px" className="dk-rec-sk-line" />
        </div>
      </div>
      <div className="dk-eif-foot">
        {Array.from({ length: FOOT_FIELDS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <div key={index}>
            <SkeletonBar w="40%" className="dk-rec-sk-label" />
            <SkeletonBar w="76%" className="dk-rec-sk-line" />
          </div>
        ))}
      </div>
    </div>
  );
}

/** What the file carried while it is inspected: the ruler and a group of rows. */
export function ManifestSkeleton() {
  return (
    <div className="dk-eif-groups" aria-busy>
      <div className="dk-eif-ruler">
        <div className="dk-eif-runs dk-sk" />
      </div>
      <div className="dk-eif-group">
        <div className="dk-eif-gh">
          <SkeletonBar w="28px" className="dk-rec-sk-label" />
          <SkeletonBar w="96px" className="dk-rec-sk-line" />
        </div>
        <div className="dk-eif-entries">
          {ENTRIES.map(([title, sub], index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={index} className="dk-eif-row">
              <i className="dk-sk dk-eif-set" />
              <span className="dk-rec-main">
                <SkeletonBar w={`${title}%`} className="dk-rec-sk-line" />
                <SkeletonBar w={`${sub}%`} className="dk-rec-sk-label" />
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/** The interchange's checks while it is inspected. */
export function EnvelopeSkeleton() {
  return (
    <div className="dk-eif-checks" aria-busy>
      <SkeletonBar w="180px" className="dk-rec-sk-label" />
      <SkeletonBar w="140px" className="dk-rec-sk-label" />
      <SkeletonBar w="120px" className="dk-rec-sk-label" />
    </div>
  );
}

/**
 * The EDI inbound file card while its code or its file is read: the folded
 * sheet and its name over the partner, the intake's chevrons, the routing
 * label, what the file carried and its interchange's checks, each where the
 * card draws it.
 */
export function DeskEdiInboundFileSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the EDI inbound file…")}>
      <CardHeadSkeleton
        kicker={[72, 48, 36]}
        stepper={stepper}
        lead={<i className="dk-sk dk-eif-glyph" />}
        title={64}
        subtitle={36}
        meta={[96, 160]}
      />
      <CardSectionSkeleton title={48}>
        <IntakeSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={88}>
        <LabelSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={104} aside={72}>
        <ManifestSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={80} aside={96}>
        <EnvelopeSkeleton />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[96]} />
    </RecordCardSkeleton>
  );
}
