import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { useEffect, useId, useState, type CSSProperties } from "react";
import type { DeskPose } from "./desk-pose";

/** How long the closing pose holds before the mark is gone. */
export const DESK_SETTLE_MS = 420;

/** The lines the screen scrolls through while a tool runs: two screens' worth, so the loop is seamless. */
const SCROLL_LINES = [3.4, 2.2, 2.9, 3.4, 2.2, 2.9] as const;

/** The lines written onto the screen while the answer arrives. */
const WRITE_LINES = [
  { width: 3.6, className: "animate-desk-write-1" },
  { width: 2.6, className: "animate-desk-write-2" },
  { width: 3.1, className: "animate-desk-write-3" },
] as const;

/** The glow is the screen's own colour, read from the mark like every other part. */
const GLOW_STOP: CSSProperties = { stopColor: "var(--desk-accent)" };

const THINK_DOTS = [
  { cx: 7.3, className: "animate-desk-think-1" },
  { cx: 8.5, className: "animate-desk-think-2" },
  { cx: 9.7, className: "animate-desk-think-3" },
] as const;

export type DeskMarkProps = {
  pose: DeskPose;
  /**
   * Whether the scene may move. Off, every pose is one still frame of the
   * same moment, which is what reduced motion wants.
   */
  animate?: boolean;
  className?: string;
};

/**
 * A small desk scene at the height of a line of text: a desk with a lamp on
 * it, a monitor, and a person in a chair working at it.
 *
 * It is drawn in solid shapes on a 20 by 16 grid, one unit to a pixel at its
 * resting size, so every edge lands on the pixel grid and nothing turns to
 * mush the way a hairline does at 16px. The agent's accent is the person and
 * the screen, and a tint in the chair; the furniture is warm wood and ink,
 * and the lamp is the one warm light. Every colour is a token.
 *
 * The screen tells the story, frame by frame, the way a sprite does: dots
 * while the model thinks, lines scrolling past while a tool runs, lines being
 * written while the answer arrives. The person sits down once as the mark
 * arrives, types while a tool runs, and pushes back from the desk when the
 * turn is over.
 *
 * The drawing never announces anything; the component around it does.
 */
export function DeskMark({ pose, animate = true, className }: DeskMarkProps) {
  const settling = pose === "settle";
  const typing = pose === "busy" || pose === "write";
  const id = useId().replace(/[^\w-]/g, "");
  const clipId = `desk-screen-${id}`;
  const glowId = `desk-glow-${id}`;

  return (
    <svg
      viewBox="0 0 20 16"
      aria-hidden
      focusable="false"
      data-slot="desk-mark"
      data-pose={pose}
      data-motion={animate ? "moving" : "still"}
      className={cn(
        "ui-desk-mark h-4 w-5 shrink-0 overflow-visible",
        animate && settling && "animate-desk-fade",
        className,
      )}
    >
      <defs>
        <clipPath id={clipId}>
          <rect x="5.9" y="2.9" width="5.2" height="3.7" rx="0.45" />
        </clipPath>
        <radialGradient id={glowId}>
          <stop offset="0.35" style={GLOW_STOP} />
          <stop offset="1" stopOpacity="0" style={GLOW_STOP} />
        </radialGradient>
      </defs>

      <g data-part="lamp">
        <path d="M0.9 5.6L4.3 5.6L5.3 9L-0.1 9Z" className="fill-desk-lamp-light" />
        <rect x="2" y="5" width="0.8" height="3.6" className="fill-desk-frame" />
        <rect x="1" y="8.2" width="2.8" height="0.9" rx="0.45" className="fill-desk-frame" />
        <path d="M0.8 5.8C0.8 3.3 4.4 3.3 4.4 5.8Z" className="fill-desk-lamp" />
      </g>

      <g data-part="desk">
        <rect x="1.2" y="10" width="3.6" height="5.6" rx="0.4" className="fill-desk-wood-shade" />
        <rect x="2.2" y="12.2" width="1.6" height="0.7" rx="0.35" className="fill-desk-wood" />
        <rect x="0.4" y="9" width="13.6" height="1.4" rx="0.5" className="fill-desk-wood" />
      </g>

      <g data-part="monitor">
        <ellipse
          data-part="glow"
          cx="8.5"
          cy="4.75"
          rx="6.5"
          ry="5.5"
          fill={`url(#${glowId})`}
          className={cn(
            "transition-opacity duration-300",
            pose === "write" ? "opacity-(--desk-glow)" : "opacity-0",
            animate && pose === "write" && "animate-desk-glow",
          )}
        />
        <rect x="8" y="7.2" width="1" height="1.6" className="fill-desk-frame" />
        <rect x="6.6" y="8.3" width="3.8" height="0.7" rx="0.35" className="fill-desk-frame" />
        <rect x="5" y="2" width="7" height="5.5" rx="1" className="fill-desk-frame" />
        <rect
          data-part="screen"
          x="5.9"
          y="2.9"
          width="5.2"
          height="3.7"
          rx="0.45"
          className={cn(
            "fill-desk-accent transition-opacity duration-300",
            settling && "opacity-40",
          )}
        />
        <rect x="10.9" y="8.4" width="2.4" height="0.6" rx="0.3" className="fill-desk-frame" />
        <g clipPath={`url(#${clipId})`} className="fill-desk-screen-ink">
          <ScreenContent pose={pose} animate={animate} />
        </g>
      </g>

      <g
        data-part="seat"
        className={cn(
          animate && !settling && "animate-desk-sit",
          animate && settling && "animate-desk-settle",
        )}
      >
        <g data-part="chair" className="fill-desk-chair">
          <rect x="17.3" y="4.9" width="1.6" height="6.3" rx="0.8" />
          <rect x="13.9" y="10.1" width="5" height="1.1" rx="0.55" />
          <rect x="15.9" y="11.2" width="1" height="2.5" />
          <rect x="14.2" y="13.5" width="4.4" height="0.8" rx="0.4" />
          <circle cx="14.8" cy="15.1" r="0.7" />
          <circle cx="18" cy="15.1" r="0.7" />
        </g>
        <g
          data-part="figure"
          className={cn("fill-desk-figure", animate && !settling && "animate-desk-sit-figure")}
        >
          <circle cx="15.5" cy="4.3" r="1.7" />
          <rect x="14.3" y="6.4" width="3" height="4.2" rx="1.3" />
          <rect x="12.3" y="9.6" width="3.6" height="1.3" rx="0.65" />
          <rect x="12.3" y="10.2" width="1.2" height="4.6" rx="0.6" />
          <rect x="11.2" y="14.3" width="2.3" height="1" rx="0.5" />
          <rect
            data-part="arms"
            x="11.9"
            y="7.8"
            width="3.8"
            height="1"
            rx="0.5"
            className={cn(animate && typing && "animate-desk-type")}
          />
        </g>
      </g>
    </svg>
  );
}

/** What is on the screen: the one part of the scene that changes with every phase. */
function ScreenContent({ pose, animate }: { pose: DeskPose; animate: boolean }) {
  switch (pose) {
    case "arrive":
      return (
        <g data-part="thinking">
          {THINK_DOTS.map((dot) => (
            <circle
              key={dot.cx}
              cx={dot.cx}
              cy="4.75"
              r="0.5"
              className={cn(animate && dot.className)}
            />
          ))}
        </g>
      );
    case "busy":
      return (
        <g data-part="scrolling" className={cn(animate && "animate-desk-scroll")}>
          {SCROLL_LINES.map((width, index) => (
            <rect
              key={`${index}-${width}`}
              x="6.6"
              y={3.45 + index}
              width={width}
              height="0.5"
              rx="0.25"
            />
          ))}
        </g>
      );
    case "write":
      return (
        <g data-part="writing">
          {WRITE_LINES.map((line, index) => (
            <rect
              key={line.className}
              x="6.6"
              y={3.45 + index}
              width={line.width}
              height="0.5"
              rx="0.25"
              className={cn("origin-left [transform-box:fill-box]", animate && line.className)}
            />
          ))}
        </g>
      );
    default:
      return null;
  }
}

export type ThinkingPresence = "working" | "settling" | "gone";

/**
 * Whether the working mark is on screen, and in which state.
 *
 * It is there while the work is, and the moment the work stops it takes the
 * closing pose for one beat and is gone. Under reduced motion there is no
 * closing pose to show, so it goes at once.
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

/**
 * The live indicator on the working line: the desk scene at work, for as
 * long as the agent is.
 *
 * It lives in one place only, beside the words of the working line under a
 * reply in progress. Everywhere else a small thing still going is the
 * breathing dot; this is the one that gets to be a scene, because it sits
 * beside the words that say what the scene shows.
 *
 * Announced, it is a status with a fixed name and no content that changes,
 * so a screen reader hears that the work is under way once and is never told
 * about the drawing moving. Settling, it is already over, so it says nothing.
 */
export function DeskThinking({ pose, className }: { pose: DeskPose; className?: string }) {
  const t = useT();
  const reduceMotion = useReducedMotion() ?? false;
  const frame = cn("inline-flex shrink-0 items-center justify-center", className);
  const mark = <DeskMark pose={pose} animate={!reduceMotion} />;

  if (pose === "settle") {
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
