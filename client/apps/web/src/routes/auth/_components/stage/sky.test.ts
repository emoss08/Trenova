import { hexToSrgb, oklchToSrgb, type Oklch } from "@/lib/oklch";
import { describe, expect, it } from "vitest";
import { readSkyTokens, skyAt, SKY_CROSSFADE_HOURS, SKY_KEYFRAMES, type SkyTokens } from "./sky";

// The token values from tokens.css, light and dark.
const LIGHT: SkyTokens = {
  brand: { l: 0.56, c: 0.207, h: 258 },
  violet: { l: 0.55, c: 0.185, h: 306 },
  teal: { l: 0.55, c: 0.093, h: 182 },
  amber: { l: 0.55, c: 0.112, h: 98 },
  rose: { l: 0.55, c: 0.17, h: 2 },
};

const DARK: SkyTokens = {
  brand: { l: 0.717, c: 0.148, h: 258 },
  violet: { l: 0.72, c: 0.157, h: 306 },
  teal: { l: 0.72, c: 0.089, h: 182 },
  amber: { l: 0.72, c: 0.111, h: 98 },
  rose: { l: 0.72, c: 0.145, h: 2 },
};

// The design's precomputed table (design_handoff_sign_in/design/app.jsx, SKY).
const DESIGN = {
  light: {
    night: ["#0046c1", "#7b38b1", "#016dea", "#00776a"],
    dawn: ["#bc3a6a", "#958325", "#8d4bc5", "#016dea"],
    morning: ["#016dea", "#178376", "#8f7d1d", "#bc3a6a"],
    midday: ["#016dea", "#2a8f81", "#3994ff", "#958325"],
    goldenHour: ["#8f7d1d", "#bc3a6a", "#8d4bc5", "#016dea"],
    dusk: ["#8d4bc5", "#b53364", "#0060dc", "#837106"],
  },
  dark: {
    night: ["#518de6", "#b27de5", "#66a4ff", "#56afa1"],
    dawn: ["#ed7a9d", "#c2b058", "#bd88f2", "#66a4ff"],
    morning: ["#66a4ff", "#5eb7a9", "#bfad55", "#ed7a9d"],
    midday: ["#66a4ff", "#66bfb0", "#7cbbff", "#c2b058"],
    goldenHour: ["#bfad55", "#ed7a9d", "#bd88f2", "#66a4ff"],
    dusk: ["#bd88f2", "#e9779a", "#5f9cf7", "#b7a54d"],
  },
} as const;

function channels(hex: string): number[] {
  const rgb = hexToSrgb(hex);
  if (!rgb) {
    throw new Error(`not a hex colour: ${hex}`);
  }
  return rgb.map((channel) => Math.round(channel * 255));
}

/** Within one step per channel: the blend runs on unrounded colours. */
function expectNear(actual: readonly string[], expected: readonly number[][]) {
  actual.forEach((hex, index) => {
    channels(hex).forEach((value, channel) => {
      expect(Math.abs(value - expected[index][channel])).toBeLessThanOrEqual(1);
    });
  });
}

function fromOklch(colors: readonly Oklch[]): number[][] {
  return colors.map((color) => oklchToSrgb(color).map((channel) => channel * 255));
}

describe("skyAt", () => {
  it.each([
    [0, "night", "night"],
    [6, "dawn", "dawn"],
    [9, "morning", "morning"],
    [13, "midday", "midday"],
    [17, "golden-hour", "goldenHour"],
    [19.5, "dusk", "dusk"],
  ] as const)("lands exactly on the %s:00 keyframe in both themes", (hour, phase, key) => {
    const light = skyAt(hour, "light", LIGHT);
    const dark = skyAt(hour, "dark", DARK);

    expect(light.phase).toBe(phase);
    expect(light.stops).toEqual(DESIGN.light[key]);
    expect(dark.phase).toBe(phase);
    expect(dark.stops).toEqual(DESIGN.dark[key]);
  });

  it("scales the lightness offsets by 0.6 in dark, so the two themes differ", () => {
    // Night's outer stop is the brand moved down .12 L; in dark only .072 of that applies.
    expect(skyAt(0, "light", LIGHT).stops[0]).toBe("#0046c1");
    expect(skyAt(0, "dark", DARK).stops[0]).toBe("#518de6");
  });

  it("holds the night across midnight and through the small hours until dawn's crossfade", () => {
    const midnight = skyAt(0, "light", LIGHT).stops;

    for (const hour of [22, 22.5, 23, 23.99, 24, -1, -0.01, 1, 3, 5, 5.25]) {
      expect(skyAt(hour, "light", LIGHT)).toEqual({ phase: "night", stops: midnight });
    }
    expect(skyAt(23.5, "dark", DARK).stops).toEqual(DESIGN.dark.night);
    expect(skyAt(4, "dark", DARK).stops).toEqual(DESIGN.dark.night);
  });

  it("wraps any hour into the day", () => {
    expect(skyAt(30, "light", LIGHT)).toEqual(skyAt(6, "light", LIGHT));
    expect(skyAt(-11, "dark", DARK)).toEqual(skyAt(13, "dark", DARK));
  });

  it("holds each keyframe's palette until the crossfade into the next begins", () => {
    for (const hour of [6, 7, 8, 8.25]) {
      expect(skyAt(hour, "light", LIGHT).stops).toEqual(DESIGN.light.dawn);
    }
    for (const hour of [13, 14, 15, 16, 16.25]) {
      expect(skyAt(hour, "light", LIGHT).stops).toEqual(DESIGN.light.midday);
      expect(skyAt(hour, "dark", DARK).stops).toEqual(DESIGN.dark.midday);
    }
    for (const hour of [19.5, 20.5, 21.25]) {
      expect(skyAt(hour, "light", LIGHT).stops).toEqual(DESIGN.light.dusk);
    }
  });

  it("crosses fully into the next palette over the 45 minutes before its keyframe", () => {
    expect(SKY_CROSSFADE_HOURS).toBe(0.75);
    expect(skyAt(16.5, "light", LIGHT).stops).not.toEqual(DESIGN.light.midday);
    expect(skyAt(16.5, "light", LIGHT).stops).not.toEqual(DESIGN.light.goldenHour);
    expect(skyAt(17, "light", LIGHT).stops).toEqual(DESIGN.light.goldenHour);
  });

  it("mixes in OKLCH at the crossfade's midpoint, turning each hue the shorter way round", () => {
    // 5:37.5 is halfway through night → dawn. Night's outer stop is the brand at
    // L .44 (hue 258) and dawn's is rose (hue 2): the short way runs up through 310.
    expectNear(
      skyAt(5.625, "light", LIGHT).stops,
      fromOklch([
        { l: 0.495, c: 0.1885, h: 310 },
        { l: 0.55, c: 0.1485, h: 22 },
        { l: 0.555, c: 0.196, h: 282 },
        { l: 0.535, c: 0.15, h: 220 },
      ]),
    );
  });

  it("eases along a smoothstep: a quarter of the crossfade moves less than a quarter of the way", () => {
    // 8:26:15 is a quarter of the way through dawn → morning, which starts at 8:15;
    // smoothstep(.25) is .15625. The outer stop turns from rose (2) back to brand (258).
    expectNear(
      [skyAt(8.4375, "light", LIGHT).stops[0]],
      fromOklch([{ l: 0.5515625, c: 0.17578125, h: 345.75 }]),
    );
  });

  it("names the palette the crossfade is nearer to", () => {
    expect(skyAt(3, "light", LIGHT).phase).toBe("night");
    expect(skyAt(5.6, "light", LIGHT).phase).toBe("night");
    expect(skyAt(5.65, "light", LIGHT).phase).toBe("dawn");
    expect(skyAt(8.6, "light", LIGHT).phase).toBe("dawn");
    expect(skyAt(8.65, "light", LIGHT).phase).toBe("morning");
    expect(skyAt(21.6, "light", LIGHT).phase).toBe("dusk");
    expect(skyAt(21.65, "light", LIGHT).phase).toBe("night");
  });

  it("gives every keyframe a hold longer than the crossfade", () => {
    for (let index = 1; index < SKY_KEYFRAMES.length; index += 1) {
      const span = SKY_KEYFRAMES[index].hour - SKY_KEYFRAMES[index - 1].hour;
      expect(span).toBeGreaterThan(SKY_CROSSFADE_HOURS);
    }
  });

  it("keeps the table in hour order, ending on a midnight that repeats the start", () => {
    const hours = SKY_KEYFRAMES.map((keyframe) => keyframe.hour);
    expect(hours).toEqual([...hours].sort((a, b) => a - b));
    expect(SKY_KEYFRAMES.at(-1)?.hour).toBe(24);
    expect(SKY_KEYFRAMES.at(-1)?.stops).toEqual(SKY_KEYFRAMES[0].stops);
  });
});

describe("readSkyTokens", () => {
  function styleOf(values: Record<string, string>) {
    return { getPropertyValue: (property: string) => values[property] ?? "" };
  }

  it("reads each ribbon token from its custom property, as the browser resolves it", () => {
    const tokens = readSkyTokens(
      styleOf({
        "--brand": " oklch(0.56 0.207 258)",
        "--accent-violet": "oklch(0.55 0.185 306)",
        "--accent-teal": "oklch(55% 0.093 182deg)",
        "--accent-amber": "oklch(0.55 0.112 98 / 1)",
        "--accent-rose": "oklch(0.55 0.17 2)",
      }),
    );

    expect(tokens?.brand).toEqual({ l: 0.56, c: 0.207, h: 258 });
    expect(tokens?.teal).toEqual({ l: 0.55, c: 0.093, h: 182 });
    expect(tokens?.amber).toEqual({ l: 0.55, c: 0.112, h: 98 });
  });

  it("is null when a token is missing, rather than drawing a black stop", () => {
    expect(
      readSkyTokens(
        styleOf({
          "--brand": "oklch(0.56 0.207 258)",
          "--accent-violet": "oklch(0.55 0.185 306)",
          "--accent-teal": "oklch(0.55 0.093 182)",
          "--accent-amber": "oklch(0.55 0.112 98)",
        }),
      ),
    ).toBeNull();
  });
});
