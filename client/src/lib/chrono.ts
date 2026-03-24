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

function parseCompactDate(text: string): Date | null {
  const trimmed = text.trim();

  const mmddHhmm = MMDD_HHMM_REGEX.exec(trimmed);
  if (mmddHhmm) {
    const month = parseInt(mmddHhmm[1], 10);
    const day = parseInt(mmddHhmm[2], 10);
    const hour = parseInt(mmddHhmm[3], 10);
    const minute = parseInt(mmddHhmm[4], 10);

    if (month < 1 || month > 12 || day < 1 || day > 31) return null;
    if (hour > 23 || minute > 59) return null;

    const now = new Date();
    return new Date(now.getFullYear(), month - 1, day, hour, minute, 0, 0);
  }

  const mmdd = MMDD_ONLY_REGEX.exec(trimmed);
  if (mmdd) {
    const month = parseInt(mmdd[1], 10);
    const day = parseInt(mmdd[2], 10);

    if (month < 1 || month > 12 || day < 1 || day > 31) return null;

    const now = new Date();
    return new Date(now.getFullYear(), month - 1, day, 0, 0, 0, 0);
  }

  return null;
}

export function parseDate(text: string): Date | null {
  if (!text || !text.trim()) return null;

  const compactResult = parseCompactDate(text);
  if (compactResult) return compactResult;

  const result = customChrono.parseDate(text);
  if (!result || isNaN(result.getTime())) return null;
  return result;
}
