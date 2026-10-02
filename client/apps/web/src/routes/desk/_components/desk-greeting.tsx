import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";

/** The beat between one word of the greeting and the next. */
const WORD_STEP_MS = 55;

export type DeskGreetingProps = {
  /** The first line of the headline: the time of day, and the person by name. */
  greeting: string;
  /** The day, as the person reads it. */
  dateline: string;
  /** The same day, machine-readable, for the `<time>` that carries it. */
  isoDate: string;
  /** The second line: what the day looks like, or the question the page asks. */
  headline: ReactNode;
  /** Staggers each line in behind the one above it, in reading order. */
  entrance: (step: number) => CSSProperties;
  className?: string;
};

/**
 * The top of the Desk's front page: the day, the person's name, and one
 * line on what is waiting.
 *
 * It is set in type alone. The dateline is a small line of the mono face,
 * the way a date is set at the head of a letter; the greeting is the one
 * place the product speaks to the person rather than about the work, and
 * takes the display serif for it, large and light; and the headline under
 * it is a quiet line of the body face. No colour, no mark, no wash — the
 * page is a room to be entered, so the words stand centred, each line
 * rising in once a beat behind the one above it, and then hold still.
 */
export function DeskGreeting({
  greeting,
  dateline,
  isoDate,
  headline,
  entrance,
  className,
}: DeskGreetingProps) {
  return (
    <header className={cn("flex flex-col items-center gap-3 text-center", className)}>
      <p className="text-foreground-subtle animate-rise font-mono text-xs" style={entrance(0)}>
        <time dateTime={isoDate}>{dateline}</time>
      </p>
      <h1 className="font-display text-foreground max-w-3xl text-4xl font-normal tracking-tight text-balance sm:text-5xl">
        <RisingWords text={greeting} from={1} />
      </h1>
      <div
        data-slot="desk-headline"
        className="text-foreground-muted animate-rise max-w-xl text-base text-balance sm:text-lg"
        style={entrance(2)}
      >
        {headline}
      </div>
    </header>
  );
}

/**
 * A line that arrives a word at a time, each rising into place a beat after
 * the one before it, so the greeting is read as it is said rather than
 * appearing as a block. Under reduced motion the global rule cuts every
 * rise, so the words simply stand.
 */
export function RisingWords({ text, from }: { text: string; from: number }) {
  const words = text.split(" ");
  return (
    <>
      {words.map((word, index) => (
        <span
          key={`${index}-${word}`}
          className="animate-rise inline-block"
          style={{ animationDelay: `${from * 45 + index * WORD_STEP_MS}ms` }}
        >
          {word}
          {index < words.length - 1 ? " " : ""}
        </span>
      ))}
    </>
  );
}
