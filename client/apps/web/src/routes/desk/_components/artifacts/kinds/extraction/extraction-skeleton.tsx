import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { SkeletonBar } from "../../skeletons/record-card-skeleton";
import { LETTER_ASPECT } from "./page-geometry";

/** Where the text lines of a page stand, as shares of its width. */
const PAGE_LINES = [72, 64, 80, 40, 76, 58, 69, 33, 74, 61, 78, 45, 70, 55, 66, 38, 75, 60];

/** The groups of the list while it is read: a heading, then label and value widths in percent. */
const GROUPS = [
  {
    head: 22,
    rows: [
      [30, 52],
      [34, 64],
      [28, 40],
    ],
  },
  {
    head: 16,
    rows: [
      [26, 48],
      [30, 78],
      [24, 56],
    ],
  },
  {
    head: 26,
    rows: [
      [22, 36],
      [26, 44],
    ],
  },
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
 * The extraction card's outline while its code or its contents are read, drawn on the card's
 * own layout classes: the head with the file, the shipment's steps and the tally, the page in
 * its viewer, the grouped list and the actions.
 */
export function ExtractionSkeleton() {
  const t = useT();
  return (
    <div
      className="dk-ex"
      aria-busy="true"
      aria-label={t("Loading what was read off the document")}
    >
      <div aria-hidden className="contents">
        <div className="dk-ex-head">
          <div className="dk-ex-id">
            <SkeletonBar w="132px" className="dk-ex-sk-title" />
            <SkeletonBar w="76px" className="dk-ex-sk-chip dk-ex-sk-end" />
          </div>
          <SkeletonBar w="220px" className="dk-ex-sk-line" />
          <SkeletonBar w="280px" className="dk-ex-sk-line" />
          <div className="dk-ex-tally">
            <SkeletonBar w="76px" className="dk-ex-sk-chip" />
            <SkeletonBar w="132px" className="dk-ex-sk-chip" />
            <SkeletonBar w="56px" className="dk-ex-sk-chip" />
          </div>
        </div>
        <div className="dk-ex-main">
          <div className="dk-ex-viewer">
            <div className="dk-ex-bar">
              <SkeletonBar w="96px" className="dk-ex-sk-line" />
            </div>
            <div className="dk-ex-well">
              <div className="dk-ex-sheet" style={{ aspectRatio: LETTER_ASPECT }}>
                <PageSkeleton />
              </div>
            </div>
          </div>
          <div className="dk-ex-fields">
            {GROUPS.map((group, index) => (
              <div key={index} className="dk-ex-grp">
                <div className="dk-ex-grp-h">
                  <SkeletonBar w={group.head} className="dk-ex-sk-line" />
                </div>
                {group.rows.map(([label, value], row) => (
                  <div key={row} className="dk-ex-row dk-ex-sk-row">
                    <SkeletonBar w={label} className="dk-ex-sk-line" />
                    <SkeletonBar w={value} className="dk-ex-sk-value" />
                  </div>
                ))}
              </div>
            ))}
          </div>
        </div>
        <div className="dk-ex-acts">
          <div className="dk-ex-btns">
            <span className="flex-1" />
            <SkeletonBar w="96px" className="dk-ex-sk-btn" />
            <SkeletonBar w="120px" className="dk-ex-sk-btn" />
          </div>
        </div>
      </div>
    </div>
  );
}
