import "./demo-banner.css";
import type { DemoBannerMessage } from "../../lib/cloud-trial";
import { PLAN_USAGE_PATH } from "../../lib/free-demo";
import {
  ArrowRightIcon,
  CalendarIcon,
  ClockIcon,
  MarkerPin01Icon,
  PackageIcon,
  RouteIcon,
  Truck01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import { Link } from "react-router";

export const DEMO_BANNER_HOLD_MS = 5200;
const SWAP_DELAY_MS = 260;
const LEAVE_MS = 500;
const PULSE_MS = 1000;
const DIGITS = ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"] as const;
const RAIN_ICONS: IconComponent[] = [
  Truck01Icon,
  PackageIcon,
  ClockIcon,
  MarkerPin01Icon,
  RouteIcon,
  CalendarIcon,
];
const RAIN_COLORS = [
  "var(--tdb-c1)",
  "var(--tdb-c2)",
  "var(--tdb-c3)",
  "var(--tdb-c4)",
  "var(--tdb-c5)",
];
const RAIN_COUNT = 22;

function seeded(index: number, salt: number): number {
  const x = Math.sin(index * 12.9898 + salt * 78.233) * 43758.5453;
  return x - Math.floor(x);
}

const RAIN = Array.from({ length: RAIN_COUNT }, (_, index) => {
  const duration = 22 + seeded(index, 1) * 18;
  return {
    Icon: RAIN_ICONS[index % RAIN_ICONS.length],
    style: {
      "--tdb-s": `${10 + seeded(index, 2) * 5}px`,
      "--tdb-y": `${3 + seeded(index, 3) * 18}px`,
      "--tdb-t": `${duration}s`,
      "--tdb-dl": `${-seeded(index, 4) * duration}s`,
      "--tdb-b": `${1.6 + seeded(index, 5) * 1.4}s`,
      "--tdb-c": RAIN_COLORS[index % RAIN_COLORS.length],
      "--tdb-o": `${0.35 + seeded(index, 6) * 0.25}`,
    } as CSSProperties,
  };
});

function unitWords(message: DemoBannerMessage, t: TranslateFn): string[] {
  const phrase =
    message.kind === "days"
      ? t("{0, plural, one {day left.} other {days left.}}", message.value)
      : t("{0, plural, one {shipment left.} other {shipments left.}}", message.value);
  return phrase.split(" ");
}

function Odometer({ value, rolled }: { value: number; rolled: boolean }) {
  const digits = String(value).split("");
  return (
    <span className="tdb-odo" style={{ "--tdb-i": 0 } as CSSProperties}>
      <span className="sr-only">{value}</span>
      {digits.map((digit, index) => (
        <span
          key={index}
          className="tdb-digit"
          aria-hidden="true"
          data-digit={digit}
          style={
            {
              "--tdb-d": `${(digits.length - 1 - index) * 90 + 150}ms`,
              transform: rolled ? `translateY(-${Number(digit) * 1.5}em)` : undefined,
            } as CSSProperties
          }
        >
          {DIGITS.map((face) => (
            <span key={face}>{face}</span>
          ))}
        </span>
      ))}
    </span>
  );
}

type Phase = "in" | "out" | "gone";

type View = {
  current: number;
  leaving: number;
  entered: boolean;
  rolled: boolean;
};

const INITIAL_VIEW: View = { current: -1, leaving: -1, entered: false, rolled: false };

function phaseOf(index: number, view: View): Phase {
  if (index === view.current) {
    return view.entered ? "in" : "gone";
  }
  if (index === view.leaving) {
    return "out";
  }
  return "gone";
}

function waitForFonts(): Promise<unknown> {
  if (typeof document === "undefined" || !document.fonts?.ready) {
    return Promise.resolve();
  }
  return document.fonts.ready;
}

/**
 * The free demo's countdown, pinned above the app header: days left, then
 * shipments left, turning every few seconds. Pointing at it (or focusing inside
 * it) holds the current message. It is never dismissible.
 */
export function DemoBanner({
  messages,
  showPlanLink,
}: {
  messages: DemoBannerMessage[];
  showPlanLink: boolean;
}) {
  const t = useT();
  const [view, setView] = useState<View>(INITIAL_VIEW);
  const [paused, setPaused] = useState(false);
  const [pulsing, setPulsing] = useState(false);
  const [width, setWidth] = useState<number | undefined>(undefined);
  const phraseRefs = useRef<(HTMLSpanElement | null)[]>([]);
  const currentRef = useRef(-1);
  const timers = useRef<number[]>([]);
  const frames = useRef<number[]>([]);

  const later = useCallback((fn: () => void, ms: number) => {
    timers.current.push(window.setTimeout(fn, ms));
  }, []);

  const nextFrame = useCallback((fn: () => void) => {
    frames.current.push(window.requestAnimationFrame(fn));
  }, []);

  const show = useCallback(
    (index: number) => {
      const previous = currentRef.current;
      const hadPrevious = previous !== -1 && previous !== index;
      currentRef.current = index;
      setView({
        current: index,
        leaving: hadPrevious ? previous : -1,
        entered: false,
        rolled: false,
      });

      const enter = () => {
        setView((v) => (v.current === index ? { ...v, entered: true } : v));
        nextFrame(() =>
          setView((v) => (v.current === index && v.entered ? { ...v, rolled: true } : v)),
        );
      };
      if (hadPrevious) {
        later(enter, SWAP_DELAY_MS);
        later(() => setView((v) => (v.leaving === previous ? { ...v, leaving: -1 } : v)), LEAVE_MS);
      } else {
        later(enter, 0);
      }

      setPulsing(false);
      nextFrame(() => {
        setPulsing(true);
        later(() => setPulsing(false), PULSE_MS);
      });
    },
    [later, nextFrame],
  );

  useEffect(() => {
    let cancelled = false;
    void waitForFonts().then(() => {
      if (!cancelled) {
        show(0);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [show]);

  useEffect(() => {
    const pending = timers.current;
    const pendingFrames = frames.current;
    return () => {
      pending.forEach((id) => window.clearTimeout(id));
      pendingFrames.forEach((id) => window.cancelAnimationFrame(id));
    };
  }, []);

  const { current: currentIndex, entered } = view;

  useEffect(() => {
    if (paused || currentIndex === -1 || messages.length < 2) {
      return undefined;
    }
    const id = window.setTimeout(
      () => show((currentIndex + 1) % messages.length),
      DEMO_BANNER_HOLD_MS,
    );
    return () => window.clearTimeout(id);
  }, [paused, currentIndex, messages.length, show]);

  useLayoutEffect(() => {
    if (!entered) {
      return;
    }
    const node = phraseRefs.current[currentIndex];
    if (node) {
      setWidth(node.offsetWidth);
    }
  }, [currentIndex, entered, messages]);

  const showing = view.current === -1 ? 0 : view.current;
  const showingKind = messages[showing]?.kind ?? "days";

  return (
    <div
      role="status"
      aria-live="polite"
      data-testid="demo-banner"
      className={cn("tdb", pulsing && "tdb-pulse")}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setPaused(false);
        }
      }}
    >
      <div className="tdb-rain" aria-hidden="true">
        {RAIN.map(({ Icon, style }, index) => (
          <span key={index} className="tdb-fl" style={style}>
            <Icon strokeWidth={2} />
          </span>
        ))}
      </div>
      <span className="tdb-chip">
        <span className="tdb-tile" aria-hidden="true">
          <i className="tdb-ring" />
          <ClockIcon strokeWidth={2.4} data-out={showingKind !== "days"} />
          <PackageIcon strokeWidth={2.4} data-out={showingKind !== "shipments"} />
        </span>
        {t("Free demo")}
      </span>
      <span className="tdb-msg">
        <span className="tdb-roll" style={{ width }}>
          {messages.map((message, index) => {
            const phase = phaseOf(index, view);
            const words = unitWords(message, t);
            const rolled =
              (index === view.current && view.rolled) ||
              (index === view.leaving && phase === "out");
            return (
              <span
                key={message.kind}
                ref={(node) => {
                  phraseRefs.current[index] = node;
                }}
                className="tdb-phrase"
                data-phase={phase}
                data-kind={message.kind}
                aria-hidden={phase === "gone" ? true : undefined}
              >
                <Odometer value={message.value} rolled={rolled} />
                {words.map((word, wordIndex) => (
                  <span
                    key={wordIndex}
                    className={wordIndex === 0 ? "tdb-unit" : undefined}
                    style={{ "--tdb-i": wordIndex + 1 } as CSSProperties}
                  >
                    {word}
                  </span>
                ))}
              </span>
            );
          })}
        </span>
        <span>{t("After that the workspace becomes read-only.")}</span>
      </span>
      <span className="flex-1" />
      {showPlanLink ? (
        <Link to={PLAN_USAGE_PATH} className="tdb-cta">
          {t("Plan & usage")}
          <ArrowRightIcon size={12} strokeWidth={2} aria-hidden="true" />
        </Link>
      ) : null}
    </div>
  );
}
