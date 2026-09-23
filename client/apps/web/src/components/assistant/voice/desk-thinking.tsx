import { useT } from "@trenova/shared/i18n/use-t";
import type { PartOfDay } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { useEffect, useState, type CSSProperties } from "react";
import type { DeskPose, DeskWorkingPose } from "./desk-pose";

/** How long the closing pose holds before the mark is gone: the confirm spring and a beat. */
export const DESK_SETTLE_MS = 420;

/**
 * Three sizes on one 40 by 32 drawing. The stroke is set per size in the
 * drawing's own units so the line lands near the same weight on screen as an
 * icon's at every size, and the small details (a drawer, the wheels) are only
 * drawn where there are pixels to draw them with.
 */
const SIZES = {
  inline: { box: "h-4 w-5", stroke: 2.4, detail: false },
  lg: { box: "h-8 w-10", stroke: 1.5, detail: true },
  xl: { box: "h-12 w-15", stroke: 1.15, detail: true },
} as const;

export type DeskMarkSize = keyof typeof SIZES;

/** The chair's distance out from under the desk, in the drawing's units. */
const CHAIR_OUT = "translate3d(3.5px, 0, 0)";

const SCREEN_OPACITY: Record<DeskPose, string> = {
  arrive: "var(--desk-screen-rest)",
  busy: "var(--desk-screen-lit)",
  write: "var(--desk-screen-lit)",
  settle: "0",
  idle: "var(--desk-screen-rest)",
};

export type DeskMarkProps = {
  pose: DeskPose;
  size?: DeskMarkSize;
  /**
   * Whether the pose may move. Off, every pose is a still drawing of the same
   * moment, which is what reduced motion and a list of rows both want.
   */
  animate?: boolean;
  /** Draws the sky and the lamp for this part of the day: the Desk's greeting. */
  timeOfDay?: PartOfDay;
  /** The page has just opened: the chair settles in and the lamp switches on, once. */
  welcome?: boolean;
  className?: string;
};

/**
 * A desk drawn in the ink: a monitor on it, a pedestal of drawers, and a
 * chair that rolls in and out from under it.
 *
 * It is line work in `currentColor`, so it takes the colour of the text it
 * sits beside and follows the theme with nothing of its own. The one tinted
 * detail is the screen, which glows in the agent's accent where one is set
 * and in the ink where none is. The lamp's cone is the only warm light, and
 * it is only on in the evening.
 *
 * The drawing never announces anything; the component around it does.
 */
export function DeskMark({
  pose,
  size = "inline",
  animate = true,
  timeOfDay,
  welcome = false,
  className,
}: DeskMarkProps) {
  const { box, stroke, detail } = SIZES[size];
  const move = animate;
  const lamp = timeOfDay === "evening";

  const chairClass = cn(
    "transition-transform duration-500 ease-settle",
    move && pose === "arrive" && "animate-desk-roll",
    move && pose === "settle" && "animate-desk-settle",
    move && welcome && pose === "idle" && "animate-desk-arrive",
  );
  const chairStyle: CSSProperties | undefined =
    pose === "arrive" && !move ? { transform: CHAIR_OUT } : undefined;

  return (
    <svg
      viewBox="0 0 40 32"
      fill="none"
      stroke="currentColor"
      strokeWidth={stroke}
      strokeLinecap="round"
      strokeLinejoin="round"
      overflow="visible"
      aria-hidden
      focusable="false"
      data-slot="desk-mark"
      data-pose={pose}
      data-motion={move ? "moving" : "still"}
      className={cn("shrink-0", box, className)}
    >
      {timeOfDay && <DeskSky timeOfDay={timeOfDay} rise={move && welcome} />}

      <g
        data-part="desk"
        className={cn(
          "origin-bottom [transform-box:fill-box]",
          move && pose === "busy" && "animate-desk-shake",
        )}
      >
        {lamp && (
          <path
            data-part="lamp-light"
            d="M7.4 8.8L11.2 7.8L13 15.5H6Z"
            stroke="none"
            className={cn("fill-[var(--desk-lamp-light)]", move && welcome && "animate-desk-lamp")}
          />
        )}
        <rect
          data-part="screen"
          x="13"
          y="4.5"
          width="11"
          height="8"
          rx="1"
          stroke="none"
          className={cn(
            "fill-[var(--agent-accent,currentColor)] transition-opacity duration-300 ease-settle",
            move && pose === "write" && "animate-desk-screen",
          )}
          style={{ opacity: SCREEN_OPACITY[pose] }}
        />
        <rect x="13" y="4.5" width="11" height="8" rx="1" />
        <path d="M18.5 12.5V15.5M2 15.5H29M4 15.5V29.5H11V15.5M26.5 15.5V29.5" />
        {detail && <path d="M4 21.5H11M6.5 18.5H8.5" />}
        {lamp && <path d="M5.5 15.5V9.5L8 7M8.2 5.6L7.4 8.8L11.2 7.8Z" />}
      </g>

      <g data-part="chair" className={chairClass} style={chairStyle}>
        <path d="M27.5 21H34M34 21L35.5 12M30.75 21V26.5M27.5 26.5H34" />
        {detail && (
          <>
            <circle cx="28" cy="28.6" r="1" />
            <circle cx="33.5" cy="28.6" r="1" />
          </>
        )}
      </g>
    </svg>
  );
}

/** Eight short rays around a small sun, drawn as one path. */
function sunRays(cx: number, cy: number): string {
  const segments: string[] = [];
  for (let step = 0; step < 8; step += 1) {
    const angle = (step * Math.PI) / 4;
    const cos = Math.cos(angle);
    const sin = Math.sin(angle);
    segments.push(
      `M${(cx + cos * 2.9).toFixed(2)} ${(cy + sin * 2.9).toFixed(2)}` +
        `L${(cx + cos * 3.8).toFixed(2)} ${(cy + sin * 3.8).toFixed(2)}`,
    );
  }

  return segments.join("");
}

const MORNING_RAYS = sunRays(35, 7);
const AFTERNOON_RAYS = sunRays(32.5, 4);

/**
 * The part of the day, up in the corner of the drawing: a sun low in the
 * morning and high in the afternoon, and a moon once the lamp is on. It is
 * drawn a step lighter than the desk, because it is the room's weather and
 * not its furniture.
 */
function DeskSky({ timeOfDay, rise }: { timeOfDay: PartOfDay; rise: boolean }) {
  const className = cn("opacity-60", rise && "animate-rise");
  const style: CSSProperties | undefined = rise ? { animationDelay: "360ms" } : undefined;

  if (timeOfDay === "evening") {
    return (
      <path
        data-part="moon"
        d="M37 3.2A3.2 3.2 0 1 0 37 9.2A3 3 0 1 1 37 3.2Z"
        className={className}
        style={style}
      />
    );
  }

  const morning = timeOfDay === "morning";

  return (
    <g data-part="sun" className={className} style={style}>
      <circle cx={morning ? 35 : 32.5} cy={morning ? 7 : 4} r="1.8" />
      <path d={morning ? MORNING_RAYS : AFTERNOON_RAYS} />
    </g>
  );
}

export type ThinkingPresence = "working" | "settling" | "gone";

/**
 * Whether the working mark is on screen, and in which state.
 *
 * It is there while the work is, and the moment the work stops it takes the
 * closing pose for one beat and is gone. A mark that may not move — reduced
 * motion, or a still mark in a row — has no closing pose to show, so it goes
 * at once.
 */
export function useThinkingPresence(working: boolean, canSettle: boolean): ThinkingPresence {
  const [settling, setSettling] = useState(false);
  const [wasWorking, setWasWorking] = useState(working);

  if (wasWorking !== working) {
    setWasWorking(working);
    setSettling(!working && canSettle);
  }

  useEffect(() => {
    if (!settling) {
      return;
    }
    const timer = window.setTimeout(() => setSettling(false), DESK_SETTLE_MS);

    return () => window.clearTimeout(timer);
  }, [settling]);

  if (working) {
    return "working";
  }

  return settling ? "settling" : "gone";
}

export type DeskThinkingProps = {
  /** Whether the agent is at work. The mark is absent when it is not. */
  working: boolean;
  pose?: DeskWorkingPose;
  size?: Exclude<DeskMarkSize, "xl">;
  /**
   * Keeps the drawing and drops the motion. A list marks many rows at once
   * and stays on screen while the person works elsewhere in it, so a desk
   * moving on each would be a column of motion on a working screen; there the
   * mark says "under way" by being there.
   */
  still?: boolean;
  /**
   * The words beside the mark already say what is happening, so the mark is
   * hidden from assistive technology rather than announced twice.
   */
  decorative?: boolean;
  className?: string;
};

/**
 * The live indicator: the drawn desk at work, for as long as an agent is.
 *
 * This is the only loop in the product, and it earns the exception the way a
 * heartbeat monitor does — it moves because the thing it describes is still
 * going, and it stops the instant that stops. Then the chair tucks in on the
 * confirm spring and the mark is gone.
 *
 * It is one shape in one place so that "something is happening" reads the
 * same in the working line, on a tool step and anywhere else it turns up.
 *
 * Announced, it is a status with a fixed name and no content that changes,
 * so a screen reader hears that the work is under way once and is never
 * told about the drawing moving.
 */
export function DeskThinking({
  working,
  pose = "arrive",
  size = "inline",
  still = false,
  decorative = false,
  className,
}: DeskThinkingProps) {
  const t = useT();
  const reduceMotion = useReducedMotion() ?? false;
  const animate = !still && !reduceMotion;
  const presence = useThinkingPresence(working, animate);

  if (presence === "gone") {
    return null;
  }

  const settling = presence === "settling";
  const mark = <DeskMark pose={settling ? "settle" : pose} size={size} animate={animate} />;
  const frame = cn("inline-flex shrink-0 items-center justify-center", className);

  if (decorative || settling) {
    return (
      <span aria-hidden data-slot="desk-thinking" className={frame}>
        {mark}
      </span>
    );
  }

  return (
    <span
      role="status"
      aria-label={t("Working on your answer")}
      data-slot="desk-thinking"
      className={frame}
    >
      {mark}
    </span>
  );
}
