import { describe, expect, it } from "vitest";
import { deliveryDecision, shouldShowDesktop } from "../desktop-notifications";

const allowed = { data: { delivery: { desktop: true, sound: true, email: "none" } }, title: "A" };

describe("deliveryDecision", () => {
  it("reads the server's decision and treats a missing one as bell-only", () => {
    expect(deliveryDecision(allowed)).toEqual({ desktop: true, sound: true });
    expect(deliveryDecision({ data: null })).toEqual({ desktop: false, sound: false });
    expect(deliveryDecision({ data: { delivery: "yes" } })).toEqual({
      desktop: false,
      sound: false,
    });
    expect(deliveryDecision({ data: { delivery: { desktop: "true" } } })).toEqual({
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
