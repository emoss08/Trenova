import { describe, expect, it } from "vitest";
import { parseDate } from "../chrono";

function startOfToday(): Date {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return d;
}

function addDays(date: Date, days: number): Date {
  const result = new Date(date);
  result.setDate(result.getDate() + days);
  return result;
}

function addMonths(date: Date, months: number): Date {
  const result = new Date(date);
  result.setMonth(result.getMonth() + months);
  return result;
}

function sameDate(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

describe("t shorthand parser", () => {
  it("parses 't' as today", () => {
    const result = parseDate("t")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, startOfToday())).toBe(true);
  });

  it("parses 't+0' as today", () => {
    const result = parseDate("t+0")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, startOfToday())).toBe(true);
  });

  it("parses 't+1' as tomorrow", () => {
    const result = parseDate("t+1")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 1))).toBe(true);
  });

  it("parses 't+7' as 7 days from now", () => {
    const result = parseDate("t+7")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 7))).toBe(true);
  });

  it("parses 't+30' as 30 days from now", () => {
    const result = parseDate("t+30")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 30))).toBe(true);
  });
});

describe("t-N (past days)", () => {
  it("parses 't-1' as yesterday", () => {
    const result = parseDate("t-1")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), -1))).toBe(true);
  });

  it("parses 't-7' as 7 days ago", () => {
    const result = parseDate("t-7")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), -7))).toBe(true);
  });

  it("parses 't-30' as 30 days ago", () => {
    const result = parseDate("t-30")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), -30))).toBe(true);
  });

  it("parses 't-1 0800' as yesterday at 08:00", () => {
    const result = parseDate("t-1 0800")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), -1))).toBe(true);
    expect(result.getHours()).toBe(8);
    expect(result.getMinutes()).toBe(0);
  });
});

describe("t with time parser", () => {
  it("parses 't 0800' as today at 08:00", () => {
    const result = parseDate("t 0800")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, startOfToday())).toBe(true);
    expect(result.getHours()).toBe(8);
    expect(result.getMinutes()).toBe(0);
  });

  it("parses 't+1 1430' as tomorrow at 14:30", () => {
    const result = parseDate("t+1 1430")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 1))).toBe(true);
    expect(result.getHours()).toBe(14);
    expect(result.getMinutes()).toBe(30);
  });

  it("parses 't 0000' as today at 00:00", () => {
    const result = parseDate("t 0000")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, startOfToday())).toBe(true);
    expect(result.getHours()).toBe(0);
    expect(result.getMinutes()).toBe(0);
  });

  it("parses 't 2359' as today at 23:59", () => {
    const result = parseDate("t 2359")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, startOfToday())).toBe(true);
    expect(result.getHours()).toBe(23);
    expect(result.getMinutes()).toBe(59);
  });
});

describe("MMDD HHMM format", () => {
  it("parses '0318 0800' as March 18 at 08:00", () => {
    const result = parseDate("0318 0800")!;
    expect(result).not.toBeNull();
    expect(result.getMonth()).toBe(2);
    expect(result.getDate()).toBe(18);
    expect(result.getHours()).toBe(8);
    expect(result.getMinutes()).toBe(0);
  });

  it("parses '1225 1430' as December 25 at 14:30", () => {
    const result = parseDate("1225 1430")!;
    expect(result).not.toBeNull();
    expect(result.getMonth()).toBe(11);
    expect(result.getDate()).toBe(25);
    expect(result.getHours()).toBe(14);
    expect(result.getMinutes()).toBe(30);
  });

  it("parses '0101 0000' as January 1 at 00:00", () => {
    const result = parseDate("0101 0000")!;
    expect(result).not.toBeNull();
    expect(result.getMonth()).toBe(0);
    expect(result.getDate()).toBe(1);
    expect(result.getHours()).toBe(0);
    expect(result.getMinutes()).toBe(0);
  });
});

describe("MMDD standalone format", () => {
  it("parses '0318' as March 18", () => {
    const result = parseDate("0318")!;
    expect(result).not.toBeNull();
    expect(result.getMonth()).toBe(2);
    expect(result.getDate()).toBe(18);
    expect(result.getFullYear()).toBe(new Date().getFullYear());
  });

  it("parses '1225' as December 25", () => {
    const result = parseDate("1225")!;
    expect(result).not.toBeNull();
    expect(result.getMonth()).toBe(11);
    expect(result.getDate()).toBe(25);
  });

  it("parses '0101' as January 1", () => {
    const result = parseDate("0101")!;
    expect(result).not.toBeNull();
    expect(result.getMonth()).toBe(0);
    expect(result.getDate()).toBe(1);
  });

  it("returns null for invalid month '1301'", () => {
    expect(parseDate("1301")).toBeNull();
  });

  it("returns null for invalid day '0132'", () => {
    expect(parseDate("0132")).toBeNull();
  });
});

describe("+Nh relative hours", () => {
  it("parses '+2h' as 2 hours from now", () => {
    const now = new Date();
    const result = parseDate("+2h")!;
    expect(result).not.toBeNull();
    const expected = new Date(now);
    expected.setHours(expected.getHours() + 2);
    expect(result.getHours()).toBe(expected.getHours());
    expect(sameDate(result, expected)).toBe(true);
  });

  it("parses '+4h' as 4 hours from now", () => {
    const now = new Date();
    const result = parseDate("+4h")!;
    expect(result).not.toBeNull();
    const expected = new Date(now);
    expected.setHours(expected.getHours() + 4);
    expect(result.getHours()).toBe(expected.getHours());
    expect(sameDate(result, expected)).toBe(true);
  });

  it("parses '+24h' rolling into next day", () => {
    const now = new Date();
    const result = parseDate("+24h")!;
    expect(result).not.toBeNull();
    const expected = new Date(now);
    expected.setHours(expected.getHours() + 24);
    expect(sameDate(result, expected)).toBe(true);
  });
});

describe("w+N / w-N week offsets", () => {
  it("parses 'w+1' as 1 week from now", () => {
    const result = parseDate("w+1")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 7))).toBe(true);
  });

  it("parses 'w+2' as 2 weeks from now", () => {
    const result = parseDate("w+2")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 14))).toBe(true);
  });

  it("parses 'w-1' as 1 week ago", () => {
    const result = parseDate("w-1")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), -7))).toBe(true);
  });

  it("parses 'w-2' as 2 weeks ago", () => {
    const result = parseDate("w-2")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), -14))).toBe(true);
  });

  it("parses 'w+1 0800' as 1 week from now at 08:00", () => {
    const result = parseDate("w+1 0800")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 7))).toBe(true);
    expect(result.getHours()).toBe(8);
    expect(result.getMinutes()).toBe(0);
  });
});

describe("m+N / m-N month offsets", () => {
  it("parses 'm+1' as 1 month from now", () => {
    const result = parseDate("m+1")!;
    expect(result).not.toBeNull();
    const expected = addMonths(startOfToday(), 1);
    expect(sameDate(result, expected)).toBe(true);
  });

  it("parses 'm+3' as 3 months from now", () => {
    const result = parseDate("m+3")!;
    expect(result).not.toBeNull();
    const expected = addMonths(startOfToday(), 3);
    expect(sameDate(result, expected)).toBe(true);
  });

  it("parses 'm-1' as 1 month ago", () => {
    const result = parseDate("m-1")!;
    expect(result).not.toBeNull();
    const expected = addMonths(startOfToday(), -1);
    expect(sameDate(result, expected)).toBe(true);
  });

  it("parses 'm-6' as 6 months ago", () => {
    const result = parseDate("m-6")!;
    expect(result).not.toBeNull();
    const expected = addMonths(startOfToday(), -6);
    expect(sameDate(result, expected)).toBe(true);
  });

  it("parses 'm+1 0900' as 1 month from now at 09:00", () => {
    const result = parseDate("m+1 0900")!;
    expect(result).not.toBeNull();
    const expected = addMonths(startOfToday(), 1);
    expect(sameDate(result, expected)).toBe(true);
    expect(result.getHours()).toBe(9);
    expect(result.getMinutes()).toBe(0);
  });
});

describe("passthrough (existing chrono behavior)", () => {
  it("parses 'tomorrow'", () => {
    const result = parseDate("tomorrow")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 1))).toBe(true);
  });

  it("parses 'next friday'", () => {
    const result = parseDate("next friday");
    expect(result).not.toBeNull();
    expect(result!.getDay()).toBe(5);
  });

  it("parses 'in 3 days'", () => {
    const result = parseDate("in 3 days")!;
    expect(result).not.toBeNull();
    expect(sameDate(result, addDays(startOfToday(), 3))).toBe(true);
  });
});

describe("edge cases", () => {
  it("returns null for empty string", () => {
    expect(parseDate("")).toBeNull();
  });

  it("returns null for whitespace", () => {
    expect(parseDate("   ")).toBeNull();
  });

  it("returns null for gibberish", () => {
    expect(parseDate("xyzzy123abc")).toBeNull();
  });

  it("returns null when chrono produces an invalid date", () => {
    expect(parseDate("not a real date zzz")).toBeNull();
  });
});
