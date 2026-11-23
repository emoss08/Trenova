import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { isBoardExhibit, type BoardArtifact, type BoardLayoutBlock } from "../../artifact-payloads";
import type { CSSProperties, ReactNode } from "react";
import { cutRows, figureRows, halfBlocks, writtenSegments } from "./board-card";
import { LineBlank, RecordRowSketch, SourceLine } from "./board-parts";
import type { RowCut } from "./desk-board-body";
import { BoardWords } from "./board-words";

const SKETCH_ROWS = 4;
const SKETCH_FIGURES = 4;
const LOG_WIDTHS = [86, 64, 78, 52, 70];

/**
 * A list's rows while it is read, in the rows' own boxes: as many as the call
 * listed (or a few when it did not say), cut where the list will be cut, with
 * what stands after them there.
 */
function RowsSketch({ size, cut }: { size: number; cut?: RowCut }) {
  const listed = Array.from({ length: size || SKETCH_ROWS }, (_, index) => index);
  const rows = cut ? cutRows(listed, cut.limit) : { shown: listed.slice(0, SKETCH_ROWS), more: 0 };
  return (
    <div className="dk-bd-rows">
      {rows.shown.map((index) => (
        <RecordRowSketch key={index} />
      ))}
      {cut?.after(rows.more)}
    </div>
  );
}

function BlockSketch({ block, cut }: { block: BoardLayoutBlock; cut?: RowCut }) {
  switch (block.type) {
    case "records":
      return <RowsSketch size={block.size} cut={cut} />;
    case "figures":
      // The same rows the figures are drawn in, so nothing moves when they land.
      return (
        <div className="dk-bd-reads">
          {figureRows(block.size || SKETCH_FIGURES).map((size, row) => (
            <div
              // oxlint-disable-next-line react/no-array-index-key -- shimmer rows are positional
              key={row}
              className="dk-bd-wait dk-read"
              style={{ "--n": size } as CSSProperties}
            >
              {Array.from({ length: size }, (_, index) => (
                // oxlint-disable-next-line react/no-array-index-key -- shimmer slots are positional
                <span key={index}>
                  <i className="ui-shimmer" />
                  <i className="ui-shimmer" />
                </span>
              ))}
            </div>
          ))}
        </div>
      );
    case "checks":
      return (
        <div className="dk-bd-wait dk-rows">
          {Array.from({ length: Math.min(block.size || SKETCH_ROWS, SKETCH_ROWS) }, (_, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- shimmer slots are positional
            <i key={index} className="ui-shimmer" />
          ))}
        </div>
      );
    case "timeline":
      return (
        <div className="dk-bd-wait">
          {LOG_WIDTHS.map((width) => (
            <i key={width} className="ui-shimmer" style={{ width: `${width}%` }} />
          ))}
        </div>
      );
    case "meter":
      return (
        <div className="dk-bd-wait">
          <i className="dk-bd-wait-h ui-shimmer" />
          <i className="dk-bd-wait-bar ui-shimmer" />
        </div>
      );
    default:
      return <i className="dk-bd-wait-box ui-shimmer" />;
  }
}

/** Each block's exhibit numeral, counting only the blocks Trenova prints: words and buttons take none. */
export function exhibitNumerals(blocks: readonly BoardLayoutBlock[]): number[] {
  const printed = blocks.map((block) => isBoardExhibit(block.type));
  return printed.map((isPrinted, index) =>
    isPrinted ? printed.slice(0, index + 1).filter(Boolean).length : 0,
  );
}

/**
 * An exhibit while its block is read: its tab, the caption as written with a
 * blank for each reading, its source line with a blank where each part will
 * stand, and the shape of what it holds. A list is cut as `cut` cuts the one
 * drawn; `children` stand after the block, as the drawn exhibit's do.
 */
export function ExhibitSketch({
  block,
  numeral,
  half,
  className,
  cut,
  children,
}: {
  block: BoardLayoutBlock;
  numeral: number;
  half: boolean;
  className?: string;
  cut?: RowCut;
  children?: ReactNode;
}) {
  const t = useT();
  return (
    <section className={cn("dk-bd-ex", half && "dk-bd-half", className)}>
      <div className="dk-bd-tab">
        <span className="dk-bd-n">{numeral}</span>
        <span>{block.title || t("Exhibit {0}", numeral)}</span>
      </div>
      {block.caption !== "" && (
        <p className="-mt-0.5 mb-2.5 text-dsk-base leading-normal text-dsk-subtle">
          <BoardWords segments={writtenSegments(block.caption)} board={null} />
        </p>
      )}
      <SourceLine line={<LineBlank className="w-40" />} stamp={<LineBlank className="w-20" />} />
      <BlockSketch block={block} cut={cut} />
      {children}
    </section>
  );
}

/** The agent's words or buttons while the board is read: a prose block, a note, an actions block. */
export function WordsSketch({
  block,
  half,
  className,
}: {
  block: BoardLayoutBlock;
  half: boolean;
  className?: string;
}) {
  if (block.type === "actions") {
    return (
      <div className={cn("dk-bd-actions", half && "dk-bd-half", className)}>
        {Array.from({ length: Math.max(1, block.size) }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- shimmer slots are positional
          <i key={index} className="block h-7 w-36 ui-shimmer rounded-md" />
        ))}
      </div>
    );
  }
  return (
    <div
      className={cn(
        "dk-bd-wait",
        block.type === "note" ? "dk-bd-note" : "dk-bd-prose",
        half && "dk-bd-half",
        className,
      )}
    >
      <i className="ui-shimmer" />
      <i className="w-2/3 ui-shimmer" />
    </div>
  );
}

/**
 * A board while its blocks are read: its title and the shape of every block it laid out, each
 * in the place and width it will be drawn, so nothing moves when the board lands.
 */
export function BoardSkeleton({ layout }: { layout: BoardArtifact }) {
  const t = useT();
  const halves = halfBlocks(layout.blocks);
  const numerals = exhibitNumerals(layout.blocks);
  return (
    <div className="dk-bd" aria-busy="true" aria-label={t("Reading the board…")}>
      <header className="dk-bd-mast">
        <div className="dk-bd-mast-top">
          <span>{t("Board")}</span>
          <span className="dk-bd-live dk-wait">
            <i />
            {t("Reading…")}
          </span>
        </div>
        <h2 className="dk-bd-title">{layout.title}</h2>
        <div className="dk-bd-wait dk-bd-verdict">
          <i className="ui-shimmer" />
          <i className="w-3/4 ui-shimmer" />
        </div>
      </header>
      <div className="dk-bd-desk">
        {layout.blocks.map((block, index) =>
          isBoardExhibit(block.type) ? (
            <ExhibitSketch
              // oxlint-disable-next-line react/no-array-index-key -- blocks keep their place
              key={index}
              block={block}
              numeral={numerals[index]}
              half={halves[index]}
            />
          ) : (
            // oxlint-disable-next-line react/no-array-index-key -- blocks keep their place
            <WordsSketch key={index} block={block} half={halves[index]} />
          ),
        )}
      </div>
    </div>
  );
}
