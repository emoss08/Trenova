import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  CardBarSkeleton,
  CardHeadSkeleton,
  CardSectionSkeleton,
  RecordCardSkeleton,
  SkeletonBar,
} from "../../skeletons/record-card-skeleton";
import { LETTER_ASPECT } from "./page-geometry";

/** Where the text lines of a page stand, as shares of its width. */
const PAGE_LINES = [72, 64, 80, 40, 76, 58, 69, 33, 74, 61, 78, 45, 70, 55, 66, 38, 75, 60];

/** The check list while it is read: each item's name and value widths, in percent. */
const CHECKS = [
  [22, 54],
  [36, 40],
  [26, 48],
] as const;

/** The route's stops while it is read: name and address widths, in percent. */
const STOPS = [
  [44, 62],
  [50, 58],
] as const;

/**
 * A page while the server draws it: paper of the page's shape with a line where each run of
 * text will be, so the page lands in place instead of blinking in.
 */
export function PageSkeleton() {
  return (
    <div className="dk-ex-psk" aria-hidden>
      <div className="dk-ex-psk-head">
        <SkeletonBar w={34} className="dk-ex-psk-title" />
        <SkeletonBar w={20} className="dk-ex-psk-title" />
      </div>
      {PAGE_LINES.map((width, index) => (
        <SkeletonBar
          key={index}
          w={width}
          className={cn("dk-ex-psk-ln", index > 0 && index % 6 === 0 && "dk-ex-psk-gap")}
        />
      ))}
    </div>
  );
}

/**
 * The extraction card's outline while its code or its contents are read, in the card's own
 * layout: the head with the stub, the lane and the rate, the check list with a snippet beside
 * each item, the route, the page in its frame and the bar. On a wide screen it takes the same
 * two columns the card does.
 */
export function ExtractionSkeleton() {
  const t = useT();
  return (
    <RecordCardSkeleton label={t("Loading what was read off the document")} className="dk-ex">
      <CardHeadSkeleton
        kicker={[112, 176, 48]}
        title={52}
        subtitle={44}
        figure
        meta={[68, 96]}
        lead={<i className="dk-sk dk-rec-stub" />}
      />
      <div className="dk-ex-body">
        <div className="dk-ex-reads">
          <CardSectionSkeleton title={220} aside={104}>
            <ol className="dk-ex-check">
              {CHECKS.map(([label, value], index) => (
                <li key={index} className="dk-ex-ck">
                  <i className="dk-sk dk-ex-ck-mark" />
                  <span className="dk-ex-ck-main">
                    <SkeletonBar w={label} className="dk-rec-sk-fact" />
                    <SkeletonBar w={value} className="dk-rec-sk-sub" />
                  </span>
                  <i className="dk-sk dk-ex-snip" />
                </li>
              ))}
            </ol>
          </CardSectionSkeleton>
          <CardSectionSkeleton title={120} aside={132}>
            <ol className="dk-ex-route">
              {STOPS.map(([name, address], index) => (
                <li key={index}>
                  <i className="dk-sk dk-ex-pin" />
                  <div>
                    <SkeletonBar w={18} className="dk-rec-sk-fact" />
                    <SkeletonBar w={name} className="dk-rec-sk-sub" />
                    <SkeletonBar w={address} className="dk-rec-sk-sub" />
                  </div>
                </li>
              ))}
            </ol>
          </CardSectionSkeleton>
        </div>
        <section className="dk-rec-sec dk-ex-doc">
          <div className="dk-rec-sh">
            <SkeletonBar w="104px" className="dk-rec-sk-h3" />
            <SkeletonBar w="132px" className="dk-rec-sk-fact" />
          </div>
          <div className="dk-ex-doc-body dk-one">
            <div className="dk-ex-well">
              <div className="dk-ex-sheet" style={{ aspectRatio: LETTER_ASPECT }}>
                <PageSkeleton />
              </div>
            </div>
          </div>
        </section>
      </div>
      <CardBarSkeleton actions={[152, 120]} />
    </RecordCardSkeleton>
  );
}
