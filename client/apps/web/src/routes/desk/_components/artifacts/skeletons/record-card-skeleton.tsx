import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";
import { CARD_DOCUMENT_SECTIONS, useCardAccess } from "../record-card/card-access";

/*
A record card's outline while its code or its record is read, drawn on the
card's own layout (`dk-rec-*`) with the Desk's shimmer (`dk-sk`), each bar the
size of the text it stands for. A card's skeleton is composed from these, so
nothing moves when the card lands.
*/

function width(value: number | string): CSSProperties {
  return { width: typeof value === "number" ? `${value}%` : value };
}

/** One shimmering bar the size of a line of text, a pill or a field. */
export function SkeletonBar({
  w,
  className,
}: {
  /** A percentage of the line, or a CSS length. */
  w: number | string;
  className?: string;
}) {
  return <i className={cn("dk-sk", className)} style={width(w)} />;
}

/** The card's outline: announced once to a screen reader, hidden from it otherwise. */
export function RecordCardSkeleton({ label, children }: { label?: string; children: ReactNode }) {
  const t = useT();
  return (
    <div className="dk-rec dk-rec-skel" aria-busy>
      <span role="status" className="sr-only">
        {label ?? t("Loading the record…")}
      </span>
      <div aria-hidden className="contents">
        {children}
      </div>
    </div>
  );
}

/** A card's head: the kicker, the name and its line, the figure and where it stands. */
export function CardHeadSkeleton({
  kicker,
  title,
  subtitle,
  figure = false,
  meta,
  stepper = false,
}: {
  /** The kicker's pieces, as widths in pixels. */
  kicker: readonly number[];
  title: number;
  subtitle?: number;
  figure?: boolean;
  /** The facts after the status pill, as widths in pixels. */
  meta: readonly number[];
  stepper?: boolean;
}) {
  return (
    <div className="dk-rec-head">
      <div className="dk-rec-kick">
        {kicker.map((piece, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <SkeletonBar key={index} w={`${piece}px`} className="dk-rec-sk-kick" />
        ))}
        {stepper && <SkeletonBar w="64px" className="dk-rec-sk-kick dk-rec-sk-step" />}
      </div>
      <div className="dk-rec-hrow">
        <div className="dk-rec-who dk-rec-sk-who">
          <SkeletonBar w={title} className="dk-rec-sk-title" />
          {subtitle !== undefined && <SkeletonBar w={subtitle} className="dk-rec-sk-sub" />}
        </div>
        {figure && (
          <div className="dk-rec-amt">
            <SkeletonBar w="96px" className="dk-rec-sk-figure" />
            <SkeletonBar w="72px" className="dk-rec-sk-sub" />
          </div>
        )}
      </div>
      <div className="dk-rec-meta">
        <SkeletonBar w="72px" className="dk-rec-sk-pill" />
        {meta.map((piece, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <SkeletonBar key={index} w={`${piece}px`} className="dk-rec-sk-fact" />
        ))}
      </div>
    </div>
  );
}

/** A titled section, its title and what sits beside it. */
export function CardSectionSkeleton({
  title,
  aside,
  children,
}: {
  /** The title's width in pixels. */
  title: number;
  /** What sits beside the title, as a width in pixels. */
  aside?: number;
  children: ReactNode;
}) {
  return (
    <section className="dk-rec-sec">
      <div className="dk-rec-sh">
        <SkeletonBar w={`${title}px`} className="dk-rec-sk-h3" />
        {aside !== undefined && <SkeletonBar w={`${aside}px`} className="dk-rec-sk-fact" />}
      </div>
      {children}
    </section>
  );
}

/** Rows of a label over a line, with something on the right, as a list in a card reads. */
export function CardRowsSkeleton({
  rows,
  trailing,
}: {
  /** Each row's label and line, as percentages. */
  rows: readonly (readonly [number, number])[];
  /** What each row ends in, as a width in pixels: a field, a figure. */
  trailing?: number;
}) {
  return (
    <div className="dk-rec-sk-rows">
      {rows.map(([label, line], index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} className="dk-rec-sk-row">
          <span>
            <SkeletonBar w={label} className="dk-rec-sk-label" />
            <SkeletonBar w={line} className="dk-rec-sk-line" />
          </span>
          {trailing !== undefined && (
            <SkeletonBar w={`${trailing}px`} className="dk-rec-sk-field" />
          )}
        </div>
      ))}
    </div>
  );
}

/** A ledger: lines with a figure on the right, under a rule, and their total. */
export function CardLedgerSkeleton({ lines }: { lines: readonly number[] }) {
  return (
    <div className="dk-rec-sk-ledger">
      {lines.map((line, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} className="dk-rec-sk-lr">
          <SkeletonBar w={line} className="dk-rec-sk-line" />
          <SkeletonBar w="64px" className="dk-rec-sk-line" />
        </div>
      ))}
      <div className="dk-rec-sk-lr dk-rec-sk-total">
        <SkeletonBar w="18%" className="dk-rec-sk-line" />
        <SkeletonBar w="80px" className="dk-rec-sk-line" />
      </div>
    </div>
  );
}

/** The bar held to the bottom: a hint and the card's action. */
export function CardBarSkeleton({ actions }: { actions: readonly number[] }) {
  return (
    <div className="dk-rec-bar">
      <SkeletonBar w="40%" className="dk-rec-sk-fact dk-rec-sk-hint" />
      {actions.map((action, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <SkeletonBar key={index} w={`${action}px`} className="dk-rec-sk-button" />
      ))}
    </div>
  );
}

/** A card's checks: each a mark in a circle, its name and what it found. */
export function CardChecksSkeleton({ rows }: { rows: readonly (readonly [number, number])[] }) {
  return (
    <div className="dk-rec-checks">
      {rows.map(([title, detail], index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} className="dk-rec-ck">
          <i className="dk-sk dk-rec-sk-mark" />
          <span className="dk-rec-sk-ckb">
            <SkeletonBar w={title} className="dk-rec-sk-line" />
            <SkeletonBar w={detail} className="dk-rec-sk-label" />
          </span>
        </div>
      ))}
    </div>
  );
}

/** Where a record is (CardWhere): the map's square beside the place's lines. */
export function CardWhereSkeleton() {
  return (
    <div className="dk-rec-where">
      <i className="dk-sk dk-rec-scope" />
      <div className="dk-rec-at">
        <SkeletonBar w="62%" className="dk-rec-sk-line" />
        <SkeletonBar w="44%" className="dk-rec-sk-label" />
        <SkeletonBar w="52%" className="dk-rec-sk-label" />
      </div>
    </div>
  );
}

/** A record's facts (CardFactGrid): a label over a value, two to a row. */
export function CardFactGridSkeleton({ cells }: { cells: number }) {
  return (
    <div className="dk-rec-facts">
      {Array.from({ length: cells }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index}>
          <SkeletonBar w="40%" className="dk-rec-sk-label" />
          <SkeletonBar w="68%" className="dk-rec-sk-line" />
        </div>
      ))}
    </div>
  );
}

/** A `CardStats` strip while it is read: a label over a figure in each cell. */
export function CardStatsSkeleton({ cells, className }: { cells: number; className?: string }) {
  return (
    <div className={cn("dk-rec-stats", className)}>
      {Array.from({ length: cells }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index}>
          <SkeletonBar w="46%" className="dk-rec-sk-label" />
          <SkeletonBar w="64%" className="dk-rec-sk-line" />
        </div>
      ))}
    </div>
  );
}

/** A `CardMeter` while it is read: its label and reading over the empty bar. */
export function CardMeterSkeleton() {
  return (
    <div className="dk-rec-gauge">
      <div className="dk-rec-gauge-h">
        <SkeletonBar w="20%" className="dk-rec-sk-label" />
        <SkeletonBar w="32%" className="dk-rec-sk-label" />
      </div>
      <i className="dk-sk dk-rec-meter" />
    </div>
  );
}

/** A filled block the card draws as one piece, such as a lane or a document tile. */
export function CardBoxSkeleton({ height }: { height: number }) {
  return <i className="dk-sk dk-rec-sk-box" style={{ height: `${height}px` }} />;
}

/** A card's timeline: an entry and who did it when, down a line. */
export function CardTimelineSkeleton({ entries }: { entries: readonly number[] }) {
  return (
    <div className="dk-rec-sk-log">
      {entries.map((entry, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index}>
          <i className="dk-sk dk-rec-sk-dot" />
          <span>
            <SkeletonBar w={entry} className="dk-rec-sk-line" />
            <SkeletonBar w="34%" className="dk-rec-sk-label" />
          </span>
        </div>
      ))}
    </div>
  );
}

/** Each placeholder tile's name and file line, as percentages of the tile's width. */
const DOCUMENT_TILE_LINES = [
  [72, 58],
  [64, 66],
  [78, 52],
  [58, 62],
] as const;

/**
 * A document rail while its code or its documents are read: tiles of a
 * preview over a name and a file line, in the rail's own row, so nothing
 * moves when the documents land.
 */
export function DocumentRailSkeleton() {
  return (
    <div className="dk-rec-docs dk-rec-docs-sk">
      <div className="dk-rec-docs-rail">
        {DOCUMENT_TILE_LINES.map(([name, meta], index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <div key={index} className="dk-rec-doc">
            <span className="dk-rec-doc-p">
              <i className="dk-sk" />
            </span>
            <span className="dk-rec-doc-b">
              <SkeletonBar w={name} className="dk-rec-sk-line" />
              <SkeletonBar w={meta} className="dk-rec-sk-label" />
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * A card's documents section while it is read: its title and the rail's
 * outline, left out for someone who may not read documents, as the card leaves
 * the section out.
 */
export function CardDocumentsSkeleton() {
  const access = useCardAccess(CARD_DOCUMENT_SECTIONS);
  if (!access.documents) return null;
  return (
    <CardSectionSkeleton title={72} aside={76}>
      <DocumentRailSkeleton />
    </CardSectionSkeleton>
  );
}
