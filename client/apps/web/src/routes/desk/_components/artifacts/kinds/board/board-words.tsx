import { deskPopoverClass } from "@/components/desk-chat/desk-popover";
import type {
  DeskBoardCard,
  DeskBoardReading,
  DeskBoardSegment,
} from "@/lib/graphql/desk-board-card";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import { ArtIcon } from "../../desk-art-kinds";
import { cardTime } from "../../record-card/record-card-values";
import {
  ARRIVAL_BEAT_MS,
  formatBoardValue,
  lockedReading,
  readingSource,
  wordPieces,
  wordRuns,
  type WordPiece,
} from "./board-card";

/** What the words need to draw their readings: each reading, and when the board was answered and read. */
export type BoardReadings = Pick<DeskBoardCard, "readings" | "answeredAt" | "readAt">;

/**
 * Cited phrases that light the exhibit they cite while they are pointed at or
 * focused: the exhibit lit now (0 for none), and how to light one.
 */
export type CiteLight = { lit: number; onLight: (numeral: number) => void };

/** When each part of arriving words lands, from when the board's values came in. */
export type WordsArrival = { landAt: number };

const READING_CLASS =
  "mx-0 inline rounded-t-[3px] whitespace-nowrap border-0 border-b border-dotted border-dsk-b-strong bg-transparent px-0.5 py-0 font-plex-mono text-[0.9em] leading-[1.3] font-medium tracking-[-0.015em] text-dsk-fg tabular-nums transition-colors duration-150 ui-focus-ring hover:border-dsk-fg hover:bg-dsk-sunken data-popup-open:border-dsk-fg data-popup-open:bg-dsk-sunken";

const MOVED_CLASS =
  "border-solid border-dsk-warn bg-[color-mix(in_oklab,var(--dsk-warn)_14%,transparent)] hover:border-dsk-warn hover:bg-[color-mix(in_oklab,var(--dsk-warn)_22%,transparent)] data-popup-open:border-dsk-warn data-popup-open:bg-[color-mix(in_oklab,var(--dsk-warn)_22%,transparent)]";

const LINE_CLASS = "mt-1 flex items-center gap-1.5 font-plex-mono text-dsk-sm text-dsk-subtle";

function arrivalStyle(phrase: number, beat: number): CSSProperties {
  return { animationDelay: `${phrase * beat}ms` };
}

/** An exhibit's numeral: a circled figure the words cite it by. */
export function Numeral({ n, struck = false }: { n: number; struck?: boolean }) {
  return <span className={cn("dk-bd-n", struck && "dk-off")}>{n}</span>;
}

/** A reading's popover: what it reads, where it was read from, and when, with whose access. */
function ReadingNote({ reading, board }: { reading: DeskBoardReading; board: BoardReadings }) {
  const t = useT();
  const source = readingSource(reading, t);
  const read = cardTime(board.readAt);
  return (
    <>
      <b className="block text-dsk-base font-medium text-dsk-fg">{reading.label}</b>
      {source !== "" && (
        <span className={LINE_CLASS}>
          <i className="size-1.5 shrink-0 rounded-full bg-dsk-success" aria-hidden="true" />
          {source}
        </span>
      )}
      {reading.was ? (
        <>
          <span className={cn(LINE_CLASS, "text-dsk-warn-fg")}>
            <i className="size-1.5 shrink-0 rounded-full bg-dsk-warn" aria-hidden="true" />
            {t(
              "{0} when answered at {1}",
              formatBoardValue(reading.was),
              cardTime(board.answeredAt),
            )}
          </span>
          <span className={LINE_CLASS}>
            {t("{0} now, read {1} with your access", formatBoardValue(reading.value), read)}
          </span>
        </>
      ) : (
        <span className={LINE_CLASS}>{t("Read {0}, with your access", read)}</span>
      )}
    </>
  );
}

/**
 * A figure in the words that the server read, not the model: a button whose
 * popover says what it is and where it came from. One that moved since the
 * answer keeps the value answered, marked amber; one with no value is said in
 * words.
 */
function Reading({
  segment,
  reading,
  board,
  arrival,
}: {
  segment: DeskBoardSegment;
  reading: DeskBoardReading | undefined;
  board: BoardReadings;
  arrival: WordsArrival | undefined;
}) {
  const t = useT();
  if (!reading || !reading.value) {
    return (
      <span className="inline-flex items-center gap-1 rounded-md bg-[repeating-linear-gradient(-45deg,transparent_0_5px,var(--dsk-b-sub)_5px_6px)] py-px pr-1.5 pl-1 align-baseline text-[0.82em] leading-[1.3] whitespace-nowrap text-dsk-subtle ring-1 ring-dsk-b ring-inset">
        <ArtIcon name="lock" size={11} stroke={2} />
        {reading ? lockedReading(reading, t) : t("a figure that could not be read")}
      </span>
    );
  }
  const answered = segment.text !== "" ? segment.text : formatBoardValue(reading.value);
  return (
    <Popover>
      <PopoverTrigger
        openOnHover
        delay={120}
        render={
          <button
            type="button"
            className={cn(
              READING_CLASS,
              reading.was && MOVED_CLASS,
              arrival && "animate-[dk-bd-land_520ms_both]",
            )}
            style={arrival ? { animationDelay: `${arrival.landAt}ms` } : undefined}
            aria-label={
              reading.was
                ? t(
                    "{0}: {1} when answered, {2} now",
                    reading.label,
                    answered,
                    formatBoardValue(reading.value),
                  )
                : t("{0}: {1}", reading.label, answered)
            }
          />
        }
      >
        {answered}
      </PopoverTrigger>
      <PopoverContent
        side="top"
        sideOffset={8}
        className={cn(deskPopoverClass, "w-auto max-w-80 min-w-48 px-3 pt-2.5 pb-2.5")}
      >
        <ReadingNote reading={reading} board={board} />
      </PopoverContent>
    </Popover>
  );
}

/**
 * A reading still being read: a blank the size of a figure, in the place it will
 * stand. It is the Desk's own skeleton bar, not the shared shimmer, whose ground
 * is the Desk canvas's own colour: on it that shimmer drew nothing, and the
 * sentence read as though a word were missing.
 */
function ReadingBlank() {
  return <span className="dk-sk dk-bd-blank" aria-hidden="true" />;
}

/** A cited phrase and the numeral of the exhibit it cites; lit while that exhibit is. */
function Cite({ segment, cite }: { segment: DeskBoardSegment; cite: CiteLight | undefined }) {
  const t = useT();
  const lit = cite !== undefined && cite.lit === segment.numeral;
  const phrase = (
    <span className={cn("dk-bd-hl", segment.tone === "warn" && "dk-warn", lit && "dk-bd-lit")}>
      {segment.text}
    </span>
  );
  if (!cite) {
    return (
      <span>
        {phrase}
        <Numeral n={segment.numeral} struck={segment.struck} />
      </span>
    );
  }
  return (
    <span onMouseEnter={() => cite.onLight(segment.numeral)} onMouseLeave={() => cite.onLight(0)}>
      {phrase}
      <button
        type="button"
        className={cn(
          "dk-bd-n cursor-pointer ui-focus-ring",
          segment.struck && "dk-off",
          lit && "dk-bd-lit",
        )}
        aria-label={t("Show exhibit {0}", segment.numeral)}
        onFocus={() => cite.onLight(segment.numeral)}
        onBlur={() => cite.onLight(0)}
      >
        {segment.numeral}
      </button>
    </span>
  );
}

function piece(
  part: WordPiece,
  index: number,
  props: {
    board: BoardReadings | null;
    readings: ReadonlyMap<string, DeskBoardReading>;
    cite: CiteLight | undefined;
    arrival: WordsArrival | undefined;
  },
) {
  if (part.kind === "text") {
    return (
      // oxlint-disable-next-line react/no-array-index-key -- the words are a fixed run of the text
      <span key={index}>
        {wordRuns(part.text).map((run, at) =>
          run.whole ? (
            // oxlint-disable-next-line react/no-array-index-key -- runs are a fixed cut of the text
            <span key={at} className="whitespace-nowrap">
              {run.text}
            </span>
          ) : (
            run.text
          ),
        )}
      </span>
    );
  }
  if (part.kind === "cite") {
    // oxlint-disable-next-line react/no-array-index-key -- the words are a fixed run of the text
    return <Cite key={index} segment={part.segment} cite={props.cite} />;
  }
  if (!props.board) {
    // oxlint-disable-next-line react/no-array-index-key -- the words are a fixed run of the text
    return <ReadingBlank key={index} />;
  }
  return (
    <Reading
      // oxlint-disable-next-line react/no-array-index-key -- the words are a fixed run of the text
      key={index}
      segment={part.segment}
      reading={props.readings.get(part.segment.reading)}
      board={props.board}
      arrival={props.arrival}
    />
  );
}

/**
 * A board's words: the verdict, a prose block or a note. Each phrase the answer
 * rests on is marked and cites its exhibit by numeral; each `{key}` is a
 * reading. Given `board: null` the readings are blanks, while the board is read.
 * Given `cite`, a cited phrase lights its exhibit while pointed at or focused.
 * Given `arrival`, the words come in phrase by phrase and the readings land.
 */
export function BoardWords({
  segments,
  board,
  cite,
  arrival,
  beat = ARRIVAL_BEAT_MS,
}: {
  segments: readonly DeskBoardSegment[];
  board: BoardReadings | null;
  cite?: CiteLight;
  arrival?: WordsArrival;
  /** How far apart the phrases arrive. */
  beat?: number;
}) {
  const readings = new Map((board?.readings ?? []).map((reading) => [reading.key, reading]));
  const { pieces } = wordPieces(segments);
  const props = { board, readings, cite, arrival };
  if (!arrival) {
    return pieces.map((part, index) => piece(part, index, props));
  }
  // The phrases arrive in turn: every piece of one phrase fades in together.
  const phrases: { phrase: number; parts: { part: WordPiece; index: number }[] }[] = [];
  pieces.forEach((part, index) => {
    const last = phrases.at(-1);
    if (last && last.phrase === part.phrase) {
      last.parts.push({ part, index });
    } else {
      phrases.push({ phrase: part.phrase, parts: [{ part, index }] });
    }
  });
  return phrases.map((group) => (
    <span
      key={group.phrase}
      className="animate-[dk-word_420ms_var(--dk-ease)_both]"
      style={arrivalStyle(group.phrase, beat)}
    >
      {group.parts.map(({ part, index }) => piece(part, index, props))}
    </span>
  ));
}
