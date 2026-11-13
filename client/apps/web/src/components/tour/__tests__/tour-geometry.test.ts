import { describe, expect, it } from "vitest";
import { grow, overlap, placeCard, sameBox } from "../tour-geometry";

const viewport = { width: 1440, height: 900 };
const card = { width: 352, height: 220 };

describe("overlap", () => {
  it("is the part two boxes share", () => {
    expect(
      overlap(
        { x: 0, y: 100, width: 800, height: 2000 },
        { x: 40, y: 500, width: 700, height: 300 },
      ),
    ).toEqual({ x: 40, y: 500, width: 700, height: 300 });
    expect(
      overlap({ x: 0, y: 0, width: 100, height: 100 }, { x: 60, y: 80, width: 100, height: 100 }),
    ).toEqual({ x: 60, y: 80, width: 40, height: 20 });
  });

  it("is nothing for boxes that only touch or sit apart", () => {
    expect(
      overlap({ x: 0, y: 0, width: 100, height: 100 }, { x: 100, y: 0, width: 50, height: 50 }),
    ).toBeNull();
    expect(
      overlap({ x: 0, y: 0, width: 100, height: 100 }, { x: 0, y: 400, width: 50, height: 50 }),
    ).toBeNull();
  });
});

describe("grow and sameBox", () => {
  it("grows a box by the same amount on every side", () => {
    expect(grow({ x: 10, y: 20, width: 100, height: 50 }, 6)).toEqual({
      x: 4,
      y: 14,
      width: 112,
      height: 62,
    });
  });

  it("compares boxes by their numbers, and nothing only with nothing", () => {
    const box = { x: 1, y: 2, width: 3, height: 4 };
    expect(sameBox(box, { ...box })).toBe(true);
    expect(sameBox(box, { ...box, height: 5 })).toBe(false);
    expect(sameBox(null, null)).toBe(true);
    expect(sameBox(box, null)).toBe(false);
  });
});

describe("placeCard", () => {
  it("hangs the card off the side the step asked for when it fits there", () => {
    const lit = { x: 200, y: 300, width: 600, height: 120 };

    expect(placeCard({ lit, viewport, card, preferred: "bottom" })).toEqual({
      anchor: lit,
      side: "bottom",
      gap: 14,
    });
    expect(placeCard({ lit, viewport, card, preferred: "left" })).toEqual({
      anchor: lit,
      side: "right",
      gap: 14,
    });
  });

  it("takes the opposite side when the asked one has no room", () => {
    const nearTheTop = { x: 200, y: 60, width: 600, height: 80 };
    const nearTheBottom = { x: 200, y: 780, width: 600, height: 80 };

    expect(placeCard({ lit: nearTheTop, viewport, card, preferred: "top" }).side).toBe("bottom");
    expect(placeCard({ lit: nearTheBottom, viewport, card, preferred: "bottom" }).side).toBe("top");
  });

  it("goes beside a part that runs the window's height, on the roomier side", () => {
    const brief = { x: 1040, y: 60, width: 400, height: 840 };
    const rail = { x: 0, y: 60, width: 300, height: 840 };

    expect(placeCard({ lit: brief, viewport, card, preferred: "top" })).toEqual({
      anchor: brief,
      side: "left",
      gap: 14,
    });
    expect(placeCard({ lit: rail, viewport, card, preferred: "top" }).side).toBe("right");
  });

  it("counts a side as room only when the card, its gap and the window's edge all fit", () => {
    const lit = { x: 200, y: 246, width: 600, height: 500 };

    expect(placeCard({ lit, viewport, card, preferred: "top" }).side).toBe("top");
    expect(placeCard({ lit: { ...lit, y: 245 }, viewport, card, preferred: "top" }).side).not.toBe(
      "top",
    );
  });

  it("keeps the card inside the window for a part that leaves no side with room", () => {
    const list = { x: 0, y: 180, width: 1440, height: 720 };

    expect(placeCard({ lit: list, viewport, card, preferred: "top" })).toEqual({
      anchor: { x: 720, y: 12, width: 0, height: 0 },
      side: "bottom",
      gap: 0,
    });
  });

  it("puts that card against the edge that covers less of the part", () => {
    const fromTheTop = { x: 0, y: 0, width: 1440, height: 760 };
    const fromTheBottom = { x: 0, y: 140, width: 1440, height: 760 };

    expect(placeCard({ lit: fromTheTop, viewport, card, preferred: "left" })).toEqual({
      anchor: { x: 720, y: 888, width: 0, height: 0 },
      side: "top",
      gap: 0,
    });
    expect(placeCard({ lit: fromTheBottom, viewport, card, preferred: "left" })).toEqual({
      anchor: { x: 720, y: 12, width: 0, height: 0 },
      side: "bottom",
      gap: 0,
    });
  });

  it("stays inside a window narrower than the card's usual room", () => {
    const phone = { width: 390, height: 700 };
    const lit = { x: 16, y: 240, width: 358, height: 260 };

    expect(placeCard({ lit, viewport: phone, card, preferred: "right" })).toEqual({
      anchor: { x: 195, y: 12, width: 0, height: 0 },
      side: "bottom",
      gap: 0,
    });
  });
});
