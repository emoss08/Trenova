import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { useId } from "react";

/**
 * How one part looks: `always` in both modes, `still` as the one frame
 * reduced motion draws, `move` while it may animate. The still frame is the
 * moment the motion is about, a sheet halfway into the open drawer.
 */
type PartStyle = { always?: string; still: string; move: string };

type FilingPart = "drawer" | "stack" | "sheet";

const PARTS: Record<FilingPart, PartStyle> = {
  drawer: { still: "translate-x-[7.6px]", move: "animate-desk-drawer" },
  stack: { still: "", move: "animate-desk-file-stack" },
  sheet: {
    always: "origin-bottom [transform-box:fill-box]",
    still: "-translate-y-[3.2px]",
    move: "animate-desk-file",
  },
};

function partClass(part: FilingPart, animate: boolean): string {
  const style = PARTS[part];

  return cn(style.always, animate ? style.move : style.still);
}

export type DeskLoadingMarkProps = {
  /**
   * Whether the desk may move. Off, it is one still frame of the same
   * moment, which is what reduced motion wants.
   */
  animate?: boolean;
  className?: string;
};

/**
 * The desk being set out: an ink desk seen side on, whose drawer slides out,
 * takes a sheet of the agent's light and closes again.
 *
 * It is the lamp's companion and is drawn to the same rules, on the same
 * 28 by 22.4 grid in solid shapes, so the two sit together at the height of
 * a line of text. The desk never moves; its drawer does, and the sheet that
 * drops into it. Sheets already filed stand in the drawer, and the new one
 * joins them before the drawer closes, which is where every loop starts and
 * ends.
 *
 * The drawing never announces anything; the component around it does.
 */
export function DeskLoadingMark({ animate = true, className }: DeskLoadingMarkProps) {
  const id = useId().replace(/[^\w-]/g, "");
  const clipId = `desk-drawer-${id}`;
  const seamId = `desk-seam-${id}`;

  return (
    <svg
      viewBox="2 6 28 22.4"
      aria-hidden
      focusable="false"
      data-slot="desk-loading-mark"
      data-motion={animate ? "moving" : "still"}
      className={cn("ui-desk-mark h-4 w-5 shrink-0 overflow-visible", className)}
    >
      <defs>
        <clipPath id={clipId}>
          <rect x="2" y="-10" width="30" height="32.2" />
        </clipPath>
        <mask id={seamId} maskUnits="userSpaceOnUse" x="2" y="6" width="28" height="23">
          <rect x="2" y="6" width="28" height="23" fill="white" />
          <rect x="10" y="22.9" width="12" height="0.7" fill="black" />
        </mask>
      </defs>

      <g data-part="drawer" className={partClass("drawer", animate)}>
        <g clipPath={`url(#${clipId})`}>
          <g data-part="stack" className={cn("fill-desk-accent", partClass("stack", animate))}>
            <rect x="12.2" y="17.8" width="5" height="5" rx="0.7" className="opacity-30" />
            <rect x="13.2" y="17.6" width="5" height="5" rx="0.7" className="opacity-45" />
          </g>
          <g data-part="sheet" className={partClass("sheet", animate)}>
            <rect x="12.8" y="17.2" width="5.2" height="6.6" rx="0.7" className="fill-desk-light" />
            <g className="fill-desk-rule">
              <rect x="13.8" y="18.7" width="3.2" height="0.8" rx="0.4" />
              <rect x="13.8" y="20.2" width="2.2" height="0.8" rx="0.4" />
            </g>
          </g>
        </g>
        <rect x="11.2" y="18.4" width="8" height="3.8" rx="0.8" className="fill-desk-drawer" />
        <rect x="18.6" y="19.4" width="2" height="1.8" rx="0.9" className="fill-desk-lamp" />
      </g>

      <g data-part="desk" className="fill-desk-lamp">
        <rect x="11" y="16.4" width="8.6" height="11.6" rx="1.2" mask={`url(#${seamId})`} />
        <rect x="19.2" y="24.6" width="1.4" height="1.8" rx="0.7" />
        <rect x="4.4" y="15.6" width="2" height="12.4" rx="1" />
        <rect x="3" y="14.6" width="17.8" height="2.2" rx="1.1" />
      </g>
    </svg>
  );
}

export type DeskLoadingProps = {
  /** What is being set out, said once and shown beneath the desk. */
  label: string;
  className?: string;
  markClassName?: string;
};

/**
 * The Desk while a surface is being set out: the filing desk above the words
 * that say what is on its way.
 *
 * It stands in for a whole surface, never a row or a card; a list keeps its
 * skeleton rows, because those hold the shape of what is coming and this
 * does not. Announced, it is a status whose words are its name, so a screen
 * reader hears once what is loading and never hears the drawing move.
 */
export function DeskLoading({ label, className, markClassName }: DeskLoadingProps) {
  const reduceMotion = useReducedMotion() ?? false;

  return (
    <div
      role="status"
      data-slot="desk-loading"
      className={cn("flex flex-col items-center justify-center gap-3 px-4 py-8", className)}
    >
      <DeskLoadingMark animate={!reduceMotion} className={cn("h-10 w-12.5", markClassName)} />
      <p className="text-muted-foreground text-sm">{label}</p>
    </div>
  );
}
