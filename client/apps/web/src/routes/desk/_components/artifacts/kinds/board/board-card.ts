import type {
  DeskBoardBlock,
  DeskBoardCard,
  DeskBoardCheck,
  DeskBoardRow,
  DeskBoardRowMeasure,
  DeskBoardSource,
  DeskBoardValue,
} from "@/lib/graphql/desk-board-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatDurationFromSeconds } from "@trenova/shared/lib/date";
import { cardMomentAfter, cardTime, money } from "../../record-card/record-card-values";

/** How much a board's block asks of the reader: blocks it, worth a look, worth knowing, clear. */
export type BoardTone = "bad" | "warn" | "info" | "ok" | "";

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
  /** What the measure is held against: "Due 2:00 PM", "Delivered 4:52 PM". */
  against: string;
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
  const base = { tone, fraction, held: false, projected: false };

  switch (measure.measure) {
    case "lateness":
      switch (measure.state) {
        case "late":
          return {
            ...base,
            against: at ? t("Due {0}", cardMomentAfter(now, at)) : "",
            standing: t("{0} behind", amount),
          };
        case "held":
          return {
            ...base,
            held: true,
            against: at ? t("Due {0}", cardMomentAfter(now, at)) : "",
            standing: t("{0} at the stop", amount),
          };
        case "projected":
          return {
            ...base,
            projected: true,
            against: at ? t("Due {0}", cardMomentAfter(now, at)) : "",
            standing: t("estimated {0} late", amount),
          };
        case "due":
          return {
            ...base,
            fraction: 0,
            against: at ? t("Due {0}", cardMomentAfter(now, at)) : "",
            standing: t("not estimated"),
          };
        case "delivered":
          return {
            ...base,
            fraction: 0,
            against: at ? t("Delivered {0}", cardMomentAfter(now, at)) : t("Delivered"),
            standing: value > 0 ? t("{0} late", amount) : t("on time"),
          };
        default:
          return {
            ...base,
            fraction: 0,
            against: at ? t("Due {0}", cardMomentAfter(now, at)) : "",
            standing: t("on time"),
          };
      }
    case "amountDue":
      return {
        ...base,
        fraction: 0,
        against: at ? t("Due {0}", cardMomentAfter(now, at)) : "",
        standing: measure.state === "settled" ? t("paid") : t("{0} due", amount),
      };
    case "pendingFor":
      return {
        ...base,
        against: at ? t("Since {0}", cardMomentAfter(now, at)) : "",
        standing: measure.state === "settled" ? t("settled") : t("waiting {0}", amount),
      };
    case "age":
      return {
        ...base,
        against: at ? t("Dated {0}", cardMomentAfter(now, at)) : "",
        standing: t("{0} old", amount),
      };
    default:
      return { ...base, fraction: 0, against: "", standing: amount };
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
