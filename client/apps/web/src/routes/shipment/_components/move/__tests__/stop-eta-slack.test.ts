import { describe, expect, it } from "vitest";
import { stopEtaSlackText } from "../stop-eta";

const t = ((template: string, ...args: string[]) =>
  args.reduce((text, arg, i) => text.replace(`{${i}}`, arg), template)) as Parameters<
  typeof stopEtaSlackText
>[1];

describe("stopEtaSlackText", () => {
  it("says how much time is left before the window closes", () => {
    expect(stopEtaSlackText(95, t)).toBe("1h 35m to spare");
  });

  it("says how far past the window a projection lands", () => {
    expect(stopEtaSlackText(-40, t)).toBe("40m past the window");
  });

  it("treats exactly on the window end as no time to spare, not late", () => {
    expect(stopEtaSlackText(0, t)).toBe("0m to spare");
  });

  it("says nothing when no slack was projected", () => {
    expect(stopEtaSlackText(null, t)).toBeNull();
    expect(stopEtaSlackText(undefined, t)).toBeNull();
  });
});
