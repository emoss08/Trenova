/* The Maps API rasterises a literal colour, so a token is read off the document
   when a shape is styled; this grey stands in only if the token is missing. */
const FALLBACK_COLOR = "#7a7a7a"; // design-tokens-ignore: Maps API needs a literal

/** The current value of a colour token, such as `--brand`, for a shape the Maps API draws. */
export function mapColor(name: string): string {
  if (typeof window === "undefined") return FALLBACK_COLOR;
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return value || FALLBACK_COLOR;
}
