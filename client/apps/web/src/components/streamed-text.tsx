import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { cn } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { Fragment, useEffect, useMemo, useState, type ReactNode } from "react";

const STREAM_TICK_MS = 28;
const MIN_CHARS_PER_TICK = 2;
const MAX_CHARS_PER_TICK = 4;

export type StreamedSegment = {
  text: string;
  /** Renders the segment as a link-like action instead of plain text. */
  onActivate?: () => void;
  label?: string;
  /** Shown in a hover card on an actionable segment; mounted only while open. */
  preview?: ReactNode;
  /** A figure or name the eye should land on, set in the foreground. */
  emphasis?: boolean;
  /** Colours an actionable segment for a warning or a failure. */
  tone?: "warning" | "danger";
};

const SEGMENT_ACTION_CLASS =
  "ui-focus-ring decoration-border-strong hover:bg-brand/20 rounded-sm underline decoration-1 underline-offset-4 transition-colors";

const SEGMENT_TONE_CLASS = {
  warning: "text-warning decoration-warning/60 hover:bg-warning/20",
  danger: "text-danger decoration-danger/60 hover:bg-danger/20",
} as const;

type Word = { text: string; start: number };

function splitWords(text: string, offset: number): Word[] {
  const words: Word[] = [];
  const pattern = /\S+\s*|\s+/g;
  let match: RegExpExecArray | null;
  while ((match = pattern.exec(text))) {
    words.push({ text: match[0], start: offset + match.index });
  }
  return words;
}

function SegmentAction({ segment, children }: { segment: StreamedSegment; children: ReactNode }) {
  const button = (
    <button
      type="button"
      aria-label={segment.label}
      onClick={segment.onActivate}
      className={cn(
        SEGMENT_ACTION_CLASS,
        segment.emphasis && "text-foreground font-medium",
        segment.tone && SEGMENT_TONE_CLASS[segment.tone],
      )}
    >
      {children}
    </button>
  );
  if (!segment.preview) return button;

  return (
    <HoverCard>
      <HoverCardTrigger delay={250} render={button} />
      <HoverCardContent align="start" className="w-80 p-0">
        {segment.preview}
      </HoverCardContent>
    </HoverCard>
  );
}

/**
 * A sentence that arrives a few characters at a time, each word resolving
 * from a blur, with a caret until it is whole. Segments with an action read
 * as underlined links, and one with a preview opens it on hover. A prefix is
 * shown whole from the start. Reduced motion shows the sentence whole at once.
 */
export function StreamedText({
  segments,
  className,
  streamKey,
  prefix,
}: {
  segments: StreamedSegment[];
  className?: string;
  /** Restarts the stream when it changes; the same sentence never streams twice. */
  streamKey: string;
  prefix?: ReactNode;
}) {
  const reduceMotion = useReducedMotion() ?? false;
  const layout = useMemo(() => {
    let offset = 0;
    return segments.map((segment) => {
      const words = splitWords(segment.text, offset);
      offset += segment.text.length;
      return { segment, words };
    });
  }, [segments]);
  const total = segments.reduce((sum, segment) => sum + segment.text.length, 0);
  const [revealed, setRevealed] = useState(reduceMotion ? total : 0);

  useEffect(() => {
    if (reduceMotion) {
      setRevealed(total);
      return undefined;
    }
    setRevealed(0);
    const timer = window.setInterval(() => {
      setRevealed((current) => {
        const step =
          MIN_CHARS_PER_TICK +
          Math.floor(Math.random() * (MAX_CHARS_PER_TICK - MIN_CHARS_PER_TICK + 1));
        const next = Math.min(total, current + step);
        if (next >= total) window.clearInterval(timer);
        return next;
      });
    }, STREAM_TICK_MS);
    return () => window.clearInterval(timer);
  }, [streamKey, total, reduceMotion]);

  const done = revealed >= total;

  return (
    <p className={cn(className, !done && "ui-stream-caret")} aria-live="polite" aria-busy={!done}>
      {prefix}
      {layout.map(({ segment, words }, index) => {
        const visible: ReactNode[] = words
          .filter((word) => word.start < revealed)
          .map((word) => (
            <span key={word.start} className="animate-word-in">
              {word.text}
            </span>
          ));
        if (visible.length === 0) return null;
        if (!segment.onActivate) {
          return segment.emphasis ? (
            <b key={index} className="text-foreground font-semibold">
              {visible}
            </b>
          ) : (
            <Fragment key={index}>{visible}</Fragment>
          );
        }
        return (
          <SegmentAction key={index} segment={segment}>
            {visible}
          </SegmentAction>
        );
      })}
    </p>
  );
}
