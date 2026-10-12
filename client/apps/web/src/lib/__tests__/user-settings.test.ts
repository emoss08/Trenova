import { describe, expect, it } from "vitest";
import {
  daysSince,
  describeSignIn,
  focusChoiceFor,
  focusUntil,
  formatClockValue,
  greetingPeriod,
  halfHourValues,
  hourIn,
  passwordChecks,
  passwordScore,
  pauseUntil,
} from "../user-settings";

const CHICAGO = "America/Chicago";
// Wednesday 14 October 2026, 15:30 in Chicago (CDT, UTC-5).
const WEDNESDAY_AFTERNOON = new Date("2026-10-14T20:30:00Z");

const iso = (seconds: number | null) =>
  seconds === null ? null : new Date(seconds * 1000).toISOString();

describe("pauseUntil", () => {
  it("pauses for an hour from the instant chosen", () => {
    expect(iso(pauseUntil("1h", WEDNESDAY_AFTERNOON, CHICAGO))).toBe("2026-10-14T21:30:00.000Z");
  });

  it("pauses until 8:00 tomorrow in the person's zone, not the browser's", () => {
    expect(iso(pauseUntil("tomorrow", WEDNESDAY_AFTERNOON, CHICAGO))).toBe(
      "2026-10-15T13:00:00.000Z",
    );
    expect(iso(pauseUntil("tomorrow", WEDNESDAY_AFTERNOON, "UTC"))).toBe(
      "2026-10-15T08:00:00.000Z",
    );
  });

  it("pauses until 8:00 next Monday, across a daylight saving change", () => {
    // Chicago leaves daylight saving on 1 November 2026, so Monday the 2nd is UTC-6.
    const friday = new Date("2026-10-30T20:00:00Z");
    expect(iso(pauseUntil("monday", friday, CHICAGO))).toBe("2026-11-02T14:00:00.000Z");
  });

  it("asks for next Monday even when it is Monday", () => {
    const mondayMorning = new Date("2026-10-12T12:00:00Z");
    expect(iso(pauseUntil("monday", mondayMorning, CHICAGO))).toBe("2026-10-19T13:00:00.000Z");
  });

  it("clears a pause", () => {
    expect(pauseUntil("off", WEDNESDAY_AFTERNOON, CHICAGO)).toBeNull();
  });
});

describe("focusUntil and focusChoiceFor", () => {
  it("ends at the end of the person's day", () => {
    expect(iso(focusUntil("eod", WEDNESDAY_AFTERNOON, CHICAGO))).toBe("2026-10-15T04:59:59.000Z");
  });

  it("reads a focus already running back as the nearest choice", () => {
    const now = WEDNESDAY_AFTERNOON;
    const at = (hours: number) => Math.floor(now.getTime() / 1000) + hours * 3600;
    expect(focusChoiceFor(at(0.5), now)).toBe("1h");
    expect(focusChoiceFor(at(1.5), now)).toBe("2h");
    expect(focusChoiceFor(at(6), now)).toBe("eod");
    expect(focusChoiceFor(null, now)).toBe("2h");
  });
});

describe("quiet hours", () => {
  it("offers every half hour of the day", () => {
    const values = halfHourValues();
    expect(values).toHaveLength(48);
    expect(values[0]).toBe("00:00");
    expect(values[1]).toBe("00:30");
    expect(values[38]).toBe("19:00");
    expect(values[47]).toBe("23:30");
  });

  it("reads a stored time in the person's format", () => {
    expect(formatClockValue("19:00", "12-hour")).toBe("7:00 PM");
    expect(formatClockValue("00:30", "12-hour")).toBe("12:30 AM");
    expect(formatClockValue("12:00", "12-hour")).toBe("12:00 PM");
    expect(formatClockValue("07:30", "24-hour")).toBe("07:30");
  });
});

describe("greeting", () => {
  it("follows the hour in the person's zone", () => {
    expect(hourIn(WEDNESDAY_AFTERNOON, CHICAGO)).toBe(15);
    expect(greetingPeriod(3)).toBe("late");
    expect(greetingPeriod(9)).toBe("morning");
    expect(greetingPeriod(15)).toBe("afternoon");
    expect(greetingPeriod(20)).toBe("evening");
  });
});

describe("passwordChecks", () => {
  it("checks each requirement on its own", () => {
    expect(passwordChecks("")).toEqual({
      length: false,
      number: false,
      case: false,
      symbol: false,
    });
    expect(passwordChecks("abcdefghijkl")).toEqual({
      length: true,
      number: false,
      case: false,
      symbol: false,
    });
    expect(passwordChecks("Reefer-Temp-2026")).toEqual({
      length: true,
      number: true,
      case: true,
      symbol: true,
    });
  });

  it("does not count a space as a symbol and counts letters beyond ASCII", () => {
    expect(passwordChecks("ab cd").symbol).toBe(false);
    expect(passwordChecks("Éa").case).toBe(true);
  });

  it("scores the requirements met", () => {
    expect(passwordScore("short1")).toBe(1);
    expect(passwordScore("Reefer-Temp-2026")).toBe(4);
  });
});

describe("daysSince", () => {
  it("counts whole days and never goes negative", () => {
    const now = WEDNESDAY_AFTERNOON;
    const seconds = Math.floor(now.getTime() / 1000);
    expect(daysSince(seconds - 94 * 86400 - 10, now)).toBe(94);
    expect(daysSince(seconds + 3600, now)).toBe(0);
  });
});

describe("describeSignIn", () => {
  it("reads each step of a sign-in the way the server records it", () => {
    expect(describeSignIn({ provider: "password", outcome: "success", mfaState: "" })).toEqual({
      kind: "password",
      tone: "ok",
    });
    expect(
      describeSignIn({ provider: "password", outcome: "success", mfaState: "challenged" }),
    ).toEqual({ kind: "code-requested", tone: "pending" });
    expect(describeSignIn({ provider: "mfa", outcome: "success", mfaState: "totp" })).toEqual({
      kind: "password-and-code",
      tone: "ok",
    });
    expect(
      describeSignIn({ provider: "mfa", outcome: "success", mfaState: "recovery_code" }),
    ).toEqual({ kind: "recovery-code", tone: "ok" });
    expect(describeSignIn({ provider: "mfa", outcome: "failed", mfaState: "rejected" })).toEqual({
      kind: "wrong-code",
      tone: "bad",
    });
  });

  it("tells a wrong password from a refusal, and single sign-on from a password", () => {
    expect(describeSignIn({ provider: "password", outcome: "failed", mfaState: "" }).kind).toBe(
      "wrong-password",
    );
    expect(describeSignIn({ provider: "password", outcome: "denied", mfaState: "" }).kind).toBe(
      "refused",
    );
    expect(describeSignIn({ provider: "sso.okta", outcome: "success", mfaState: "" }).kind).toBe(
      "single-sign-on",
    );
    expect(
      describeSignIn({ provider: "login_throttled", outcome: "denied", mfaState: "" }),
    ).toEqual({ kind: "throttled", tone: "bad" });
  });
});
