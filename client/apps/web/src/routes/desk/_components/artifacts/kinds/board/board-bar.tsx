import "../../../../_styles/board-beyond.css";
import type { DeskBoardBar } from "@/lib/graphql/desk-board-card";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";

/** The cut between two segments that meet: a fill that ends where another starts. */
const MEET = 0.0005;

function percent(fraction: number): string {
  return `${Math.round(fraction * 10000) / 100}%`;
}

/**
 * A row's bar exactly as the server laid it out: one `<i>` per segment, classed by its kind
 * (`dk-bd-used`, `dk-bd-beyond`, `dk-bd-ghost`, `dk-bd-projected`, `dk-bd-tail`,
 * `dk-bd-credit`, `dk-bd-held`) and placed by its start and width; a segment that follows
 * another is cut from it (`dk-bd-next`, `dk-bd-cut`). A credit in a centred bar is placed from
 * the right. The bound is a `<b>` through the track.
 */
export function SegmentBar({ bar }: { bar: DeskBoardBar }) {
  const segments = bar.segments;
  return (
    <span className={cn("dk-bd-track", bar.centred && "dk-bd-centred")}>
      {segments.map((segment, index) => {
        const end = segment.start + segment.width;
        const follows = segments.some(
          (other, at) => at !== index && Math.abs(other.start + other.width - segment.start) < MEET,
        );
        const followed = segments.some(
          (other, at) => at !== index && Math.abs(other.start - end) < MEET,
        );
        const credit = bar.centred && segment.kind === "credit";
        const style: CSSProperties = credit
          ? { right: percent(1 - end), width: percent(segment.width) }
          : {
              left: segment.start > 0 ? percent(segment.start) : undefined,
              width: follows ? `calc(${percent(segment.width)} - 2px)` : percent(segment.width),
            };
        return (
          <i
            // oxlint-disable-next-line react/no-array-index-key -- segments keep their order
            key={index}
            className={cn(
              `dk-bd-${segment.kind}`,
              segment.tone === "warn" && "dk-warn",
              follows && !credit && "dk-bd-next",
              followed && !credit && "dk-bd-cut",
            )}
            style={style}
          />
        );
      })}
      {bar.bound !== null && bar.bound !== undefined && <b style={{ left: percent(bar.bound) }} />}
    </span>
  );
}
