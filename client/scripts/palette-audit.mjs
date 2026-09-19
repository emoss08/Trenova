/**
 * Reads the oklch values straight out of tokens.css and checks the things a
 * palette can get wrong without anything failing.
 *
 * A colour decision is invisible to a type checker, a test and a build. The
 * warning tone shipped for months as oklch(0.75 0.16 70) — a bright amber that
 * drew near-white text on a solid badge at 2.1:1 — and nothing said so. So the
 * pairs the design actually relies on are asserted here:
 *
 *   - body, secondary and tertiary text clear AA on every surface they land on
 *   - each tone's subtle badge clears AA
 *   - the ink on each solid fill clears AA against that fill
 *   - hairlines are actually visible against the surface they divide
 *   - every value is inside sRGB, so a browser does not gamut-map it somewhere
 *     other than where it was placed
 *   - danger, brand and warning stay far enough apart in hue that a primary
 *     action cannot be mistaken for a destructive one
 *
 * There is no build step: tokens.css stays the file people edit, and this reads
 * what they wrote.
 */

/* ------------------------------------------------- sRGB <-> OKLCH, contrast */

const srgbToLin = (c) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);
const linToSrgb = (c) => (c <= 0.0031308 ? c * 12.92 : 1.055 * c ** (1 / 2.4) - 0.055);

function oklchToRgb([L, C, H]) {
  const h = (H * Math.PI) / 180;
  const a = C * Math.cos(h);
  const b = C * Math.sin(h);
  const l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s = (L - 0.0894841775 * a - 1.291485548 * b) ** 3;
  return [
    4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
    -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
    -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s,
  ].map(linToSrgb);
}

const inGamut = (lch) => oklchToRgb(lch).every((v) => v >= -0.0015 && v <= 1.0015);

const hex = (lch) =>
  "#" +
  oklchToRgb(lch)
    .map((v) => Math.round(Math.max(0, Math.min(1, v)) * 255).toString(16).padStart(2, "0"))
    .join("");

const relLum = ([r, g, b]) =>
  0.2126 * srgbToLin(Math.max(0, Math.min(1, r))) +
  0.7152 * srgbToLin(Math.max(0, Math.min(1, g))) +
  0.0722 * srgbToLin(Math.max(0, Math.min(1, b)));

function contrast(a, b) {
  const [x, y] = [relLum(oklchToRgb(a)), relLum(oklchToRgb(b))];
  const [hi, lo] = x > y ? [x, y] : [y, x];
  return (hi + 0.05) / (lo + 0.05);
}

/* ------------------------------------------------------------------ parsing */

/**
 * Pulls `--name: oklch(L C H)` out of one block, resolving `var(--hue-*)` in
 * the hue slot. Values carrying an alpha channel are scrims and overlays, which
 * are composited rather than read against a fixed ground, so they are skipped.
 */
function parseBlock(block, hues) {
  const out = {};
  const re = /^\s*(--[a-z0-9-]+):\s*oklch\(\s*([\d.]+)\s+([\d.]+)\s+(var\(--[a-z-]+\)|[\d.]+)\s*\)\s*;/gm;
  for (const m of block.matchAll(re)) {
    const raw = m[4];
    const hue = raw.startsWith("var(")
      ? hues[raw.slice(4, -1)]
      : Number(raw);
    if (hue === undefined || Number.isNaN(hue)) continue;
    out[m[1].slice(2)] = [Number(m[2]), Number(m[3]), hue];
  }
  return out;
}

export function parseTokens(src) {
  const stripped = src.replace(/\/\*[\s\S]*?\*\//g, "");
  const hues = Object.fromEntries(
    [...stripped.matchAll(/^\s*(--hue-[a-z]+):\s*([\d.]+)\s*;/gm)].map((m) => [m[1], Number(m[2])]),
  );
  const darkAt = stripped.indexOf(".dark {");
  if (darkAt === -1) return null;
  return {
    hues,
    light: parseBlock(stripped.slice(0, darkAt), hues),
    dark: parseBlock(stripped.slice(darkAt), hues),
  };
}

/* -------------------------------------------------------------------- audit */

const TONES = ["danger", "warning", "success", "info", "neutral"];
const ACCENTS = ["indigo", "teal", "amber", "rose", "emerald", "sky", "violet", "slate"];

function auditTheme(theme, p) {
  const out = [];
  const has = (k) => p[k] !== undefined;
  const pair = (fg, bg, min, why) => {
    if (!has(fg) || !has(bg)) return;
    const r = contrast(p[fg], p[bg]);
    if (r < min) {
      out.push(
        `${theme}: ${why} — --${fg} (${hex(p[fg])}) on --${bg} (${hex(p[bg])}) is ${r.toFixed(2)}:1, needs ${min}:1.`,
      );
    }
  };

  pair("foreground", "canvas", 7, "body text on the page ground");
  pair("foreground", "card", 7, "body text on a panel");
  pair("foreground", "sunken", 7, "body text in a well");
  pair("foreground-muted", "card", 4.5, "secondary text");
  pair("foreground-subtle", "card", 4.5, "tertiary text on a panel");
  pair("foreground-subtle", "canvas", 4.5, "tertiary text on the page ground");
  pair("foreground-subtle", "sunken", 4.5, "column headers");

  for (const t of [...TONES, "brand"]) {
    pair(`${t}-subtle-foreground`, `${t}-subtle`, 4.5, `the ${t} soft badge`);
    // --brand-foreground is the ink ON the brand fill, not brand-coloured text.
    if (t !== "brand") pair(`${t}-foreground`, "card", 4.5, `${t} text on a panel`);
    const ink = t === "brand" ? "brand-foreground" : "foreground-on-solid";
    pair(ink, t, 4.5, `the solid ${t} fill`);
  }

  for (const a of ACCENTS) {
    pair(`accent-${a}-on-subtle`, `accent-${a}-subtle`, 4.5, `the ${a} category chip`);
  }

  pair("highlight-foreground", "highlight", 4.5, "selected text");

  // Not WCAG: a hairline nobody can see is how the surface ladder collapsed
  // into one flat plane the first time round.
  for (const [line, surface, floor] of [
    ["border", "card", 1.2],
    ["border", "canvas", 1.2],
    ["border-subtle", "card", 1.14],
    ["border-strong", "card", 1.2],
  ]) {
    if (!has(line) || !has(surface)) continue;
    const r = contrast(p[line], p[surface]);
    if (r < floor) {
      out.push(
        `${theme}: --${line} is invisible against --${surface} (${r.toFixed(2)}:1, needs ${floor}:1). A divider nobody can see is not a divider.`,
      );
    }
  }

  for (const [name, v] of Object.entries(p)) {
    if (!inGamut(v)) {
      out.push(
        `${theme}: --${name} is outside sRGB, so the browser will not show where it was placed — it lands near ${hex(v)}. Lower its chroma.`,
      );
    }
  }
  return out;
}

/**
 * Copper sits between red and amber, so four things crowd the same arc: the
 * destructive action, the primary action, the warning tone and the amber
 * category. Without a floor they converge into one orange and the colour stops
 * carrying the difference.
 */
function auditHueSeparation(p, hues) {
  const MIN = 20;
  const arc = [
    ["danger", p.danger?.[2]],
    ["brand", hues["--hue-brand"]],
    ["warning", p.warning?.[2]],
    ["the amber category accent", hues["--hue-amber"]],
  ];
  const out = [];
  for (let i = 1; i < arc.length; i++) {
    const [a, ah] = arc[i - 1];
    const [b, bh] = arc[i];
    if (ah === undefined || bh === undefined) continue;
    const gap = Math.abs(ah - bh);
    if (gap < MIN) {
      out.push(
        `hues: ${a} (${ah}°) and ${b} (${bh}°) are ${gap}° apart, under the ${MIN}° floor. At that distance they read as the same orange, and the colour stops telling them apart.`,
      );
    }
  }
  return out;
}

export function auditPalette(src) {
  const parsed = parseTokens(src);
  if (!parsed) return ["tokens.css has no .dark block, so the palette could not be read."];
  const { light, dark, hues } = parsed;
  if (Object.keys(light).length < 40) {
    return ["tokens.css parsed into fewer than 40 light tokens, which means the palette did not read."];
  }
  return [
    ...auditTheme("light", light),
    ...auditTheme("dark", { ...light, ...dark }),
    ...auditHueSeparation(light, hues),
  ];
}
