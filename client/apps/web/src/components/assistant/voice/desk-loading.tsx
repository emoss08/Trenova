import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { useId, type CSSProperties } from "react";
import { CONE, HINGE, SHADE } from "./desk-thinking";
import {
  GROUND_Y,
  HIP_Y,
  LAMP_X,
  PILE,
  REACH_ANGLE,
  REACH_LEAN,
  SHOULDER_Y,
  VISITOR_X,
} from "./desk-visitor-motion";
import "./desk-visitor.css";

/** The desk's lamp: the working-line lamp at half size, moved over the pile. */
const LAMP = `translate(${LAMP_X} 0.9) scale(0.5)`;

const LIMB = "origin-top [transform-box:fill-box]";

type VisitorPart =
  | "walk"
  | "fade"
  | "turn"
  | "bob"
  | "lean"
  | "leg-front"
  | "leg-back"
  | "arm-front"
  | "arm-back"
  | "sheet-x"
  | "sheet-y"
  | "sheet-fade"
  | "pile"
  | "pool"
  | "cone";

/**
 * The one frame reduced motion draws, the sheet touching down on the pile:
 * the visitor at the desk leaning into the reach, every other part at rest.
 */
const STILL: Partial<Record<VisitorPart, CSSProperties>> = {
  lean: {
    transform: `translate(${VISITOR_X}px, ${GROUND_Y}px) rotate(${REACH_LEAN}deg) translate(-${VISITOR_X}px, -${GROUND_Y}px)`,
  },
  "arm-front": { transform: `rotate(${REACH_ANGLE}deg)` },
};

export type DeskLoadingMarkProps = {
  /**
   * Whether the visitor may move. Off, it is one still frame of the same
   * moment, which is what reduced motion wants.
   */
  animate?: boolean;
  className?: string;
};

/**
 * The Desk being set out: someone carries a sheet to the desk, lays it on
 * the pile under the lamp and walks off, and the pile sinks by a sheet as
 * they go so the loop meets itself.
 *
 * It is the lamp's companion and is drawn to the same rules, in solid ink
 * shapes with the agent's light, and the lamp itself stands on the desk.
 * Every part moves on one clock (see `desk-visitor-motion.ts`): the sheet
 * rides in the hand, the steps follow the distance covered, and the lamp's
 * pool brightens as the sheet lands. The drawing clips at its edges, so the
 * visitor walks in and out of frame.
 *
 * The drawing never announces anything; the component around it does.
 */
export function DeskLoadingMark({ animate = true, className }: DeskLoadingMarkProps) {
  const clipId = `desk-visitor-${useId().replace(/[^\w-]/g, "")}`;
  const clip = `url(#${clipId})`;
  const part = (name: VisitorPart, base?: string) => ({
    "data-part": name,
    className: cn(base, animate && `animate-desk-visitor-${name}`),
    style: animate ? undefined : STILL[name],
  });

  return (
    <svg
      viewBox="5 0 40 24"
      aria-hidden
      focusable="false"
      data-slot="desk-loading-mark"
      data-motion={animate ? "moving" : "still"}
      className={cn("ui-desk-mark h-6 w-10 shrink-0 overflow-hidden", className)}
    >
      <defs>
        <clipPath id={clipId}>
          <rect x="-10" y="-10" width="70" height="24" />
        </clipPath>
      </defs>

      <g data-part="desk" className="fill-desk-lamp">
        <rect x="6.4" y="14" width="16.2" height="1.6" rx="0.8" />
        <rect x="7.8" y="15" width="1.4" height="8" rx="0.7" />
        <rect x="13.4" y="15" width="7.6" height="8" rx="0.9" />
        <g className="fill-desk-drawer">
          <rect x="14.2" y="16" width="6" height="3" rx="0.5" />
          <rect x="14.2" y="19.8" width="6" height="2.4" rx="0.5" />
        </g>
        <rect x="16.4" y="17.2" width="1.6" height="0.6" rx="0.3" />
        <rect x="16.4" y="20.7" width="1.6" height="0.6" rx="0.3" />
      </g>

      <g data-part="lamp">
        <g clipPath={clip}>
          <g transform={LAMP}>
            <g transform={HINGE}>
              <path d={CONE} opacity="0.2" {...part("cone", "fill-desk-accent")} />
            </g>
          </g>
        </g>
        <ellipse
          cx={PILE.centre}
          cy="14.25"
          rx="3.6"
          ry="0.45"
          opacity="0.8"
          {...part("pool", "fill-desk-light origin-center [transform-box:fill-box]")}
        />
        <g transform={LAMP} className="fill-desk-lamp">
          <rect x="5.4" y="23.4" width="8" height="2.8" rx="1.4" />
          <path
            d="M9.4 23.6 L7.8 14.6 L15.6 9.4"
            fill="none"
            strokeWidth="2.2"
            strokeLinecap="round"
            strokeLinejoin="round"
            className="stroke-desk-lamp"
          />
          <circle cx="7.8" cy="14.6" r="1.7" />
          <g transform={HINGE}>
            <path d={SHADE} strokeWidth="1" strokeLinejoin="round" className="stroke-desk-lamp" />
            <circle cx="0" cy="-1.3" r="1.3" />
            <ellipse cx="0" cy="3.7" rx="2.3" ry="0.95" className="fill-desk-light" />
          </g>
        </g>
      </g>

      <g clipPath={clip} className="fill-desk-light">
        <g {...part("pile")}>
          {PILE.rows.map((y) => (
            <rect key={y} x={PILE.x} y={y} width={PILE.width} height="0.8" rx="0.4" />
          ))}
        </g>
        <g {...part("sheet-fade")}>
          <g {...part("sheet-x")}>
            <g {...part("sheet-y")}>
              <rect
                data-part="sheet"
                x={PILE.x}
                y={PILE.sheetY}
                width={PILE.width}
                height="0.8"
                rx="0.4"
              />
            </g>
          </g>
        </g>
      </g>

      <g {...part("fade")}>
        <g {...part("walk")}>
          <g {...part("turn")}>
            <g {...part("bob")}>
              <g {...part("lean")}>
                <rect
                  x={VISITOR_X - 0.75}
                  y={HIP_Y}
                  width="1.5"
                  height="6.4"
                  rx="0.75"
                  {...part("leg-back", cn(LIMB, "fill-desk-drawer"))}
                />
                <rect
                  x={VISITOR_X - 0.6}
                  y={SHOULDER_Y}
                  width="1.2"
                  height="5"
                  rx="0.6"
                  {...part("arm-back", cn(LIMB, "fill-desk-drawer"))}
                />
                <g className="fill-desk-lamp">
                  <rect x={VISITOR_X - 1.7} y="10.4" width="3.4" height="7" rx="1.7" />
                  <circle cx={VISITOR_X - 0.3} cy="8.1" r="1.75" />
                </g>
                <rect
                  x={VISITOR_X - 0.75}
                  y={HIP_Y}
                  width="1.5"
                  height="6.4"
                  rx="0.75"
                  {...part("leg-front", cn(LIMB, "fill-desk-lamp"))}
                />
                <rect
                  x={VISITOR_X - 0.6}
                  y={SHOULDER_Y}
                  width="1.2"
                  height="5"
                  rx="0.6"
                  {...part("arm-front", cn(LIMB, "fill-desk-lamp"))}
                />
              </g>
            </g>
          </g>
        </g>
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
 * The Desk while a surface is being set out: the visitor at the desk above
 * the words that say what is on its way.
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
      <DeskLoadingMark animate={!reduceMotion} className={cn("h-15 w-25", markClassName)} />
      <p className="text-muted-foreground text-sm">{label}</p>
    </div>
  );
}
