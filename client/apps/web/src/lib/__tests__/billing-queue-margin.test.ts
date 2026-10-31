import { describe, expect, it } from "vitest";
import { shipmentMargin } from "../billing-queue-margin";

const t = ((message: string, ...args: unknown[]) =>
  args.reduce<string>(
    (text, arg, index) => text.replace(`{${index}}`, String(arg)),
    message,
  )) as never;

describe("shipmentMargin", () => {
  it("says the margin in dollars and a whole percent", () => {
    expect(shipmentMargin({ profit: "1174.62", marginPercent: "31.04" }, t)).toEqual({
      text: "$1,174.62 · 31%",
      thin: false,
    });
  });

  it("calls a margin under the profile's target thin, or under 10% without one", () => {
    expect(
      shipmentMargin(
        { profit: "300", marginPercent: "18", profile: { targetMarginPercent: "20" } },
        t,
      )?.thin,
    ).toBe(true);
    expect(shipmentMargin({ profit: "300", marginPercent: "18" }, t)?.thin).toBe(false);
    expect(shipmentMargin({ profit: "40", marginPercent: "8" }, t)?.thin).toBe(true);
  });

  it("shows a loss as a negative amount", () => {
    expect(shipmentMargin({ profit: "-212.5", marginPercent: "-6.2" }, t)?.text).toBe(
      "-$212.50 · -6%",
    );
  });

  it("shows nothing while the margin is not known", () => {
    expect(shipmentMargin(undefined, t)).toBeNull();
    expect(shipmentMargin({ profit: "300", marginPercent: null }, t)).toBeNull();
  });
});
