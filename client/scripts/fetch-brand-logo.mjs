#!/usr/bin/env node
/**
 * Fetches an integration's brand logo from Brandfetch into the integrations
 * catalog, so every integration card shows the vendor's real mark.
 *
 * Writes <slug>-light.svg (for light mode), <slug>-dark.svg (for dark mode) and
 * <slug>-icon.svg (the square mark, for compact places) to
 * apps/web/public/integrations/logos.
 *
 * A Brandfetch SVG often draws its wordmark in `currentColor`, which an <img>
 * renders black. Those fills are replaced with the brand's own "dark" colour in
 * the light-mode file and its "light" colour in the dark-mode file.
 *
 * When Brandfetch has no square SVG mark, pass --icon-viewbox to crop one out of
 * the logo (the mark's bounding box in the logo's own coordinates).
 *
 * The API key is read from BRANDFETCH_API_KEY and never written anywhere.
 *
 * Usage:
 *   BRANDFETCH_API_KEY=... node scripts/fetch-brand-logo.mjs \
 *     --domain quickbooks.co.uk --slug quickbooks \
 *     [--icon-viewbox "0 27.6024 227.502 227.502"]
 *
 * Always render the three files on a light and a dark background before
 * committing them.
 */

import { writeFileSync } from "node:fs";
import { join } from "node:path";
import { parseArgs } from "node:util";

const OUT_DIR = new URL("../apps/web/public/integrations/logos/", import.meta.url).pathname;
const API = "https://api.brandfetch.io/v2/brands/";

const { values } = parseArgs({
  options: {
    domain: { type: "string" },
    slug: { type: "string" },
    "icon-viewbox": { type: "string" },
  },
});

const fail = (message) => {
  process.stderr.write(`${message}\n`);
  process.exit(1);
};

const key = process.env.BRANDFETCH_API_KEY?.trim();
if (!key) fail("Set BRANDFETCH_API_KEY.");
if (!values.domain) fail("Pass --domain, for example --domain quickbooks.co.uk.");
if (!values.slug || !/^[a-z0-9-]+$/.test(values.slug)) {
  fail("Pass --slug in lowercase letters, digits and dashes, for example --slug quickbooks.");
}

const response = await fetch(API + encodeURIComponent(values.domain), {
  headers: { Authorization: `Bearer ${key}` },
});
if (!response.ok) fail(`Brandfetch answered ${response.status} for ${values.domain}.`);
const brand = await response.json();

const svgOf = (types) => {
  for (const type of types) {
    for (const logo of brand.logos ?? []) {
      if (logo.type !== type) continue;
      const svg = logo.formats.find((format) => format.format === "svg");
      if (svg) return svg.src;
    }
  }
  return null;
};

const colorOf = (type, fallback) =>
  brand.colors?.find((color) => color.type === type)?.hex?.toUpperCase() ?? fallback;

const download = async (src) => {
  const asset = await fetch(src);
  if (!asset.ok) fail(`Could not download ${src}: ${asset.status}.`);
  return asset.text();
};

const recolor = (svg, hex) => svg.replaceAll('fill="currentColor"', `fill="${hex}"`);

const logoSrc = svgOf(["logo"]);
if (!logoSrc) fail(`Brandfetch has no SVG logo for ${values.domain}.`);
const logo = await download(logoSrc);
const ink = colorOf("dark", "#060018");
const paper = colorOf("light", "#FCFCFC");

writeFileSync(join(OUT_DIR, `${values.slug}-light.svg`), recolor(logo, ink));
writeFileSync(join(OUT_DIR, `${values.slug}-dark.svg`), recolor(logo, paper));

const markSrc = svgOf(["symbol", "icon"]);
let icon = null;
if (markSrc) {
  icon = recolor(await download(markSrc), ink);
} else if (values["icon-viewbox"]) {
  const box = values["icon-viewbox"];
  icon = recolor(logo, ink).replace(/<svg\b[^>]*>/, (tag) => {
    const attrs = tag
      .slice(4, -1)
      .replace(/\s(width|height|viewBox)="[^"]*"/g, "")
      .trim();
    return `<svg width="32" height="32" viewBox="${box}"${attrs ? ` ${attrs}` : ""}>`;
  });
}

if (icon) {
  writeFileSync(join(OUT_DIR, `${values.slug}-icon.svg`), icon);
} else {
  process.stderr.write(
    "No square SVG mark on Brandfetch; pass --icon-viewbox to crop one from the logo.\n",
  );
}

process.stdout.write(
  `Wrote ${values.slug}-light.svg and ${values.slug}-dark.svg${icon ? ` and ${values.slug}-icon.svg` : ""} for ${brand.name}.\n`,
);
