import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { useEffect, useId, useState } from "react";
import { isClosingPose, type DeskPose } from "./desk-pose";

/** How long the closing beat holds before the mark is gone. */
export const DESK_SETTLE_MS = 900;

/** Where the lamp's head turns: its hinge, and the angle it rests at. */
const HINGE = "translate(16.4 9.6) rotate(-22)";

const CONE = "M-3.4 3.4 L3.4 3.4 L10.5 19 L-10.5 19 Z";
const SHADE = "M-2.1 -1.1 L2.1 -1.1 L3.9 3.3 L-3.9 3.3 Z";

const DASHES = [
  { x: 18.2, className: "animate-desk-dash-1" },
  { x: 21.6, className: "animate-desk-dash-2" },
  { x: 25, className: "animate-desk-dash-3" },
] as const;

const WAVES = [
  { radius: 3, className: "animate-desk-wave-1" },
  { radius: 5.2, className: "animate-desk-wave-2" },
  { radius: 7.4, className: "animate-desk-wave-3" },
] as const;

/** An arc of a signal wave, centred where the head points when it looks out. */
function wavePath(radius: number): string {
  const spread = (38 * Math.PI) / 180;
  const dx = radius * Math.cos(spread);
  const dy = radius * Math.sin(spread);

  return `M${22.5 + dx} ${11.5 - dy} A${radius} ${radius} 0 0 1 ${22.5 + dx} ${11.5 + dy}`;
}

type LampPart = "mark" | "lamp" | "head" | "beam" | "cone" | "glow" | "pool" | "bulb" | "fault";

/** The things that appear on the desk for some poses and are not drawn otherwise. */
type LampExtra = "dashes" | "tick" | "card" | "helper" | "waves";

/**
 * How one part looks in a pose: `always` in both modes, `still` as the one
 * frame reduced motion draws, `move` while it may animate. A still frame is
 * the moment each motion is about, so the two modes never disagree.
 */
type PartStyle = { always?: string; still?: string; move?: string };

type PoseStyle = { parts: Partial<Record<LampPart, PartStyle>>; extra?: LampExtra };

const BASE: Record<LampPart, string> = {
  mark: "ui-desk-mark h-4 w-5 shrink-0 overflow-visible",
  lamp: "",
  head: "origin-top-left",
  beam: "",
  cone: "fill-desk-accent opacity-20 origin-top [transform-box:fill-box]",
  glow: "",
  pool: "fill-desk-light opacity-90 origin-center [transform-box:fill-box]",
  bulb: "fill-desk-light",
  fault: "fill-desk-failed opacity-0",
};

const DARK: PartStyle = { still: "opacity-0" };

/** Every pose, part by part. A part a pose does not name is drawn at rest. */
const POSES: Record<DeskPose, PoseStyle> = {
  start: {
    parts: {
      beam: { move: "animate-desk-light-on" },
      glow: { move: "animate-desk-light-on" },
      bulb: { move: "animate-desk-light-on" },
    },
  },
  think: {
    parts: {
      cone: { move: "animate-desk-breathe-cone" },
      pool: { move: "animate-desk-breathe-pool" },
    },
  },
  write: {
    parts: { cone: { always: "opacity-25" }, glow: { always: "opacity-0" } },
    extra: "dashes",
  },
  retry: {
    parts: {
      head: { move: "animate-desk-shake" },
      beam: { move: "animate-desk-light-retry" },
      glow: { move: "animate-desk-light-retry" },
      bulb: { move: "animate-desk-light-retry" },
    },
  },
  lookup: {
    parts: {
      head: { move: "animate-desk-scan" },
      pool: { move: "animate-desk-scan-pool" },
    },
  },
  discover: {
    parts: {
      head: { move: "animate-desk-hunt" },
      cone: { move: "animate-desk-widen" },
      pool: { move: "animate-desk-hunt-pool" },
    },
  },
  navigate: {
    parts: {
      head: { move: "animate-desk-travel" },
      pool: { move: "animate-desk-travel-pool" },
    },
  },
  change: {
    parts: {
      head: { move: "animate-desk-stamp" },
      pool: { move: "animate-desk-stamp-pool" },
    },
    extra: "tick",
  },
  present: {
    parts: {
      cone: { always: "opacity-25" },
      pool: { move: "animate-desk-present-pool" },
    },
    extra: "card",
  },
  ask: {
    parts: {
      head: { still: "-rotate-58", move: "animate-desk-turn" },
      cone: { always: "scale-y-60" },
      glow: DARK,
      pool: { move: "animate-desk-away-pool" },
      bulb: { move: "animate-desk-blink" },
    },
  },
  delegate: {
    parts: {
      beam: { still: "opacity-0", move: "animate-desk-handoff-light" },
      pool: { move: "animate-desk-handoff-pool" },
    },
    extra: "helper",
  },
  web: {
    parts: {
      head: { still: "-rotate-96", move: "animate-desk-lookout" },
      cone: { always: "scale-y-60" },
      glow: DARK,
      pool: { move: "animate-desk-away-pool" },
    },
    extra: "waves",
  },
  done: {
    parts: {
      mark: { move: "animate-desk-fade" },
      head: { still: "rotate-5", move: "animate-desk-nod" },
      beam: { still: "opacity-0", move: "animate-desk-light-off" },
      glow: { still: "opacity-0", move: "animate-desk-light-off" },
      bulb: { still: "opacity-0", move: "animate-desk-light-off" },
    },
  },
  await: {
    parts: {
      mark: { move: "animate-desk-fade" },
      lamp: { always: "[--desk-accent:var(--desk-await)] [--desk-light:var(--desk-await)]" },
    },
  },
  failed: {
    parts: {
      mark: { move: "animate-desk-fade" },
      head: { still: "rotate-14", move: "animate-desk-droop" },
      beam: { still: "opacity-0", move: "animate-desk-light-fail" },
      glow: { still: "opacity-0", move: "animate-desk-light-fail" },
      bulb: { still: "opacity-0", move: "animate-desk-light-fail" },
      fault: { still: "opacity-100", move: "animate-desk-fault" },
    },
  },
};

function partClass(pose: DeskPose, part: LampPart, animate: boolean): string {
  const style = POSES[pose].parts[part];

  return cn(BASE[part], style?.always, animate ? style?.move : style?.still);
}

export type DeskMarkProps = {
  pose: DeskPose;
  /**
   * Whether the lamp may move. Off, every pose is one still frame of the
   * same moment, which is what reduced motion wants.
   */
  animate?: boolean;
  className?: string;
};

/**
 * A desk lamp at the height of a line of text: an ink lamp on an ink desk,
 * throwing the agent's light.
 *
 * It is drawn in solid shapes on a 28 by 22.4 grid, so at its resting size
 * a unit is under a pixel and nothing is thinner than a stroke that holds
 * at 16px. The lamp never moves; its head turns about the hinge, its light
 * changes, and a few small things appear on the desk under it. Which of
 * them is the pose, one for each moment of a turn and one for each kind of
 * tool (see `DeskPose`).
 *
 * The drawing never announces anything; the component around it does.
 */
export function DeskMark({ pose, animate = true, className }: DeskMarkProps) {
  const id = useId().replace(/[^\w-]/g, "");
  const clipId = `desk-light-${id}`;
  const extra = POSES[pose].extra;
  const part = (name: LampPart) => partClass(pose, name, animate);

  return (
    <svg
      viewBox="2 6 28 22.4"
      aria-hidden
      focusable="false"
      data-slot="desk-mark"
      data-pose={pose}
      data-motion={animate ? "moving" : "still"}
      className={cn(part("mark"), className)}
    >
      <defs>
        <clipPath id={clipId}>
          <rect x="2" y="-8" width="28" height="34" />
        </clipPath>
      </defs>

      <g data-part="lamp" className={part("lamp")}>
        <rect
          data-part="desk"
          x="3"
          y="26"
          width="26"
          height="2.4"
          rx="1.2"
          className="fill-desk-lamp"
        />
        <g data-part="glow" className={part("glow")}>
          <ellipse
            data-part="pool"
            cx="22.4"
            cy="26.3"
            rx="5.2"
            ry="0.9"
            className={part("pool")}
          />
        </g>

        {extra === "dashes" && <Dashes animate={animate} />}
        {extra === "card" && <Card animate={animate} />}
        {extra === "tick" && <Tick animate={animate} />}
        {extra === "helper" && <Helper animate={animate} clipId={clipId} />}

        <g data-part="stand" className="fill-desk-lamp">
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
        </g>

        {extra === "waves" && <Waves animate={animate} />}

        <g clipPath={`url(#${clipId})`}>
          <g transform={HINGE}>
            <g data-part="head" className={part("head")}>
              <g data-part="beam" className={part("beam")}>
                <path data-part="cone" d={CONE} className={part("cone")} />
              </g>
              <path
                d={SHADE}
                strokeWidth="1"
                strokeLinejoin="round"
                className="fill-desk-lamp stroke-desk-lamp"
              />
              <circle cx="0" cy="-1.3" r="1.3" className="fill-desk-lamp" />
              <ellipse cx="0" cy="3.7" rx="2.3" ry="0.95" className="fill-desk-bulb-off" />
              <ellipse
                data-part="bulb"
                cx="0"
                cy="3.7"
                rx="2.3"
                ry="0.95"
                className={part("bulb")}
              />
              {pose === "failed" && (
                <ellipse
                  data-part="fault"
                  cx="0"
                  cy="3.7"
                  rx="2.3"
                  ry="0.95"
                  className={part("fault")}
                />
              )}
            </g>
          </g>
        </g>
      </g>
    </svg>
  );
}

type ExtraProps = { animate: boolean };

/** The answer arriving: lines lit on the desk one after another. */
function Dashes({ animate }: ExtraProps) {
  return (
    <g data-part="dashes" className="fill-desk-light">
      {DASHES.map((dash) => (
        <rect
          key={dash.x}
          x={dash.x}
          y="24.4"
          width="2.6"
          height="1.2"
          rx="0.6"
          className={animate ? cn("opacity-0", dash.className) : undefined}
        />
      ))}
    </g>
  );
}

/** A change made: a tick in the pool of light. */
function Tick({ animate }: ExtraProps) {
  return (
    <path
      data-part="tick"
      d="M19.9 22.3 L21.8 24.1 L25.2 20.4"
      fill="none"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn(
        "stroke-desk-light origin-center [transform-box:fill-box]",
        animate && "animate-desk-tick",
      )}
    />
  );
}

/** Something presented: a card standing in the light. */
function Card({ animate }: ExtraProps) {
  return (
    <g
      data-part="card"
      className={cn("origin-bottom [transform-box:fill-box]", animate && "animate-desk-card")}
    >
      <rect
        x="18.4"
        y="17.2"
        width="8"
        height="8.8"
        rx="1.2"
        className="fill-desk-accent opacity-20"
      />
      <rect x="19.8" y="19.2" width="5.2" height="1" rx="0.5" className="fill-desk-light" />
      <rect x="19.8" y="21.4" width="3.6" height="1" rx="0.5" className="fill-desk-light" />
    </g>
  );
}

/**
 * A task handed on: a second lamp, half the size and facing the first,
 * whose light takes over the same pool.
 */
function Helper({ animate, clipId }: ExtraProps & { clipId: string }) {
  const light = animate ? "animate-desk-helper-light" : undefined;

  return (
    <g
      data-part="helper"
      clipPath={`url(#${clipId})`}
      className={cn(animate && "animate-desk-helper")}
    >
      <g transform="translate(33.2 13) scale(-0.5 0.5)" className="fill-desk-lamp">
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
          <g data-part="helper-beam" className={light}>
            <path d={CONE} className="fill-desk-accent opacity-20" />
          </g>
          <path d={SHADE} strokeWidth="1" strokeLinejoin="round" className="stroke-desk-lamp" />
          <circle cx="0" cy="-1.3" r="1.3" />
          <ellipse cx="0" cy="3.7" rx="2.3" ry="0.95" className="fill-desk-bulb-off" />
          <ellipse
            data-part="helper-bulb"
            cx="0"
            cy="3.7"
            rx="2.3"
            ry="0.95"
            className={cn("fill-desk-light", light)}
          />
        </g>
      </g>
    </g>
  );
}

/** The web: signal waves leaving the head as it looks out past the desk. */
function Waves({ animate }: ExtraProps) {
  return (
    <g
      data-part="waves"
      fill="none"
      strokeWidth="1.2"
      strokeLinecap="round"
      className="stroke-desk-light"
    >
      {WAVES.map((wave) => (
        <path
          key={wave.radius}
          d={wavePath(wave.radius)}
          className={
            animate ? cn("[transform-origin:22.5px_11.5px] opacity-0", wave.className) : undefined
          }
        />
      ))}
    </g>
  );
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
 * The live indicator on the working line: the desk lamp at work, for as
 * long as the agent is.
 *
 * It lives in one place only, beside the words of the working line under a
 * reply in progress. Everywhere else a small thing still going is the
 * breathing dot; this is the one that gets to be a drawing, because it sits
 * beside the words that say what the drawing shows.
 *
 * Announced, it is a status with a fixed name and no content that changes,
 * so a screen reader hears that the work is under way once and is never told
 * about the drawing moving. Closing, it is already over, so it says nothing.
 */
export function DeskThinking({ pose, className }: { pose: DeskPose; className?: string }) {
  const t = useT();
  const reduceMotion = useReducedMotion() ?? false;
  const frame = cn("inline-flex shrink-0 items-center justify-center", className);
  const mark = <DeskMark pose={pose} animate={!reduceMotion} />;

  if (isClosingPose(pose)) {
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
