import type {
  DeskBoardBlock,
  DeskBoardCard,
  DeskBoardCheck,
  DeskBoardReading,
  DeskBoardRow,
  DeskBoardRowMeasure,
  DeskBoardSegment,
  DeskBoardSource,
  DeskBoardValue,
} from "@/lib/graphql/desk-board-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatDurationFromSeconds } from "@trenova/shared/lib/date";
import {
  cardDayAfter,
  cardMomentAfter,
  cardTime,
  money,
} from "../../record-card/record-card-values";

/** How much a board's block asks of the reader: blocks it, worth a look, worth knowing, clear. */
export type BoardTone = "bad" | "warn" | "info" | "ok" | "";

/** How many figures the card sets side by side in one row. */
export const FIGURES_PER_ROW = 4;

/**
 * The rows a run of figures is drawn in: one row of up to four, or balanced
 * rows of at most four, the longer first. Five are three and two rather than
 * four and a lone one; eight are four and four.
 */
export function figureRows(count: number): number[] {
  if (count <= 0) {
    return [];
  }
  const rows = Math.ceil(count / FIGURES_PER_ROW);
  const base = Math.floor(count / rows);
  const longer = count % rows;

  return Array.from({ length: rows }, (_, index) => (index < longer ? base + 1 : base));
}

export function boardTone(value: string): BoardTone {
  return value === "bad" || value === "warn" || value === "info" || value === "ok" ? value : "";
}

/** The tone class every board part takes, so `.dk-warn` reads the same everywhere. */
export function toneClass(tone: string): string {
  const known = boardTone(tone);
  return known === "" ? "" : `dk-${known}`;
}

/** A value the server read, as the board writes it. */
export function formatBoardValue(value: DeskBoardValue | null | undefined): string {
  if (!value) return "";
  const number = Number(value.number);
  switch (value.kind) {
    case "count":
      return Number.isFinite(number) ? new Intl.NumberFormat().format(number) : "";
    case "money":
      return money(value.number, value.currency || "USD");
    case "percent":
      return Number.isFinite(number)
        ? new Intl.NumberFormat(undefined, {
            style: "percent",
            maximumFractionDigits: 1,
          }).format(number)
        : "";
    case "duration":
      return formatDurationFromSeconds(Number.isFinite(number) ? Math.abs(number) : 0);
    case "text":
      return value.text;
    default:
      return Number.isFinite(number)
        ? new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(number)
        : "";
  }
}

/** A number of records of one kind, as a source line names them. */
export function recordsPhrase(resource: string, count: number, t: TranslateFn): string {
  switch (resource) {
    case "shipment":
      return t("{0, plural, one {# shipment} other {# shipments}}", count);
    case "invoice":
      return t("{0, plural, one {# invoice} other {# invoices}}", count);
    case "journal_entry":
      return t("{0, plural, one {# journal entry} other {# journal entries}}", count);
    case "carrier":
      return t("{0, plural, one {# carrier} other {# carriers}}", count);
    case "customer":
      return t("{0, plural, one {# customer} other {# customers}}", count);
    case "worker":
      return t("{0, plural, one {# worker} other {# workers}}", count);
    case "tractor":
      return t("{0, plural, one {# tractor} other {# tractors}}", count);
    case "trailer":
      return t("{0, plural, one {# trailer} other {# trailers}}", count);
    case "location":
      return t("{0, plural, one {# location} other {# locations}}", count);
    default:
      return t("{0, plural, one {# record} other {# records}}", count);
  }
}

/** One record a block could not read, named by its kind. */
export function unreadableRecord(resource: string, t: TranslateFn): string {
  switch (resource) {
    case "shipment":
      return t("This shipment could not be read");
    case "invoice":
      return t("This invoice could not be read");
    case "journal_entry":
      return t("This journal entry could not be read");
    case "carrier":
      return t("This carrier could not be read");
    case "customer":
      return t("This customer could not be read");
    case "worker":
      return t("This worker could not be read");
    case "tractor":
      return t("This tractor could not be read");
    case "trailer":
      return t("This trailer could not be read");
    case "location":
      return t("This location could not be read");
    default:
      return t("This record could not be read");
  }
}

/** Where a block was read from, as its source line says it after "Read from". */
export function sourcePhrase(source: DeskBoardSource, t: TranslateFn): string {
  switch (source.kind) {
    case "records":
      return recordsPhrase(source.resource, source.count, t);
    case "activity":
      return t("its history");
    case "analysis":
      return source.title ? t("the analysis “{0}”", source.title) : t("an analysis");
    case "table":
      return source.title ? t("the table “{0}”", source.title) : t("a table");
    case "metrics":
      return t("Trenova's figures");
    case "checks":
      return source.resource
        ? recordsPhrase(source.resource, source.count, t)
        : t("the checks above");
    case "mixed":
      return t("several sources");
    default:
      return "";
  }
}

/** The line under a block's tab: where it read from, and how much of it the reader may read. */
export function sourceLine(source: DeskBoardSource, t: TranslateFn): string {
  const from = sourcePhrase(source, t);
  if (from === "") return "";
  const hidden = source.count - source.readable;
  if ((source.kind === "records" || source.kind === "checks") && source.resource && hidden > 0) {
    return t("Read from {0} · {1} you may read", from, source.readable);
  }
  return t("Read from {0}", from);
}

/** Where a reading was read from, with the aggregate in words: counted, summed, the largest. */
export function readingSource(reading: DeskBoardReading, t: TranslateFn): string {
  const source = reading.source;
  if (source.kind === "records" && source.resource) {
    const records = recordsPhrase(source.resource, source.count, t);
    const from =
      reading.aggregate === "count"
        ? t("Counted from {0}", records)
        : reading.aggregate === "sum"
          ? t("Summed from {0}", records)
          : reading.aggregate === "max"
            ? t("The largest of {0}", records)
            : t("Read from {0}", records);
    const hidden = source.count - source.readable;
    return hidden > 0 ? t("{0} · {1} you may read", from, source.readable) : from;
  }
  return sourceLine(source, t);
}

type ReadingKind = "amount" | "count" | "time" | "figure";

function readingKind(reading: DeskBoardReading): ReadingKind {
  if (reading.aggregate === "count") return "count";
  if (reading.measure === "amountDue") return "amount";
  if (measureIsTrack(reading.measure)) return "time";
  return "figure";
}

/**
 * A reading with no value, said in words where the value would stand: one the
 * reader may not read because some of its records are not theirs, or one that
 * could not be read. Never a blank, never a zero.
 */
export function lockedReading(reading: DeskBoardReading, t: TranslateFn): string {
  const source = reading.source;
  const withheld = source.kind === "records" && source.readable < source.count;
  switch (readingKind(reading)) {
    case "amount":
      return withheld ? t("an amount you may not read") : t("an amount that could not be read");
    case "count":
      return withheld ? t("a count you may not read") : t("a count that could not be read");
    case "time":
      return withheld ? t("a time you may not read") : t("a time that could not be read");
    default:
      return withheld ? t("a figure you may not read") : t("a figure that could not be read");
  }
}

const CITATION = /\[([^[\]\n]+)\]\[(\d{1,2})\]/gu;
const READING_TOKEN = /\{([A-Za-z][A-Za-z0-9_]{0,31})\}/gu;

function wordSegments(text: string): DeskBoardSegment[] {
  const segments: DeskBoardSegment[] = [];
  let last = 0;
  for (const match of text.matchAll(READING_TOKEN)) {
    const at = match.index ?? 0;
    if (at > last) {
      segments.push({
        text: text.slice(last, at),
        numeral: 0,
        tone: "",
        struck: false,
        reading: "",
      });
    }
    segments.push({ text: "", numeral: 0, tone: "", struck: false, reading: match[1] });
    last = at + match[0].length;
  }
  if (last < text.length) {
    segments.push({ text: text.slice(last), numeral: 0, tone: "", struck: false, reading: "" });
  }
  return segments;
}

/**
 * A board's words as the agent wrote them, split as the server splits them
 * (`board.Segments`): a cited phrase `[phrase][n]` cites exhibit n, and a
 * `{key}` places a reading, with no value yet. What a reply draws while the
 * board it showed is still being read.
 */
export function writtenSegments(text: string): DeskBoardSegment[] {
  const segments: DeskBoardSegment[] = [];
  let last = 0;
  for (const match of text.matchAll(CITATION)) {
    const at = match.index ?? 0;
    if (at > last) segments.push(...wordSegments(text.slice(last, at)));
    const numeral = Number(match[2]);
    segments.push({
      text: match[1],
      numeral: numeral >= 1 ? numeral : 0,
      tone: "",
      struck: false,
      reading: "",
    });
    last = at + match[0].length;
  }
  if (last < text.length) segments.push(...wordSegments(text.slice(last)));
  return segments;
}

/** A run of the words as it arrives: plain words, a cited phrase or a reading, and its phrase. */
export type WordPiece =
  | { kind: "text"; text: string; phrase: number }
  | { kind: "cite"; segment: DeskBoardSegment; phrase: number }
  | { kind: "reading"; segment: DeskBoardSegment; phrase: number };

const CLAUSE = /[^,.;:!?]+[,.;:!?]*\s*|[,.;:!?]+\s*/gu;
const CLAUSE_END = /[,.;:!?]\s*$/u;

/**
 * The words cut into the phrases they arrive by: each clause of plain words is
 * a phrase, ending at its punctuation, and a cited phrase or a reading arrives
 * with the clause it stands in.
 */
export function wordPieces(segments: readonly DeskBoardSegment[]): {
  pieces: WordPiece[];
  phrases: number;
} {
  const pieces: WordPiece[] = [];
  let phrase = 0;
  let open = false;
  for (const segment of segments) {
    if (segment.reading !== "") {
      pieces.push({ kind: "reading", segment, phrase });
      open = true;
      continue;
    }
    if (segment.numeral > 0) {
      pieces.push({ kind: "cite", segment, phrase });
      open = true;
      continue;
    }
    for (const clause of segment.text.match(CLAUSE) ?? []) {
      pieces.push({ kind: "text", text: clause, phrase });
      open = true;
      if (CLAUSE_END.test(clause)) {
        phrase += 1;
        open = false;
      }
    }
  }
  return { pieces, phrases: open ? phrase + 1 : phrase };
}

/** How far apart the parts of a board arriving in the conversation land, as the mockup times it. */
export const ARRIVAL_BEAT_MS = 240;

/** The moments, from when the board's values land, each part of an arriving board takes its place. */
export type BoardArrival = {
  /** When the readings in the words land, after the words and the exhibits. */
  landAt: number;
  /** When the exhibit with this place among the exhibits rises. */
  exhibitAt: (index: number) => number;
  /** When the first of its rows follows. */
  rowsAt: (index: number) => number;
};

export function boardArrival(phrases: number, exhibits: number): BoardArrival {
  const rowsFrom = phrases + exhibits + 1;
  return {
    landAt: rowsFrom * ARRIVAL_BEAT_MS,
    exhibitAt: (index) => (phrases + index) * ARRIVAL_BEAT_MS,
    rowsAt: (index) => (rowsFrom + index) * ARRIVAL_BEAT_MS,
  };
}

/** The ids of the records in a list a person may read, in the order the list draws them: the order a record opened from it steps through. */
export function readableIds(records: readonly { id: string; readable: boolean }[]): string[] {
  return records.filter((record) => record.readable).map((record) => record.id);
}

/** How many of a list's rows the conversation draws; the rest are on the board. */
export const CONVERSATION_ROWS = 6;

/** A list cut to the rows the conversation draws, and how many are left for the board. */
export function cutRows<T>(
  rows: readonly T[],
  limit = CONVERSATION_ROWS,
): { shown: T[]; more: number } {
  return rows.length > limit
    ? { shown: rows.slice(0, limit), more: rows.length - limit }
    : { shown: [...rows], more: 0 };
}

/** What a note's tone means, said above it. */
export function noteLabel(tone: string, t: TranslateFn): string {
  switch (tone) {
    case "bad":
      return t("Blocks it");
    case "warn":
      return t("Worth acting on");
    default:
      return t("What I'd do");
  }
}

/** A checks block's checks grouped by how much they ask, worst first. */
export function checkGroups(
  checks: readonly DeskBoardCheck[],
  t: TranslateFn,
): { tone: BoardTone; label: string; checks: DeskBoardCheck[] }[] {
  const labels: Record<Exclude<BoardTone, "">, string> = {
    bad: t("Blocks it"),
    warn: t("Needs a look"),
    info: t("Worth knowing"),
    ok: t("All clear"),
  };
  return (["bad", "warn", "info", "ok"] as const)
    .map((tone) => ({
      tone,
      label: labels[tone],
      checks: checks.filter((check) => boardTone(check.tone) === tone),
    }))
    .filter((group) => group.checks.length > 0);
}

/** A record's measure, read beside its row: what it is measured against, and how it stands. */
export type MeasureReading = {
  /** What the measure is held against, whole: "Due Mar 19, 11:00 AM at Rockford, IL". */
  against: string;
  /** What the row draws of it, written as precisely as its age needs: "Due Mar 19". */
  shown: string;
  /**
   * The stop it is measured at, when it names one and a moment: the row draws the moment and
   * then the stop in a span of its own, the first thing to go when the row is narrow.
   */
  at: { verb: "due" | "delivered"; when: string; place: string } | null;
  /** How the record stands: "1h 40m behind", "on time". */
  standing: string;
  tone: BoardTone;
  /** How far along the track the bar reaches, from 0 to 1. */
  fraction: number;
  /** The record waits at a stop rather than on the road. */
  held: boolean;
  /** The lateness is the estimate's, for a window still open. */
  projected: boolean;
};

const WEEK_SECONDS = 7 * 24 * 60 * 60;

/**
 * How precisely a row writes a moment. A date (when an invoice falls due, the day it is dated)
 * is its day: `invoice.DueDateFromPaymentTerm` adds whole days to the invoice date, so its time
 * of day means nothing. A moment within a week of now, either way, keeps its time, as a load's
 * appointment today does; one further off is its day, as a load 248 days late needs no minute.
 * A day carries its year only when it is not now's.
 */
export function measureMoment(now: number, at: number, kind: "date" | "moment"): string {
  if (!at) return "";
  if (kind === "moment" && Math.abs(at - now) <= WEEK_SECONDS) return cardMomentAfter(now, at);
  return cardDayAfter(now, at);
}

function seconds(value: DeskBoardValue): number {
  const number = Number(value.number);
  return Number.isFinite(number) ? Math.max(0, number) : 0;
}

export function measureReading(
  measure: DeskBoardRowMeasure,
  furthest: number,
  now: number,
  t: TranslateFn,
): MeasureReading {
  const amount = formatBoardValue(measure.value);
  const tone = boardTone(measure.tone);
  const at = measure.at ?? 0;
  const value = measure.value.kind === "duration" ? seconds(measure.value) : 0;
  const fraction = furthest > 0 ? Math.min(1, value / furthest) : 0;
  const base = { tone, fraction, held: false, projected: false, at: null };

  const whole = at ? cardMomentAfter(now, at) : "";
  const when = measureMoment(now, at, "moment");
  const day = measureMoment(now, at, "date");
  const place = measure.place;
  const due = (): Pick<MeasureReading, "against" | "shown" | "at"> => {
    if (!at) {
      const text = place ? t("At {0}", place) : "";
      return { against: text, shown: text, at: null };
    }
    if (!place) return { against: t("Due {0}", whole), shown: t("Due {0}", when), at: null };
    return {
      against: t("Due {0} at {1}", whole, place),
      shown: t("Due {0}", when),
      at: { verb: "due", when, place },
    };
  };
  const delivered = (): Pick<MeasureReading, "against" | "shown" | "at"> => {
    if (!at) return { against: t("Delivered"), shown: t("Delivered"), at: null };
    if (!place) {
      return { against: t("Delivered {0}", whole), shown: t("Delivered {0}", when), at: null };
    }
    return {
      against: t("Delivered {0} at {1}", whole, place),
      shown: t("Delivered {0}", when),
      at: { verb: "delivered", when, place },
    };
  };
  const plain = (text: string): Pick<MeasureReading, "against" | "shown"> => ({
    against: text,
    shown: text,
  });

  switch (measure.measure) {
    case "lateness":
      switch (measure.state) {
        case "late":
          return { ...base, ...due(), standing: t("{0} behind", amount) };
        case "held":
          return { ...base, held: true, ...due(), standing: t("{0} at the stop", amount) };
        case "projected":
          return {
            ...base,
            projected: true,
            ...due(),
            standing: t("estimated {0} late", amount),
          };
        case "due":
          return { ...base, fraction: 0, ...due(), standing: t("not estimated") };
        case "delivered":
          return {
            ...base,
            fraction: 0,
            ...delivered(),
            standing: value > 0 ? t("{0} late", amount) : t("on time"),
          };
        default:
          return { ...base, fraction: 0, ...due(), standing: t("on time") };
      }
    case "amountDue":
      return {
        ...base,
        fraction: 0,
        ...plain(day ? t("Due {0}", day) : ""),
        standing: measure.state === "settled" ? t("paid") : t("{0} due", amount),
      };
    case "pendingFor":
      return {
        ...base,
        against: whole ? t("Since {0}", whole) : "",
        shown: when ? t("Since {0}", when) : "",
        standing: measure.state === "settled" ? t("settled") : t("waiting {0}", amount),
      };
    case "age":
      return {
        ...base,
        ...plain(day ? t("Dated {0}", day) : ""),
        standing: t("{0} old", amount),
      };
    default:
      return { ...base, fraction: 0, against: "", shown: "", standing: amount };
  }
}

/** The furthest a block's records run by their measure, so each row's track is drawn against it. */
export function furthestOf(rows: readonly DeskBoardRow[]): number {
  return rows.reduce((furthest, row) => {
    const value = row.measure?.value;
    return value && value.kind === "duration" ? Math.max(furthest, seconds(value)) : furthest;
  }, 0);
}

/** A measure that reads as a track along the row; money reads as a figure at its end. */
export function measureIsTrack(measure: string): boolean {
  return measure === "lateness" || measure === "pendingFor" || measure === "age";
}

/** A record's initials, for the face a comparison heads its column with. */
export function initialsOf(label: string): string {
  const words = label
    .replace(/[^\p{L}\p{N}\s]/gu, " ")
    .split(/\s+/u)
    .filter(Boolean);
  const letters = words.length > 1 ? [words[0], words[1]] : words;
  return letters
    .map((word) => word.charAt(0))
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

/** The reads the foot lists: each exhibit's numeral, what it read, and whether it could. */
export type BoardRead = { numeral: number; phrase: string; struck: boolean };

export function boardReads(card: DeskBoardCard, t: TranslateFn): BoardRead[] {
  return card.blocks
    .filter((block) => block.numeral > 0)
    .map((block) => ({
      numeral: block.numeral,
      phrase:
        block.state === "unreadable"
          ? t("could not be read")
          : sourcePhrase(block.source, t) || t("several sources"),
      struck: block.state === "unreadable",
    }));
}

/** The time the board was read, as the postmark in its corner says it. */
export function readStamp(card: DeskBoardCard, t: TranslateFn): string {
  return t("read {0}", cardTime(card.readAt));
}

/**
 * Blocks drawn half width: the agent's half blocks taken two at a time along each run, so a
 * block left without a partner, alone or last of an odd run, is drawn at full width.
 */
export function halfBlocks(blocks: readonly Pick<DeskBoardBlock, "half">[]): boolean[] {
  const halves = blocks.map(() => false);
  let run = 0;
  for (let index = 0; index <= blocks.length; index += 1) {
    if (index < blocks.length && blocks[index].half) {
      run += 1;
      continue;
    }
    const paired = run - (run % 2);
    for (let offset = 0; offset < paired; offset += 1) {
      halves[index - run + offset] = true;
    }
    run = 0;
  }
  return halves;
}

/**
 * Where a reply's foot stands: inside the last block when that block is an
 * exhibit drawn full width, else as one full-width line under the desk, which
 * closes the ragged bottom of two exhibits of unequal height side by side and
 * never cramps the foot into half the reply. A desk with no blocks has it under.
 */
export function replyFootPlace(
  blocks: readonly { type: string }[],
  halves: readonly boolean[],
  isExhibit: (type: string) => boolean,
): "inside" | "under" {
  const last = blocks.length - 1;
  return last >= 0 && isExhibit(blocks[last].type) && !halves[last] ? "inside" : "under";
}

/** A run of words: plain, or a token that must stand whole on one line. */
export type WordRun = { text: string; whole: boolean };

const UNBREAKABLE = /[\p{L}\p{N}]+(?:-[\p{L}\p{N}]+)+/gu;
const DIGIT = /\p{N}/u;

/**
 * Words cut into plain runs and the tokens a line must never break inside: groups
 * of letters and digits joined by hyphens that hold a digit, such as a record's
 * number ("SEED-INV-0006") or a date ("2026-11-29"), which a browser would
 * otherwise break after a hyphen. The runs joined give back the words unchanged.
 */
export function wordRuns(text: string): WordRun[] {
  const runs: WordRun[] = [];
  let last = 0;
  for (const match of text.matchAll(UNBREAKABLE)) {
    if (!DIGIT.test(match[0])) continue;
    const at = match.index ?? 0;
    if (at > last) runs.push({ text: text.slice(last, at), whole: false });
    runs.push({ text: match[0], whole: true });
    last = at + match[0].length;
  }
  if (last < text.length) runs.push({ text: text.slice(last), whole: false });
  return runs;
}
