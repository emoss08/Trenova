import * as chrono from "chrono-node";
import { ParsingContext } from "chrono-node";

const tShorthandParser: chrono.Parser = {
  pattern: () => /\bt([+-](\d+))?(?:\s+(\d{4}))?\b/i,
  extract: (context: ParsingContext, match: RegExpMatchArray) => {
    const sign = match[1]?.[0] === "-" ? -1 : 1;
    const daysOffset = match[2] ? sign * parseInt(match[2], 10) : 0;
    const timeStr = match[3];

    const result = context.createParsingComponents();
    const target = new Date(context.refDate);
    target.setDate(target.getDate() + daysOffset);

    result.assign("year", target.getFullYear());
    result.assign("month", target.getMonth() + 1);
    result.assign("day", target.getDate());

    if (timeStr) {
      const hour = parseInt(timeStr.slice(0, 2), 10);
      const minute = parseInt(timeStr.slice(2, 4), 10);
      result.assign("hour", hour);
      result.assign("minute", minute);
      result.assign("second", 0);
    }

    return result;
  },
};

const weekOffsetParser: chrono.Parser = {
  pattern: () => /\bw([+-])(\d+)(?:\s+(\d{4}))?\b/i,
  extract: (context: ParsingContext, match: RegExpMatchArray) => {
    const sign = match[1] === "-" ? -1 : 1;
    const weeks = parseInt(match[2], 10);
    const timeStr = match[3];

    const result = context.createParsingComponents();
    const target = new Date(context.refDate);
    target.setDate(target.getDate() + sign * weeks * 7);

    result.assign("year", target.getFullYear());
    result.assign("month", target.getMonth() + 1);
    result.assign("day", target.getDate());

    if (timeStr) {
      result.assign("hour", parseInt(timeStr.slice(0, 2), 10));
      result.assign("minute", parseInt(timeStr.slice(2, 4), 10));
      result.assign("second", 0);
    }

    return result;
  },
};

const monthOffsetParser: chrono.Parser = {
  pattern: () => /\bm([+-])(\d+)(?:\s+(\d{4}))?\b/i,
  extract: (context: ParsingContext, match: RegExpMatchArray) => {
    const sign = match[1] === "-" ? -1 : 1;
    const months = parseInt(match[2], 10);
    const timeStr = match[3];

    const result = context.createParsingComponents();
    const target = new Date(context.refDate);
    target.setMonth(target.getMonth() + sign * months);

    result.assign("year", target.getFullYear());
    result.assign("month", target.getMonth() + 1);
    result.assign("day", target.getDate());

    if (timeStr) {
      result.assign("hour", parseInt(timeStr.slice(0, 2), 10));
      result.assign("minute", parseInt(timeStr.slice(2, 4), 10));
      result.assign("second", 0);
    }

    return result;
  },
};

const relativeHoursParser: chrono.Parser = {
  pattern: () => /\+(\d+)h\b/i,
  extract: (context: ParsingContext, match: RegExpMatchArray) => {
    const hours = parseInt(match[1], 10);

    const result = context.createParsingComponents();
    const target = new Date(context.refDate);
    target.setHours(target.getHours() + hours);

    result.assign("year", target.getFullYear());
    result.assign("month", target.getMonth() + 1);
    result.assign("day", target.getDate());
    result.assign("hour", target.getHours());
    result.assign("minute", target.getMinutes());
    result.assign("second", 0);

    return result;
  },
};

export const customChrono = chrono.casual.clone();
customChrono.parsers.unshift(tShorthandParser);
customChrono.parsers.unshift(weekOffsetParser);
customChrono.parsers.unshift(monthOffsetParser);
customChrono.parsers.unshift(relativeHoursParser);

const MMDD_HHMM_REGEX = /^(\d{2})(\d{2})\s+(\d{2})(\d{2})$/;
const MMDD_ONLY_REGEX = /^(\d{2})(\d{2})$/;

/** A calendar date only when it exists: 02/31 is refused rather than rolled into March. */
function realDate(year: number, month: number, day: number, hour = 0, minute = 0): Date | null {
  if (month < 1 || month > 12 || hour > 23 || minute > 59) return null;
  const date = new Date(year, month - 1, day, hour, minute, 0, 0);
  return date.getMonth() === month - 1 && date.getDate() === day ? date : null;
}

function parseCompactDate(text: string, refDate: Date): Date | null {
  const trimmed = text.trim();

  const mmddHhmm = MMDD_HHMM_REGEX.exec(trimmed);
  if (mmddHhmm) {
    return realDate(
      refDate.getFullYear(),
      parseInt(mmddHhmm[1], 10),
      parseInt(mmddHhmm[2], 10),
      parseInt(mmddHhmm[3], 10),
      parseInt(mmddHhmm[4], 10),
    );
  }

  const mmdd = MMDD_ONLY_REGEX.exec(trimmed);
  if (mmdd) {
    return realDate(refDate.getFullYear(), parseInt(mmdd[1], 10), parseInt(mmdd[2], 10));
  }

  return null;
}

/**
 * A date a person typed, read against `refDate` as "now". Pass the reader's wall clock
 * (`userWallClockNow`) so "t" and "t+1" mean today and tomorrow where they are, not where the
 * device thinks it is.
 */
export function parseDate(text: string, refDate: Date = new Date()): Date | null {
  if (!text || !text.trim()) return null;

  const compactResult = parseCompactDate(text, refDate);
  if (compactResult) return compactResult;
  if (MMDD_HHMM_REGEX.test(text.trim()) || MMDD_ONLY_REGEX.test(text.trim())) return null;

  const result = customChrono.parseDate(text, refDate);
  if (!result || isNaN(result.getTime())) return null;
  return result;
}

// Free text a person typed into a date field ("next tue", "3/4"). These live with the
// parser rather than in lib/date, which nearly every screen imports: chrono and its
// locale tables are only read by the screens that take typed dates.
export function generateDateOnly(date: string, refDate?: Date): Date | null {
  if (!date || typeof date !== "string") {
    return null;
  }

  const parsed = parseDate(date, refDate);
  if (parsed && !isNaN(parsed.getTime())) {
    const normalized = new Date(parsed);
    normalized.setHours(0, 0, 0, 0);
    return normalized;
  }
  return null;
}

export function generateDateTime(date: string, refDate?: Date): Date | null {
  if (!date || typeof date !== "string") {
    return null;
  }

  const parsed = parseDate(date, refDate);
  return parsed && !isNaN(parsed.getTime()) ? parsed : null;
}
