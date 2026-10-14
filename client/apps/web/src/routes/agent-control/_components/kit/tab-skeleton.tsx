import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";

/**
 * The pieces AI control's tab skeletons are drawn from. A line of text is a bar inside the
 * element that would hold the text, so it takes that element's size and line height and
 * the row is as tall as the loaded one; a mark, a control or a switch is a block of its
 * own size. Each piece sits in the page's own classes (`.hero`, `.figs`, `.sh2`, `.seg`),
 * so a skeleton takes the room the loaded tab takes. The page imports this module up
 * front, so it holds nothing but markup.
 */

type Width = number | string;

type SkTextProps = {
  w: Width;
  className?: string;
};

/**
 * One line of text: a bar at the surrounding text's size that takes exactly one of its
 * lines, so a bar inside `.lbl` or `.big` holds the line the label or the figure will.
 */
export function SkText({ w, className }: SkTextProps) {
  return <span className={cn("tsk-t ui-shimmer", className)} style={{ width: w }} />;
}

/** A paragraph: one bar per line, each on a line of the paragraph's height. */
export function SkLines({ widths }: { widths: readonly Width[] }) {
  return widths.map((width, index) => (
    <span key={index} className="tsk-ln">
      <SkText w={width} />
    </span>
  ));
}

type SkBlockProps = {
  w: Width;
  h: Width;
  /** A radius in pixels, or "full" for a pill or a circle; the control radius otherwise. */
  r?: number | "full";
  className?: string;
  style?: CSSProperties;
};

/** A mark, a control, a tile or a switch, at its own size. */
export function SkBlock({ w, h, r, className, style }: SkBlockProps) {
  return (
    <span
      className={cn("tsk-b ui-shimmer", r === "full" && "tsk-round", className)}
      style={{ width: w, height: h, borderRadius: typeof r === "number" ? r : undefined, ...style }}
    />
  );
}

/** An icon-only button: the glyph's size, inside the button's square. */
export function SkIconButton({ glyph = 13 }: { glyph?: number }) {
  return <SkBlock w={glyph} h={glyph} r={3} style={{ margin: (28 - glyph) / 2 }} />;
}

/** An on/off switch. */
export function SkSwitch() {
  return <SkBlock w={30} h={18} r="full" />;
}

/** A search box at the toolbar's size, or the large one a page leads with. */
export function SkSearch({ large = false }: { large?: boolean }) {
  return <SkBlock w="100%" h={large ? 36 : 34} className={cn("tsk-search", large && "lg")} />;
}

const HERO_LINES: readonly Width[] = ["91.667%", "66.667%"];

type HeroSkeletonProps = {
  /** Each line of Nova's sentence, as a share of its measure. */
  lines?: readonly Width[];
  /** The one action beside the sentence, when the tab always has one: its width and its note's. */
  control?: { button: number; note?: number };
};

/**
 * Nova's sentence at the head of a tab: the ring, Nova's name and the tab's, then the
 * sentence drawn the way it draws itself while it is read, and the control beside it.
 */
export function HeroSkeleton({ lines = HERO_LINES, control }: HeroSkeletonProps) {
  return (
    <section className="hero rh" aria-hidden>
      <div className="hero-s">
        <span className="who">
          <SkBlock w={20} h={20} r="full" />
          <SkBlock w={34} h={10} />
          <SkBlock w={72} h={10} />
        </span>
        <div className="mt-2.5 flex flex-col gap-2">
          {lines.map((width, index) => (
            <Skeleton key={index} className="h-6" style={{ width }} />
          ))}
        </div>
      </div>
      {control && (
        <div className="hc">
          <SkBlock w={control.button} h={36} />
          {control.note !== undefined && (
            <span>
              <SkText w={control.note} />
            </span>
          )}
        </div>
      )}
    </section>
  );
}

/**
 * One figure's three lines, by width: the label, the value and the note under it, and
 * whether the note carries a meter above it, as a figure measured against a budget does.
 */
export type FigSkeleton = readonly [label: number, value: number, sub: number, meter?: boolean];

/** A row of figures separated by rules, each line where the loaded one lands. */
export function FigsSkeleton({ items }: { items: readonly FigSkeleton[] }) {
  return (
    <section className="figs" aria-hidden style={{ "--n": items.length } as CSSProperties}>
      {items.map(([label, value, sub, meter], index) => (
        <div key={index} className="fg">
          <span className="lbl">
            <SkText w={label} />
          </span>
          <span className="big">
            <SkText w={value} />
          </span>
          <span className="sub">
            {meter && <span className="ubar bud" />}
            <SkText w={sub} />
          </span>
        </div>
      ))}
    </section>
  );
}

type SecHeadSkeletonProps = {
  title: number;
  /** The count or note beside the heading. */
  note?: number;
  /** What sits at the far end: a note in the small type, or a button of this width. */
  end?: { note: number } | { button: number };
};

/** A section heading: the title, a note beside it, and what sits at the far end. */
export function SecHeadSkeleton({ title, note, end }: SecHeadSkeletonProps) {
  return (
    <header className="sh2" aria-hidden>
      <span className="tsk-sh">
        <SkText w={title} />
      </span>
      {note !== undefined && (
        <em>
          <SkText w={note} />
        </em>
      )}
      <span className="sp" />
      {end &&
        ("note" in end ? (
          <span className="sh2-n">
            <SkText w={end.note} />
          </span>
        ) : (
          <SkBlock w={end.button} h={28} className="self-center" />
        ))}
    </header>
  );
}

/** A segmented choice, one block per option at the option's width. */
export function SegSkeleton({ widths }: { widths: readonly number[] }) {
  return (
    <div className="seg" aria-hidden>
      {widths.map((width, index) => (
        <SkBlock key={index} w={width} h={24} />
      ))}
    </div>
  );
}
