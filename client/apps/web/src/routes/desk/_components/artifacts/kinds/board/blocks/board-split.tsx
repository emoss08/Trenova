import type { DeskBoardMetric } from "@/lib/graphql/desk-board-card";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import { formatBoardValue } from "../board-card";
import { toNumber } from "./chart-values";
import type { DeskBoardSplit } from "./metric-types";

const PART_CLASS: Record<string, string> = {
  used: "dk-bd-used",
  over: "dk-bd-over",
  credit: "dk-bd-credit",
};

function partClass(kind: string): string {
  return PART_CLASS[kind] ?? "dk-bd-used";
}

/** Whether a split has a part to draw. */
export function splitDraws(split: DeskBoardSplit | null | undefined): split is DeskBoardSplit {
  return !!split && split.parts.length > 0;
}

/** What the split shows: the whole, and each part with its value. */
export function splitLabel(split: DeskBoardSplit, t: TranslateFn): string {
  const parts = new Intl.ListFormat(undefined, { type: "unit" }).format(
    split.parts.map((part) => t("{0} {1}", part.label, formatBoardValue(part.value))),
  );
  return t("Of {0}: {1}", formatBoardValue(split.total), parts);
}

/**
 * Parts of a whole: the headline figure large (the metric's first figure, else the whole), one
 * strip cut into its parts (used solid, over the warn hatch, credit lighter), and each part
 * keyed under it with its value. The keys stand in one line where the exhibit is wide and one
 * to a line where it is narrow, so the sketch holds the same height at every width.
 */
export function BoardSplit({
  split,
  figures,
}: {
  split: DeskBoardSplit;
  figures: DeskBoardMetric["figures"];
}) {
  const t = useT();
  const headline = figures.find((figure) => figure.value);
  const sizes = split.parts.map((part) => Math.max(0, toNumber(part.value.number) ?? 0));
  const whole = Math.max(
    toNumber(split.total.number) ?? 0,
    sizes.reduce((sum, size) => sum + size, 0),
  );
  return (
    <div className="dk-bd-whole">
      {headline?.value ? (
        <b className="dk-bd-big">
          {formatBoardValue(headline.value)}
          <small>{headline.label}</small>
        </b>
      ) : (
        <b className="dk-bd-big">{formatBoardValue(split.total)}</b>
      )}
      <div className="dk-bd-split" role="img" aria-label={splitLabel(split, t)}>
        {split.parts.map((part, index) =>
          sizes[index] > 0 && whole > 0 ? (
            <i
              // oxlint-disable-next-line react/no-array-index-key -- parts keep their order
              key={index}
              className={partClass(part.kind)}
              style={{ width: `${(sizes[index] / whole) * 100}%` }}
            />
          ) : null,
        )}
      </div>
      <div
        className={cn("dk-bd-keys", split.parts.length > 3 && "dk-bd-keys-many")}
        style={{ "--n": split.parts.length } as CSSProperties}
      >
        {split.parts.map((part, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- parts keep their order
          <span key={index}>
            <i className={cn(part.kind !== "used" && partClass(part.kind))} />
            {part.label}
            <b>{formatBoardValue(part.value)}</b>
          </span>
        ))}
      </div>
    </div>
  );
}
