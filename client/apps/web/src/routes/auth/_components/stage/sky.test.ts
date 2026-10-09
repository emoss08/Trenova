import { hexToSrgb } from "@/lib/oklch";
import { describe, expect, it } from "vitest";
import { readSkyTokens, skyAt, SKY_KEYFRAMES, type SkyTokens } from "./sky";

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

function midpoint(from: readonly string[], to: readonly string[]): number[][] {
  return from.map((hex, index) => {
    const a = channels(hex);
    const b = channels(to[index]);
    return a.map((value, channel) => (value + b[channel]) / 2);
  });
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

  it("holds the night across midnight: 22:00 → 24:00 → 0:00 is one palette", () => {
    const midnight = skyAt(0, "light", LIGHT).stops;

    for (const hour of [22, 22.5, 23, 23.99, 24, -1, -0.01]) {
      expect(skyAt(hour, "light", LIGHT)).toEqual({ phase: "night", stops: midnight });
    }
    expect(skyAt(23.5, "dark", DARK).stops).toEqual(DESIGN.dark.night);
  });

  it("wraps any hour into the day", () => {
    expect(skyAt(30, "light", LIGHT)).toEqual(skyAt(6, "light", LIGHT));
    expect(skyAt(-11, "dark", DARK)).toEqual(skyAt(13, "dark", DARK));
  });

  it("is halfway between two keyframes at the midpoint of their span", () => {
    expectNear(
      skyAt(7.5, "light", LIGHT).stops,
      midpoint(DESIGN.light.dawn, DESIGN.light.morning),
    );
    expectNear(
      skyAt(15, "dark", DARK).stops,
      midpoint(DESIGN.dark.midday, DESIGN.dark.goldenHour),
    );
    expectNear(
      skyAt(20.75, "light", LIGHT).stops,
      midpoint(DESIGN.light.dusk, DESIGN.light.night),
    );
  });

  it("eases along a smoothstep: a quarter of the span moves less than a quarter of the way", () => {
    // 6:45 is a quarter of dawn → morning; smoothstep(.25) is .15625.
    const quarter = skyAt(6.75, "light", LIGHT).stops;
    const dawn = DESIGN.light.dawn.map(channels);
    const morning = DESIGN.light.morning.map(channels);
    const expected = dawn.map((from, index) =>
      from.map((value, channel) => value + (morning[index][channel] - value) * 0.15625),
    );
    expectNear(quarter, expected);
  });

  it("names the nearer keyframe's phase", () => {
    expect(skyAt(7.4, "light", LIGHT).phase).toBe("dawn");
    expect(skyAt(7.6, "light", LIGHT).phase).toBe("morning");
    expect(skyAt(21.9, "light", LIGHT).phase).toBe("night");
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
