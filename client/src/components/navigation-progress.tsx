import { useEffect, useRef, useState } from "react";
import { useNavigation } from "react-router";

const INITIAL_PROGRESS = 0.15;
const MAX_TRICKLE_PROGRESS = 0.94;
const FIRST_TRICKLE_MS = 50;
const TRICKLE_MIN_INTERVAL_MS = 140;
const TRICKLE_JITTER_MS = 160;
const COMPLETE_TRANSITION = "transform 150ms ease-out, opacity 200ms ease 100ms";
const TRICKLE_TRANSITION = "transform 250ms ease-out";
const RESET_DELAY_MS = 350;

type LoaderPhase = "idle" | "loading" | "done";

function trickleAmount(progress: number) {
  if (progress < 0.25) return 0.1 + Math.random() * 0.06;
  if (progress < 0.55) return 0.05 + Math.random() * 0.04;
  if (progress < 0.8) return 0.02 + Math.random() * 0.02;
  return 0.005 + Math.random() * 0.005;
}

export function NavigationProgress() {
  const navigation = useNavigation();
  const isNavigating = navigation.state !== "idle";

  const [phase, setPhase] = useState<LoaderPhase>("idle");
  const [progress, setProgress] = useState(0);
  const phaseRef = useRef<LoaderPhase>("idle");
  const timersRef = useRef<{ trickle?: number; reset?: number }>({});

  useEffect(() => {
    const timers = timersRef.current;

    const changePhase = (next: LoaderPhase) => {
      phaseRef.current = next;
      setPhase(next);
    };

    if (isNavigating) {
      window.clearTimeout(timers.reset);
      changePhase("loading");
      setProgress(INITIAL_PROGRESS);

      const trickle = () => {
        setProgress((current) =>
          Math.min(current + trickleAmount(current), MAX_TRICKLE_PROGRESS),
        );
        timers.trickle = window.setTimeout(
          trickle,
          TRICKLE_MIN_INTERVAL_MS + Math.random() * TRICKLE_JITTER_MS,
        );
      };

      timers.trickle = window.setTimeout(trickle, FIRST_TRICKLE_MS);

      return () => window.clearTimeout(timers.trickle);
    }

    if (phaseRef.current !== "loading") {
      return;
    }

    changePhase("done");
    setProgress(1);
    timers.reset = window.setTimeout(() => {
      changePhase("idle");
      setProgress(0);
    }, RESET_DELAY_MS);

    return () => window.clearTimeout(timers.reset);
  }, [isNavigating]);

  if (phase === "idle") {
    return null;
  }

  const isAtStart = phase === "loading" && progress === INITIAL_PROGRESS;

  return (
    <div aria-hidden="true" className="pointer-events-none fixed inset-x-0 top-0 z-[100] h-0.5">
      <div
        className="relative h-full bg-brand will-change-transform"
        style={{
          transform: `translateX(${(progress - 1) * 100}%)`,
          transition: isAtStart ? "none" : phase === "done" ? COMPLETE_TRANSITION : TRICKLE_TRANSITION,
          opacity: phase === "done" ? 0 : 1,
        }}
      >
        <div
          className="absolute -top-px right-0 h-1 w-24 rotate-2"
          style={{ boxShadow: "0 0 10px var(--brand), 0 0 5px var(--brand)" }}
        />
      </div>
    </div>
  );
}
