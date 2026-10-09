/** A colour in OKLCH: lightness 0–1, chroma, hue in degrees. */
export type Oklch = { l: number; c: number; h: number };

/** Gamma-encoded sRGB, each channel 0–1. */
export type Srgb = readonly [number, number, number];

const NUMBER = String.raw`[+-]?(?:\d+\.?\d*|\.\d+)(?:e[+-]?\d+)?`;
const OKLCH_PATTERN = new RegExp(
  String.raw`^oklch\(\s*(${NUMBER}%?|none)\s+(${NUMBER}%?|none)\s+(${NUMBER}(?:deg)?|none)\s*(?:\/\s*[^)]*)?\)$`,
  "i",
);
const RGB_PATTERN = new RegExp(
  String.raw`^rgba?\(\s*(${NUMBER})[\s,]+(${NUMBER})[\s,]+(${NUMBER})\s*(?:[,/]\s*[^)]*)?\)$`,
  "i",
);
const HEX_PATTERN = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i;

/** OKLCH's chroma axis: 100% is 0.4. */
const CHROMA_PERCENT_SCALE = 0.004;

function component(raw: string, percentScale: number): number {
  if (raw.toLowerCase() === "none") {
    return 0;
  }
  if (raw.endsWith("%")) {
    return Number.parseFloat(raw) * percentScale;
  }
  return Number.parseFloat(raw);
}

function clamp01(value: number): number {
  return Math.min(1, Math.max(0, value));
}

function encodeChannel(linear: number): number {
  const value = clamp01(linear);
  return value <= 0.0031308 ? value * 12.92 : 1.055 * value ** (1 / 2.4) - 0.055;
}

function decodeChannel(encoded: number): number {
  return encoded <= 0.04045 ? encoded / 12.92 : ((encoded + 0.055) / 1.055) ** 2.4;
}

export function oklchToSrgb({ l, c, h }: Oklch): Srgb {
  const radians = (h * Math.PI) / 180;
  const a = c * Math.cos(radians);
  const b = c * Math.sin(radians);

  const lp = l + 0.3963377774 * a + 0.2158037573 * b;
  const mp = l - 0.1055613458 * a - 0.0638541728 * b;
  const sp = l - 0.0894841775 * a - 1.291485548 * b;
  const lc = lp * lp * lp;
  const mc = mp * mp * mp;
  const sc = sp * sp * sp;

  return [
    encodeChannel(4.0767416621 * lc - 3.3077115913 * mc + 0.2309699292 * sc),
    encodeChannel(-1.2684380046 * lc + 2.6097574011 * mc - 0.3413193965 * sc),
    encodeChannel(-0.0041960863 * lc - 0.7034186147 * mc + 1.707614701 * sc),
  ];
}

export function srgbToOklch([red, green, blue]: Srgb): Oklch {
  const r = decodeChannel(red);
  const g = decodeChannel(green);
  const b = decodeChannel(blue);

  const lc = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
  const mc = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
  const sc = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);

  const l = 0.2104542553 * lc + 0.793617785 * mc - 0.0040720468 * sc;
  const a = 1.9779984951 * lc - 2.428592205 * mc + 0.4505937099 * sc;
  const bb = 0.0259040371 * lc + 0.7827717662 * mc - 0.808675766 * sc;
  const hue = (Math.atan2(bb, a) * 180) / Math.PI;

  return { l, c: Math.hypot(a, bb), h: hue < 0 ? hue + 360 : hue };
}

export function hexToSrgb(hex: string): Srgb | null {
  const match = HEX_PATTERN.exec(hex.trim());
  if (!match) {
    return null;
  }
  const digits =
    match[1].length === 3
      ? match[1]
          .split("")
          .map((digit) => digit + digit)
          .join("")
      : match[1];
  const value = Number.parseInt(digits, 16);
  return [((value >> 16) & 255) / 255, ((value >> 8) & 255) / 255, (value & 255) / 255];
}

export function srgbToHex(color: Srgb): string {
  return `#${color
    .map((channel) =>
      Math.round(clamp01(channel) * 255)
        .toString(16)
        .padStart(2, "0"),
    )
    .join("")}`;
}

/** Mixes two gamma-encoded colours channel by channel; `t` 0 is `from`, 1 is `to`. */
export function mixSrgb(from: Srgb, to: Srgb, t: number): Srgb {
  return [
    from[0] + (to[0] - from[0]) * t,
    from[1] + (to[1] - from[1]) * t,
    from[2] + (to[2] - from[2]) * t,
  ];
}

/**
 * Reads a resolved CSS colour — what `getComputedStyle` returns for a custom property
 * once its `var()` references are substituted — in the forms the design tokens use:
 * `oklch()`, `#rgb`/`#rrggbb` and `rgb()`. Anything else is null.
 */
export function parseCssColor(value: string): Oklch | null {
  const input = value.trim();

  const oklch = OKLCH_PATTERN.exec(input);
  if (oklch) {
    return {
      l: component(oklch[1], 0.01),
      c: component(oklch[2], CHROMA_PERCENT_SCALE),
      h: oklch[3].toLowerCase() === "none" ? 0 : Number.parseFloat(oklch[3]),
    };
  }

  const hex = hexToSrgb(input);
  if (hex) {
    return srgbToOklch(hex);
  }

  const rgb = RGB_PATTERN.exec(input);
  if (rgb) {
    return srgbToOklch([
      Number.parseFloat(rgb[1]) / 255,
      Number.parseFloat(rgb[2]) / 255,
      Number.parseFloat(rgb[3]) / 255,
    ]);
  }

  return null;
}
