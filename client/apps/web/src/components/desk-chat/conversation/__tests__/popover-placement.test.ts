import { describe, expect, it } from "vitest";
import { popoverPlacement } from "../desk-citations";

const WIDTH = 250;

function mark(left: number, top: number) {
  return { left, top, bottom: top + 16, width: 16 };
}

/*
A footnote's card is 250px wide and hangs above its mark. At the Desk it keeps
12px from the window's edges and opens below when fewer than 230px are free
above. In the assistant's 400px panel the window is the wrong frame: the card
keeps 10px inside the conversation's scroll area and opens below when fewer
than 150px of it are free above the mark.
*/
describe("popoverPlacement at the Desk", () => {
  const desk = { left: 12, right: 1440 - 12, top: 0, room: 230 };

  it("hangs centred above a mark with room all round", () => {
    expect(popoverPlacement(mark(700, 400), WIDTH, desk)).toEqual({ below: false, shift: 0 });
  });

  it("nudges the card in from the window's left edge", () => {
    expect(popoverPlacement(mark(20, 400), WIDTH, desk)).toEqual({ below: false, shift: 109 });
  });

  it("nudges the card in from the window's right edge", () => {
    expect(popoverPlacement(mark(1400, 400), WIDTH, desk).shift).toBe(-105);
  });

  it("opens below a mark near the top of the window", () => {
    expect(popoverPlacement(mark(700, 229), WIDTH, desk).below).toBe(true);
    expect(popoverPlacement(mark(700, 230), WIDTH, desk).below).toBe(false);
  });
});

describe("popoverPlacement in the assistant's panel", () => {
  // The panel's scroll area runs from x 1040 to 1440 and starts 46px down.
  const panel = { left: 1040 + 10, right: 1440 - 10, top: 46, room: 150 };

  it("keeps the card 10px inside the scroll area, not the window", () => {
    const placed = popoverPlacement(mark(1060, 500), WIDTH, panel);
    const cardLeft = 1060 + 8 - WIDTH / 2 + placed.shift;

    expect(cardLeft).toBe(1050);
  });

  it("keeps it inside on the right as well", () => {
    const placed = popoverPlacement(mark(1420, 500), WIDTH, panel);
    const cardRight = 1420 + 8 + WIDTH / 2 + placed.shift;

    expect(cardRight).toBe(1430);
  });

  it("flips below when less than 150px of the scroll area is free above the mark", () => {
    expect(popoverPlacement(mark(1240, 46 + 149), WIDTH, panel).below).toBe(true);
    expect(popoverPlacement(mark(1240, 46 + 150), WIDTH, panel).below).toBe(false);
  });

  it("centres a card wider than the scroll area rather than pushing it off both sides", () => {
    const narrow = { left: 1050, right: 1250, top: 46, room: 150 };
    const placed = popoverPlacement(mark(1100, 500), WIDTH, narrow);
    const cardCentre = 1100 + 8 + placed.shift;

    expect(cardCentre).toBe(1150);
  });
});
