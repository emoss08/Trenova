import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { BoardArtifact, BoardLayoutBlock } from "../../artifact-payloads";
import type { CSSProperties } from "react";
import { figureRows, halfBlocks } from "./board-card";

const SKETCH_ROWS = 4;
const SKETCH_FIGURES = 4;
const LOG_WIDTHS = [86, 64, 78, 52, 70];

function BlockSketch({ block }: { block: BoardLayoutBlock }) {
  switch (block.type) {
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
    case "records":
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
          <i className="ui-shimmer dk-bd-wait-h" />
          <i className="ui-shimmer dk-bd-wait-bar" />
        </div>
      );
    default:
      return <i className="ui-shimmer dk-bd-wait-box" />;
  }
}

/** Each block's exhibit numeral, counting only the blocks Trenova prints: words take none. */
function exhibitNumerals(blocks: readonly BoardLayoutBlock[]): number[] {
  const printed = blocks.map((block) => block.type !== "prose" && block.type !== "note");
  return printed.map((isPrinted, index) =>
    isPrinted ? printed.slice(0, index + 1).filter(Boolean).length : 0,
  );
}

/**
 * A board while its blocks are read: its title and the shape of every block it laid out, each
 * in the place and width it will be drawn, so nothing moves when the board lands.
 */
export function BoardSkeleton({ layout, question }: { layout: BoardArtifact; question: string }) {
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
        {question !== "" && (
          <div className="dk-bd-asked">
            <span>{t("You asked")}</span>
            <span className="dk-bd-bubble">{question}</span>
          </div>
        )}
        <h2 className="dk-bd-title">{layout.title}</h2>
        <div className="dk-bd-wait dk-bd-verdict">
          <i className="ui-shimmer" />
          <i className="ui-shimmer w-3/4" />
        </div>
      </header>
      <div className="dk-bd-desk">
        {layout.blocks.map((block, index) => {
          if (block.type === "prose" || block.type === "note") {
            return (
              <div
                // oxlint-disable-next-line react/no-array-index-key -- blocks keep their place
                key={index}
                className={cn(
                  "dk-bd-wait",
                  block.type === "note" ? "dk-bd-note" : "dk-bd-prose",
                  halves[index] && "dk-bd-half",
                )}
              >
                <i className="ui-shimmer" />
                <i className="ui-shimmer w-2/3" />
              </div>
            );
          }
          const numeral = numerals[index];
          return (
            // oxlint-disable-next-line react/no-array-index-key -- blocks keep their place
            <section key={index} className={cn("dk-bd-ex", halves[index] && "dk-bd-half")}>
              <div className="dk-bd-tab">
                <span className="dk-bd-n">{numeral}</span>
                <span>{block.title || t("Exhibit {0}", numeral)}</span>
              </div>
              <div className="dk-bd-src">
                <span className="ui-shimmer dk-bd-wait-src" />
              </div>
              <BlockSketch block={block} />
            </section>
          );
        })}
      </div>
    </div>
  );
}
