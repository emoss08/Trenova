import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardRowsSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";

const WAY_STOPS = 3;
const PASS_FIELDS = 4;
const NEST_LAYERS = ["isa", "gs", "st"] as const;
const TAPE_LINES = [86, 62, 34, 70, 22, 58, 40, 30, 36, 52] as const;
const LINK_ROWS = [[30, 44]] as const;

/** The way's stops while the message is read: a ring each on the wire, its name and when. */
function WaySkeleton() {
  return (
    <ol className="dk-edi-way">
      {Array.from({ length: WAY_STOPS }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <li key={index} className="dk-edi-stop dk-ahead">
          <i className="dk-sk dk-edi-node" />
          <SkeletonBar w={72} className="dk-rec-sk-line" />
          <SkeletonBar w={96} className="dk-rec-sk-label" />
        </li>
      ))}
    </ol>
  );
}

/** The plain reading while the message is read: the headline over the stub's fields. */
function ReadingSkeleton() {
  return (
    <div className="dk-edi-read dk-edi-read-sk">
      <div className="dk-edi-read-h">
        <i className="dk-sk dk-edi-read-i" />
        <span className="dk-edi-read-t">
          <SkeletonBar w={140} className="dk-rec-sk-line" />
          <SkeletonBar w={200} className="dk-rec-sk-label" />
        </span>
      </div>
      <div className="dk-edi-pass">
        {Array.from({ length: PASS_FIELDS }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <div key={index}>
            <SkeletonBar w="48%" className="dk-rec-sk-label" />
            <SkeletonBar w="70%" className="dk-rec-sk-line" />
          </div>
        ))}
      </div>
    </div>
  );
}

/** The envelope's layers while the message is read, one inside the next. */
function NestSkeleton() {
  return NEST_LAYERS.reduceRight<ReactNode>(
    (inner, layer) => (
      <div className={`dk-edi-nest dk-edi-${layer}`}>
        <div className="dk-edi-flap">
          <SkeletonBar w={52} className="dk-rec-sk-label" />
          <SkeletonBar w={96} className="dk-rec-sk-line" />
        </div>
        {inner}
      </div>
    ),
    null,
  );
}

/** The X12 tape while it is inspected: its paper and a line for each segment. */
export function TapeSkeleton() {
  return (
    <div className="dk-edi-paper dk-edi-tape-sk" aria-busy>
      {TAPE_LINES.map((width, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} className="dk-edi-seg">
          <SkeletonBar w={14} className="dk-rec-sk-label" />
          <SkeletonBar w={`${width}%`} className="dk-rec-sk-label" />
        </div>
      ))}
    </div>
  );
}

/**
 * The EDI message card while its code or its message is read: the stamp and
 * its document over the partner, the way it travels, what it says, the
 * envelope's layers, the X12 tape and the records it names, each where the
 * card draws it.
 */
export function DeskEdiMessageSkeleton({ stepper = false }: { stepper?: boolean }) {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading the EDI message…")}>
      <CardHeadSkeleton
        kicker={[56, 48, 64]}
        stepper={stepper}
        lead={<i className="dk-sk dk-edi-stamp" />}
        title={128}
        subtitle={96}
        meta={[112, 72]}
      />
      <CardSectionSkeleton title={48}>
        <WaySkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={84}>
        <ReadingSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={60}>
        <NestSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={56} aside={120}>
        <TapeSkeleton />
      </CardSectionSkeleton>
      <CardSectionSkeleton title={104}>
        <CardRowsSkeleton rows={LINK_ROWS} />
      </CardSectionSkeleton>
      <CardBarSkeleton actions={[120]} />
    </RecordCardSkeleton>
  );
}
