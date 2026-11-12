import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { agoOrIn, spanBetween, trackShare } from "../panel-time";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d+)\}/g, (_match, index: string) =>
    String(args[Number(index)]),
  )) as unknown as TranslateFn;

const NOW = 1_800_000_000;
const HOUR = 3600;

describe("a panel's times", () => {
  it("says how long between two instants, whichever came first", () => {
    expect(spanBetween(NOW - 43 * HOUR, NOW)).toBe(spanBetween(NOW, NOW - 43 * HOUR));
    expect(spanBetween(NOW, NOW + 5)).not.toBe("");
  });

  it("puts an instant before or after now, and says nothing of one that is not known", () => {
    expect(agoOrIn(t, NOW - 2 * HOUR, NOW)).toBe(`${spanBetween(NOW - 2 * HOUR, NOW)} ago`);
    expect(agoOrIn(t, NOW + 6 * HOUR, NOW)).toBe(`in ${spanBetween(NOW, NOW + 6 * HOUR)}`);
    expect(agoOrIn(t, NOW + 20, NOW)).toBe("Just now");
    expect(agoOrIn(t, 0, NOW)).toBe("");
  });

  it("places an instant along the track and keeps a mark at either end whole", () => {
    expect(trackShare(50, 0, 100)).toBe(50);
    expect(trackShare(0, 0, 100)).toBe(3);
    expect(trackShare(100, 0, 100)).toBe(97);
    expect(trackShare(500, 0, 100)).toBe(97);
    expect(trackShare(5, 10, 10)).toBe(50);
  });
});
