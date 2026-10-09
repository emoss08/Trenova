import { describe, expect, it } from "vitest";
import {
  hexToSrgb,
  mixSrgb,
  oklchToSrgb,
  parseCssColor,
  srgbToHex,
  srgbToOklch,
} from "../oklch";

describe("oklchToSrgb", () => {
  it("matches the design's precomputed token colours", () => {
    expect(srgbToHex(oklchToSrgb({ l: 0.56, c: 0.207, h: 258 }))).toBe("#016dea");
    expect(srgbToHex(oklchToSrgb({ l: 0.717, c: 0.148, h: 258 }))).toBe("#66a4ff");
    expect(srgbToHex(oklchToSrgb({ l: 0.55, c: 0.17, h: 2 }))).toBe("#bc3a6a");
  });

  it("maps the neutral ends to black and white", () => {
    expect(srgbToHex(oklchToSrgb({ l: 0, c: 0, h: 0 }))).toBe("#000000");
    expect(srgbToHex(oklchToSrgb({ l: 1, c: 0, h: 0 }))).toBe("#ffffff");
  });

  it("clips a colour outside sRGB instead of overflowing a channel", () => {
    const [r, g, b] = oklchToSrgb({ l: 0.9, c: 0.4, h: 140 });
    for (const channel of [r, g, b]) {
      expect(channel).toBeGreaterThanOrEqual(0);
      expect(channel).toBeLessThanOrEqual(1);
    }
  });
});

describe("srgbToOklch", () => {
  it("round-trips a token colour", () => {
    const back = srgbToOklch(oklchToSrgb({ l: 0.56, c: 0.207, h: 258 }));
    expect(back.l).toBeCloseTo(0.56, 3);
    expect(back.c).toBeCloseTo(0.207, 3);
    expect(back.h).toBeCloseTo(258, 1);
  });
});

describe("parseCssColor", () => {
  it.each([
    ["oklch(0.56 0.207 258)", { l: 0.56, c: 0.207, h: 258 }],
    ["  oklch(56% 0.207 258deg) ", { l: 0.56, c: 0.207, h: 258 }],
    ["oklch(0.985 0 0)", { l: 0.985, c: 0, h: 0 }],
    ["oklch(0.145 0 none)", { l: 0.145, c: 0, h: 0 }],
    ["oklch(0 0 0 / 0.4)", { l: 0, c: 0, h: 0 }],
    ["OKLCH(0.5 50% 10)", { l: 0.5, c: 0.2, h: 10 }],
  ])("reads %j", (input, expected) => {
    const parsed = parseCssColor(input);
    expect(parsed?.l).toBeCloseTo(expected.l, 6);
    expect(parsed?.c).toBeCloseTo(expected.c, 6);
    expect(parsed?.h).toBeCloseTo(expected.h, 6);
  });

  it("reads hex and rgb() through sRGB", () => {
    expect(parseCssColor("#ffffff")?.l).toBeCloseTo(1, 3);
    expect(parseCssColor("#000")?.l).toBeCloseTo(0, 3);
    expect(parseCssColor("rgb(255 255 255)")?.l).toBeCloseTo(1, 3);
    expect(parseCssColor("rgba(0, 0, 0, 0.5)")?.l).toBeCloseTo(0, 3);
  });

  it.each(["", "var(--brand)", "color-mix(in oklch, red, blue)", "oklch(0.5 0.1)", "#12345"])(
    "is null for %j",
    (input) => {
      expect(parseCssColor(input)).toBeNull();
    },
  );
});

describe("hex helpers", () => {
  it("expands short hex and round-trips long hex", () => {
    expect(srgbToHex(hexToSrgb("#0af") ?? [0, 0, 0])).toBe("#00aaff");
    expect(srgbToHex(hexToSrgb("#016dea") ?? [0, 0, 0])).toBe("#016dea");
  });

  it("mixes channel by channel", () => {
    expect(srgbToHex(mixSrgb([0, 0, 0], [1, 1, 1], 0.5))).toBe("#808080");
    expect(mixSrgb([0.2, 0.4, 0.6], [0.4, 0.4, 0.4], 0)).toEqual([0.2, 0.4, 0.6]);
  });
});
