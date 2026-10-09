import {
  mixSrgb,
  oklchToSrgb,
  parseCssColor,
  srgbToHex,
  type Oklch,
  type Srgb,
} from "@/lib/oklch";

export type SkyTheme = "light" | "dark";

export type SkyPhase = "night" | "dawn" | "morning" | "midday" | "golden-hour" | "dusk";

/** The tokens the ribbon is drawn from: the brand and four categorical accents. */
export type SkyToken = "brand" | "violet" | "teal" | "amber" | "rose";

/** A token and the OKLCH lightness offset applied to it. */
export type SkyStop = readonly [token: SkyToken, deltaL: number];

export type SkyStops = readonly [SkyStop, SkyStop, SkyStop, SkyStop];

export type SkyKeyframe = { hour: number; phase: SkyPhase; stops: SkyStops };

export type SkyTokens = Readonly<Record<SkyToken, Oklch>>;

export type SkyFrame = {
  phase: SkyPhase;
  /** The ribbon's four stops, outer to inner, as `#rrggbb`. */
  stops: [string, string, string, string];
};

const NIGHT: SkyStops = [
  ["brand", -0.12],
  ["violet", -0.06],
  ["brand", 0],
  ["teal", -0.04],
];

/**
 * The ribbon's palette by local hour. A keyframe's stops hold from its hour and blend
 * into the next keyframe's; 22:00 and 24:00 repeat midnight so the night holds across
 * the wrap rather than blending back through dusk.
 */
export const SKY_KEYFRAMES: readonly SkyKeyframe[] = [
  { hour: 0, phase: "night", stops: NIGHT },
  {
    hour: 6,
    phase: "dawn",
    stops: [
      ["rose", 0],
      ["amber", 0.06],
      ["violet", 0],
      ["brand", 0],
    ],
  },
  {
    hour: 9,
    phase: "morning",
    stops: [
      ["brand", 0],
      ["teal", 0],
      ["amber", 0.04],
      ["rose", 0],
    ],
  },
  {
    hour: 13,
    phase: "midday",
    stops: [
      ["brand", 0],
      ["teal", 0.04],
      ["brand", 0.12],
      ["amber", 0.06],
    ],
  },
  {
    hour: 17,
    phase: "golden-hour",
    stops: [
      ["amber", 0.04],
      ["rose", 0],
      ["violet", 0],
      ["brand", 0],
    ],
  },
  {
    hour: 19.5,
    phase: "dusk",
    stops: [
      ["violet", 0],
      ["rose", -0.02],
      ["brand", -0.04],
      ["amber", 0],
    ],
  },
  { hour: 22, phase: "night", stops: NIGHT },
  { hour: 24, phase: "night", stops: NIGHT },
];

/** Dark tokens already sit high on the lightness axis, so offsets move them less. */
const DARK_OFFSET_SCALE = 0.6;

/** The CSS custom property each ribbon token is read from. */
export const SKY_TOKEN_PROPERTIES: Readonly<Record<SkyToken, string>> = {
  brand: "--brand",
  violet: "--accent-violet",
  teal: "--accent-teal",
  amber: "--accent-amber",
  rose: "--accent-rose",
};

function smoothstep(t: number): number {
  return t * t * (3 - 2 * t);
}

function wrapHour(hour: number): number {
  return ((hour % 24) + 24) % 24;
}

function resolveStop([token, deltaL]: SkyStop, theme: SkyTheme, tokens: SkyTokens): Srgb {
  const base = tokens[token];
  const offset = theme === "dark" ? deltaL * DARK_OFFSET_SCALE : deltaL;
  return oklchToSrgb({ l: Math.min(1, Math.max(0, base.l + offset)), c: base.c, h: base.h });
}

function keyframeSpan(hour: number): [SkyKeyframe, SkyKeyframe] {
  let index = 0;
  while (index < SKY_KEYFRAMES.length - 2 && SKY_KEYFRAMES[index + 1].hour <= hour) {
    index += 1;
  }
  return [SKY_KEYFRAMES[index], SKY_KEYFRAMES[index + 1]];
}

/**
 * The ribbon's palette at `hour` (local, fractional; any value wraps into 0–24). Each
 * stop is its token's own OKLCH colour moved by the keyframe's lightness offset, and
 * consecutive keyframes are blended in sRGB along a smoothstep, so the colour lingers
 * at a keyframe and moves fastest between two.
 */
export function skyAt(hour: number, theme: SkyTheme, tokens: SkyTokens): SkyFrame {
  const wrapped = wrapHour(hour);
  const [from, to] = keyframeSpan(wrapped);
  const progress = Math.min(1, Math.max(0, (wrapped - from.hour) / (to.hour - from.hour)));
  const eased = smoothstep(progress);

  const stops = from.stops.map((stop, index) =>
    srgbToHex(
      mixSrgb(
        resolveStop(stop, theme, tokens),
        resolveStop(to.stops[index], theme, tokens),
        eased,
      ),
    ),
  ) as SkyFrame["stops"];

  return { phase: progress < 0.5 ? from.phase : to.phase, stops };
}

/**
 * Reads the ribbon tokens from resolved styles — normally
 * `getComputedStyle(document.documentElement)`, so the result follows the active theme.
 * Null when any of them is missing or in a form the parser does not read.
 */
export function readSkyTokens(style: Pick<CSSStyleDeclaration, "getPropertyValue">): SkyTokens | null {
  const tokens: Partial<Record<SkyToken, Oklch>> = {};
  for (const [token, property] of Object.entries(SKY_TOKEN_PROPERTIES) as [SkyToken, string][]) {
    const color = parseCssColor(style.getPropertyValue(property));
    if (!color) {
      return null;
    }
    tokens[token] = color;
  }
  return tokens as SkyTokens;
}
