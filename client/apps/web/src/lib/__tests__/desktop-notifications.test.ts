import { describe, expect, it } from "vitest";
import { deliveryDecision, shouldShowDesktop } from "../desktop-notifications";

const allowed = {
  data: { delivery: { urgent: false, desktop: true, sound: true, email: "none" } },
  title: "A",
};

describe("deliveryDecision", () => {
  it("reads the server's decision and treats a missing one as bell-only", () => {
    expect(deliveryDecision(allowed)).toEqual({ urgent: false, desktop: true, sound: true });
    expect(
      deliveryDecision({
        data: { delivery: { urgent: true, desktop: false, sound: false, email: "now" } },
      }),
    ).toEqual({ urgent: true, desktop: false, sound: false });
    expect(deliveryDecision({ data: null })).toEqual({
      urgent: false,
      desktop: false,
      sound: false,
    });
    expect(deliveryDecision({ data: { delivery: "yes" } })).toEqual({
      urgent: false,
      desktop: false,
      sound: false,
    });
    expect(deliveryDecision({ data: { delivery: { urgent: "true", desktop: "true" } } })).toEqual({
      urgent: false,
      desktop: false,
      sound: false,
    });
  });
});

describe("shouldShowDesktop", () => {
  it("shows only what the server allowed, with permission, while Trenova is out of sight", () => {
    expect(shouldShowDesktop(allowed, { permission: "granted", visible: false })).toBe(true);
    expect(shouldShowDesktop(allowed, { permission: "granted", visible: true })).toBe(false);
    expect(shouldShowDesktop(allowed, { permission: "default", visible: false })).toBe(false);
    expect(shouldShowDesktop(allowed, { permission: "unsupported", visible: false })).toBe(false);
    expect(
      shouldShowDesktop(
        { ...allowed, data: { delivery: { desktop: false } } },
        { permission: "granted", visible: false },
      ),
    ).toBe(false);
  });
});
