import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { cardDayAfter } from "../../../record-card/record-card-values";
import { SegmentBar } from "../board-bar";
import { formatBoardValue, measureMoment } from "../board-card";
import { fillWords, wholeWords } from "../board-words-fill";
import { ANSWERED_AT, duration, words } from "./fixtures";

afterEach(cleanup);

function parts(container: HTMLElement) {
  const track = container.querySelector(".dk-bd-track") as HTMLElement;
  return {
    track,
    fills: [...track.querySelectorAll(":scope > i")].map((fill) => ({
      className: fill.className,
      left: (fill as HTMLElement).style.left,
      right: (fill as HTMLElement).style.right,
      width: (fill as HTMLElement).style.width,
    })),
    bound: (track.querySelector(":scope > b") as HTMLElement | null)?.style.left ?? null,
  };
}

/*
The bar is the server's: boardbar.Layout puts every row of a block on one scale and the client
draws each segment where it says, classed by its kind. These are the mockup's rows (Measures).
*/
describe("SegmentBar", () => {
  it("draws Hector Alvarez's week: worked, the rest of the rota to the line, the overtime past it", () => {
    const { container } = render(
      <SegmentBar
        bar={{
          centred: false,
          bound: 0.7692,
          segments: [
            { kind: "used", start: 0, width: 0.2212, tone: "" },
            { kind: "ghost", start: 0.2212, width: 0.548, tone: "" },
            { kind: "projected", start: 0.7692, width: 0.1635, tone: "warn" },
          ],
        }}
      />,
    );
    const { fills, bound } = parts(container);

    expect(fills).toEqual([
      { className: "dk-bd-used dk-bd-cut", left: "", right: "", width: "22.12%" },
      {
        className: "dk-bd-ghost dk-bd-next dk-bd-cut",
        left: "22.12%",
        right: "",
        width: "calc(54.8% - 2px)",
      },
      {
        className: "dk-bd-projected dk-warn dk-bd-next",
        left: "76.92%",
        right: "",
        width: "calc(16.35% - 2px)",
      },
    ]);
    expect(bound).toBe("76.92%");
  });

  it("draws a credit to the left of a centred bound, placed from the right", () => {
    const { container } = render(
      <SegmentBar
        bar={{
          centred: true,
          bound: 0.5,
          segments: [{ kind: "credit", start: 0, width: 0.5, tone: "" }],
        }}
      />,
    );
    const { track, fills, bound } = parts(container);

    expect(track.className).toBe("dk-bd-track dk-bd-centred");
    expect(fills).toEqual([{ className: "dk-bd-credit", left: "", right: "50%", width: "50%" }]);
    expect(bound).toBe("50%");
  });

  it("draws an empty track with no mark for a row on time", () => {
    const { container } = render(
      <SegmentBar bar={{ centred: false, bound: null, segments: [] }} />,
    );
    const { fills, bound } = parts(container);

    expect(fills).toEqual([]);
    expect(bound).toBeNull();
  });
});

describe("fillWords", () => {
  it("fills the slot with the value, or the moment as a day when the server says it is a date", () => {
    const now = ANSWERED_AT;
    const ends = now + 3 * 86_400;

    expect(fillWords(words("{0} left", { value: duration(3 * 86_400) }), now)).toBe(
      `${formatBoardValue(duration(3 * 86_400))} left`,
    );
    expect(fillWords(words("Expires {0}", { at: ends, day: true }), now)).toBe(
      `Expires ${measureMoment(now, ends, "date")}`,
    );
    expect(fillWords(words("Not estimated"), now)).toBe("Not estimated");
    expect(fillWords(null, now)).toBe("");
  });

  it("writes the whole fact with its place for a title", () => {
    const now = ANSWERED_AT;
    const ends = now + 40 * 86_400;

    expect(wholeWords(words("Expires {0}", { at: ends, day: true, place: "at Dallas" }), now)).toBe(
      `Expires ${cardDayAfter(now, ends)} at Dallas`,
    );
  });
});
