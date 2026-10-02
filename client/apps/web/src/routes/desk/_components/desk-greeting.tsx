import type { SkyPhase } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";

export type DeskGreetingProps = {
  /** The light outside in the person's timezone, which colours the wash and the mark. */
  sky: SkyPhase;
  /** The first line of the headline: the time of day, and the person by name. */
  greeting: string;
  /** The day, as the person reads it. */
  dateline: string;
  /** The same day, machine-readable, for the `<time>` that carries it. */
  isoDate: string;
  /** The second line of the headline: what the day looks like, or the question the page asks. */
  headline: ReactNode;
  /** Staggers each line in behind the one above it, in reading order. */
  entrance: (step: number) => CSSProperties;
  className?: string;
};

/**
 * The top of the Desk's front page: a greeting under the day's own light.
 *
 * Behind the first lines sits a wash of the part of the day, a pool of dawn
 * amber, daylight blue, dusk rose or night indigo that fades out long before
 * it reaches anything else, and the dateline leads with a small mark of the
 * same sky. That is all the colour here. The headline is two lines of one
 * heading: the greeting in ink, and under it, quieter, what the day looks
 * like — or, when nothing is waiting, the question the page is for.
 *
 * It stands centred, because the page is a room to be entered rather than
 * a column to be read, and every line rises in once, a beat apart, and then
 * holds still.
 */
export function DeskGreeting({
  sky,
  greeting,
  dateline,
  isoDate,
  headline,
  entrance,
  className,
}: DeskGreetingProps) {
  return (
    <header
      data-sky={sky}
      className={cn("relative isolate flex flex-col items-center gap-4 text-center", className)}
    >
      <div
        aria-hidden
        data-slot="desk-daylight"
        data-sky={sky}
        className="ui-daylight animate-rise pointer-events-none absolute -top-32 left-1/2 -z-10 h-72 w-200 max-w-none -translate-x-1/2"
        style={entrance(0)}
      />
      <p
        className="text-foreground-subtle animate-rise flex items-center gap-2 text-sm"
        style={entrance(0)}
      >
        <SkyMark sky={sky} className="size-4" />
        <time dateTime={isoDate}>{dateline}</time>
      </p>
      <h1 className="max-w-3xl text-2xl font-semibold text-balance sm:text-3xl">
        <span className="animate-rise block" style={entrance(1)}>
          {greeting}
        </span>
        <span
          data-slot="desk-headline"
          className="text-foreground-muted animate-rise mt-1 block text-xl font-medium sm:text-2xl"
          style={entrance(2)}
        >
          {headline}
        </span>
      </h1>
    </header>
  );
}

/**
 * The sky, as a mark the size of a letter: the sun on the horizon at dawn,
 * high and whole by day, going down at dusk, and a moon with a star at night.
 * Solid shapes in the day's two colours, so it reads at 16px.
 */
export function SkyMark({ sky, className }: { sky: SkyPhase; className?: string }) {
  return (
    <svg
      viewBox="0 0 16 16"
      aria-hidden
      focusable="false"
      data-slot="sky-mark"
      data-sky={sky}
      className={cn("shrink-0", className)}
    >
      <SkyShapes sky={sky} />
    </svg>
  );
}

/** Eight rays around the day's sun, as short bars turned about its centre. */
const DAY_RAYS = [0, 45, 90, 135, 180, 225, 270, 315] as const;
/** The three rays a sun on the horizon still shows above it. */
const DAWN_RAYS = [-60, 0, 60] as const;

function SkyShapes({ sky }: { sky: SkyPhase }) {
  switch (sky) {
    case "dawn":
      return (
        <>
          <g className="fill-daylight-sun">
            <path d="M3.9 11.2a4.1 4.1 0 0 1 8.2 0Z" />
            {DAWN_RAYS.map((angle) => (
              <rect
                key={angle}
                x="7.35"
                y="1.2"
                width="1.3"
                height="2.4"
                rx="0.65"
                transform={`rotate(${angle} 8 11.2)`}
              />
            ))}
          </g>
          <rect x="1.5" y="12.4" width="13" height="1.4" rx="0.7" className="fill-accent-rose" />
        </>
      );
    case "day":
      return (
        <g className="fill-daylight-sun">
          <circle cx="8" cy="8" r="3.3" />
          {DAY_RAYS.map((angle) => (
            <rect
              key={angle}
              x="7.35"
              y="0.6"
              width="1.3"
              height="2.3"
              rx="0.65"
              transform={`rotate(${angle} 8 8)`}
            />
          ))}
        </g>
      );
    case "dusk":
      return (
        <>
          <path d="M3.4 11.6a4.6 4.6 0 0 1 9.2 0Z" className="fill-accent-rose" />
          <rect x="1.5" y="12.4" width="13" height="1.4" rx="0.7" className="fill-accent-violet" />
        </>
      );
    default:
      return (
        <>
          <path
            d="M10.6 12.9A5.4 5.4 0 0 1 6.2 2.6a4.6 4.6 0 1 0 7.2 7.3a5.4 5.4 0 0 1-2.8 3Z"
            className="fill-daylight-moon"
          />
          <path
            d="M12.4 1.4l.55 1.35l1.35.55l-1.35.55l-.55 1.35l-.55-1.35l-1.35-.55l1.35-.55Z"
            className="fill-daylight-sun"
          />
        </>
      );
  }
}
