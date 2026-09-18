/* The two axes a badge is allowed to vary on. See docs/engineering/design-system.md.

   A tone states severity. An accent distinguishes categories that have no
   severity — a HOS duty status, an agent trigger type. Picking an accent for
   something that has a severity is the mistake this split exists to prevent. */

export const BADGE_TONES = [
  "neutral",
  "brand",
  "info",
  "success",
  "warning",
  "danger",
] as const;

export const BADGE_ACCENTS = [
  "accent-indigo",
  "accent-teal",
  "accent-amber",
  "accent-rose",
  "accent-emerald",
  "accent-sky",
  "accent-violet",
  "accent-slate",
] as const;

export type BadgeTone = (typeof BADGE_TONES)[number];
export type BadgeAccent = (typeof BADGE_ACCENTS)[number];
export type BadgeAppearance = "subtle" | "solid" | "outline";

export type BadgeVariant = BadgeTone | BadgeAccent;
