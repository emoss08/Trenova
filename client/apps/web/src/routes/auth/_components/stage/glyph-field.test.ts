import { describe, expect, it } from "vitest";
import { glyphFieldSettled, type GlyphFieldMotion } from "./glyph-field";

/** A field that has drawn in, with the pointer off the stage and nothing easing. */
const STILL: GlyphFieldMotion = {
  intro: 1,
  pulse: 1,
  done: 0,
  doneTarget: 0,
  hover: 0,
  hoverTarget: 0,
  mouse: [-999, -999],
  smoothedMouse: [-999, -999],
  palette: [
    [0.1, 0.2, 0.3],
    [0.4, 0.5, 0.6],
  ],
  paletteTarget: [
    [0.1, 0.2, 0.3],
    [0.4, 0.5, 0.6],
  ],
};

describe("glyphFieldSettled", () => {
  it("is settled when nothing is left to ease, so the loop stops", () => {
    expect(glyphFieldSettled(STILL)).toBe(true);
  });

  it.each<[string, Partial<GlyphFieldMotion>]>([
    ["the intro is still drawing in", { intro: 0.6 }],
    ["a burst is still crossing", { pulse: 0.4 }],
    ["the success ribbon is still widening", { done: 0.5, doneTarget: 1 }],
    ["the glow is still fading in", { hover: 0.3, hoverTarget: 1 }],
    ["the glow has not caught the pointer", { mouse: [400, 300], smoothedMouse: [380, 300] }],
    [
      "a new palette is still easing in",
      {
        paletteTarget: [
          [0.1, 0.2, 0.3],
          [0.9, 0.5, 0.6],
        ],
      },
    ],
  ])("keeps running while %s", (_, change) => {
    expect(glyphFieldSettled({ ...STILL, ...change })).toBe(false);
  });

  it("counts values within a hair of their targets as arrived", () => {
    expect(
      glyphFieldSettled({
        ...STILL,
        done: 0.9995,
        doneTarget: 1,
        hover: 0.0004,
        mouse: [400, 300],
        smoothedMouse: [399.7, 300.2],
      }),
    ).toBe(true);
  });
});
