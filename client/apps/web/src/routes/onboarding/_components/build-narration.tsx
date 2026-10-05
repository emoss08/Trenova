import type { NovaSegment } from "@/lib/onboarding-copy";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Fragment, useEffect, useRef, useState } from "react";

export type BuildOutcome = "pending" | "success" | "error";

const MIN_LINE_MS = 560;
const LINE_JITTER_MS = 520;
const READY_DELAY_MS = 300;

function seconds(ms: number): string {
  return `${(ms / 1000).toFixed(1)}s`;
}

type Progress = {
  current: number;
  elapsed: string[];
  lineStartedAt: number;
  lastDueAt: number | null;
};

/**
 * The narrated build. Lines advance on their own cadence while the real request runs;
 * the last line keeps spinning until the server answers, so the story never finishes
 * before the workspace exists. A refusal stops the line it lands on.
 */
export function BuildNarration({
  lines,
  outcome,
  settledAt,
  onReady,
  onFailed,
}: {
  lines: readonly (readonly NovaSegment[])[];
  outcome: BuildOutcome;
  settledAt: number | null;
  onReady: () => void;
  onFailed: () => void;
}) {
  const last = lines.length - 1;
  const [progress, setProgress] = useState<Progress>({
    current: 0,
    elapsed: [],
    lineStartedAt: 0,
    lastDueAt: null,
  });
  const callbacks = useRef({ onReady, onFailed });
  const failed = outcome === "error";
  const { current, elapsed, lineStartedAt, lastDueAt } = progress;
  const finished =
    current === last && lastDueAt !== null && outcome === "success" && settledAt !== null;

  useEffect(() => {
    callbacks.current = { onReady, onFailed };
  }, [onReady, onFailed]);

  useEffect(() => {
    if (failed || (current === last && lastDueAt !== null)) {
      return undefined;
    }
    const delay = MIN_LINE_MS + Math.random() * LINE_JITTER_MS;
    const id = window.setTimeout(() => {
      const now = performance.now();
      setProgress((previous) =>
        previous.current < last
          ? {
              current: previous.current + 1,
              elapsed: [...previous.elapsed, seconds(delay)],
              lineStartedAt: now,
              lastDueAt: null,
            }
          : { ...previous, lastDueAt: now },
      );
    }, delay);
    return () => window.clearTimeout(id);
  }, [current, last, lastDueAt, failed]);

  useEffect(() => {
    if (failed) {
      callbacks.current.onFailed();
    }
  }, [failed]);

  useEffect(() => {
    if (!finished) {
      return undefined;
    }
    const id = window.setTimeout(() => callbacks.current.onReady(), READY_DELAY_MS);
    return () => window.clearTimeout(id);
  }, [finished]);

  return (
    <div className="nv-nar">
      {lines.slice(0, current + 1).map((line, index) => {
        const done = index < current || (index === last && finished);
        const state = done ? "done" : failed ? "failed" : "current";
        const time =
          index < current
            ? elapsed[index]
            : done && lastDueAt !== null && settledAt !== null
              ? seconds(Math.max(lastDueAt, settledAt) - lineStartedAt)
              : "";
        return (
          <div key={index} className="nv-nl" data-state={state}>
            <span className="nv-ck" aria-hidden="true">
              {state === "done" ? (
                <CheckIcon size={11} strokeWidth={2.2} />
              ) : state === "failed" ? (
                <span className="nv-dot" />
              ) : (
                <span className="nv-spn" />
              )}
            </span>
            <span className="nv-tx">
              {line.map((segment, segmentIndex) =>
                segment.bold ? (
                  <b key={segmentIndex}>{segment.text}</b>
                ) : (
                  <Fragment key={segmentIndex}>{segment.text}</Fragment>
                ),
              )}
            </span>
            {state === "done" ? <span className="nv-el">{time}</span> : null}
          </div>
        );
      })}
    </div>
  );
}
