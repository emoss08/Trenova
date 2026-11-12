import type { DeskCaseTone } from "@/lib/graphql/desk-case";
import type { BadgeTone } from "@trenova/shared/types/badge";

// i18n-ignore: badge variants, never shown to a person
const BADGE: Readonly<Record<DeskCaseTone, BadgeTone>> = {
  Neutral: "neutral",
  Good: "success",
  Warning: "warning",
  Bad: "danger",
  Info: "info",
};

// i18n-ignore: class names, never shown to a person
const TEXT: Readonly<Record<DeskCaseTone, string>> = {
  Neutral: "",
  Good: "dk-ok",
  Warning: "dk-meh",
  Bad: "dk-bad",
  Info: "",
};

/** The badge variant for a tone the server gave a line of the case. */
export function badgeTone(tone: DeskCaseTone): BadgeTone {
  return BADGE[tone];
}

/** The text class for a figure of that tone: an arrival, an amount, a meter. */
export function textTone(tone: DeskCaseTone): string {
  return TEXT[tone];
}
