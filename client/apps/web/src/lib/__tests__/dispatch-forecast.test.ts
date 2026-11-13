import {
  createRefitPoll,
  FORECAST_REFIT_POLL_FOR_MS,
  FORECAST_REFIT_POLL_MS,
} from "@/lib/dispatch-forecast";
import { describe, expect, it } from "vitest";

describe("createRefitPoll", () => {
  it("keeps the page's own pace while nothing is being refitted", () => {
    expect(createRefitPoll(300_000)(false)).toBe(300_000);
    expect(createRefitPoll(false)(false)).toBe(false);
    expect(createRefitPoll(false)(undefined)).toBe(false);
  });

  it("asks quickly while a refit runs, then returns to the page's pace when it lands", () => {
    let now = 1_000;
    const poll = createRefitPoll(300_000, () => now);

    expect(poll(true)).toBe(FORECAST_REFIT_POLL_MS);
    now += FORECAST_REFIT_POLL_MS;
    expect(poll(true)).toBe(FORECAST_REFIT_POLL_MS);
    expect(poll(false)).toBe(300_000);
  });

  it("stops asking quickly for a refit that has not landed after two minutes", () => {
    let now = 1_000;
    const poll = createRefitPoll(false, () => now);

    expect(poll(true)).toBe(FORECAST_REFIT_POLL_MS);
    now += FORECAST_REFIT_POLL_FOR_MS - 1;
    expect(poll(true)).toBe(FORECAST_REFIT_POLL_MS);
    now += 1;
    expect(poll(true)).toBe(false);
  });

  it("gives a later refit its own two minutes", () => {
    let now = 1_000;
    const poll = createRefitPoll(false, () => now);

    expect(poll(true)).toBe(FORECAST_REFIT_POLL_MS);
    now += FORECAST_REFIT_POLL_FOR_MS;
    expect(poll(true)).toBe(false);
    expect(poll(false)).toBe(false);
    expect(poll(true)).toBe(FORECAST_REFIT_POLL_MS);
  });
});
