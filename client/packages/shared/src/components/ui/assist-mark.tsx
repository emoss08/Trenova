import { createIcon, type IconComponent } from "@trenova/shared/components/icons";

/**
 * The mark for anything the system suggests, drafts or fills in on its own.
 *
 * It is the diamond of an advisory road sign with a single point at its centre:
 * on the road that shape means "here is something to take into account", which
 * is exactly the standing of a machine's suggestion to a dispatcher. Sparkles is
 * the glyph every product spends on this, and it says magic, which is the wrong
 * promise for a rate or a settlement.
 *
 * Built with createIcon so it is a real IconComponent: it takes `size`,
 * `strokeWidth` and `className`, and fits any slot typed for one.
 */
export const ASSIST_MARK_DIAMOND =
  "M2.7 10.3a2.41 2.41 0 0 0 0 3.41l7.59 7.59a2.41 2.41 0 0 0 3.41 0l7.59-7.59a2.41 2.41 0 0 0 0-3.41l-7.59-7.59a2.41 2.41 0 0 0-3.41 0Z";

export const AssistMark: IconComponent = createIcon(
  "AssistMark",
  <>
    <path d={ASSIST_MARK_DIAMOND} />
    <circle cx="12" cy="12" r="2" fill="currentColor" />
  </>,
);
