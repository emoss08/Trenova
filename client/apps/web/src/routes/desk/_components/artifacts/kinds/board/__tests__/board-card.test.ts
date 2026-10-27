import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { cardMomentAfter, cardTime } from "../../../record-card/record-card-values";
import {
  boardReads,
  checkGroups,
  formatBoardValue,
  furthestOf,
  halfBlocks,
  initialsOf,
  measureReading,
  readStamp,
  sourceLine,
  toneClass,
} from "../board-card";
import {
  ANSWERED_AT,
  HOUR,
  READ_AT,
  closeBoard,
  count,
  duration,
  lateBoard,
  money,
  shipmentRow,
} from "./fixtures";

const t = translate as TranslateFn;
const NOW = READ_AT;

describe("halfBlocks", () => {
  it("pairs half blocks two at a time and draws one left without a partner full width", () => {
    const blocks = [true, true, true, false, true, false, true, true].map((half) => ({ half }));
    expect(halfBlocks(blocks)).toEqual([true, true, false, false, false, false, true, true]);
  });

  it("draws a lone half block at full width", () => {
    expect(halfBlocks([{ half: true }])).toEqual([false]);
  });
});

describe("formatBoardValue", () => {
  it("writes each kind of value as the board shows it", () => {
    expect(formatBoardValue(count(1240))).toBe("1,240");
    expect(formatBoardValue(money("18240.5"))).toBe("$18,240.50");
    expect(formatBoardValue({ kind: "percent", number: "0.875", currency: "", text: "" })).toBe(
      "87.5%",
    );
    expect(formatBoardValue({ kind: "text", number: "", currency: "", text: "Dallas" })).toBe(
      "Dallas",
    );
    expect(formatBoardValue(null)).toBe("");
  });
});

describe("sourceLine", () => {
  it("says how many of the named records the reader may read when some are hidden", () => {
    expect(
      sourceLine({ kind: "records", resource: "shipment", count: 3, readable: 2, title: "" }, t),
    ).toBe("Read from 3 shipments · 2 you may read");
  });

  it("names where a block was read from when nothing is hidden", () => {
    expect(
      sourceLine({ kind: "records", resource: "invoice", count: 1, readable: 1, title: "" }, t),
    ).toBe("Read from 1 invoice");
    expect(
      sourceLine({ kind: "analysis", resource: "", count: 0, readable: 0, title: "Lanes" }, t),
    ).toBe("Read from the analysis “Lanes”");
    expect(sourceLine({ kind: "metrics", resource: "", count: 0, readable: 0, title: "" }, t)).toBe(
      "Read from Trenova's figures",
    );
    expect(sourceLine({ kind: "", resource: "", count: 0, readable: 0, title: "" }, t)).toBe("");
  });
});

describe("measureReading", () => {
  const due = ANSWERED_AT - 2 * HOUR;

  it("measures a late load against its appointment and the furthest behind", () => {
    const reading = measureReading(
      { measure: "lateness", value: duration(50 * 60), state: "late", at: due, tone: "warn" },
      100 * 60,
      NOW,
      t,
    );
    expect(reading).toEqual({
      against: `Due ${cardMomentAfter(NOW, due)}`,
      standing: "50m behind",
      tone: "warn",
      fraction: 0.5,
      held: false,
    });
  });

  it("says a held load waits at the stop", () => {
    const reading = measureReading(
      { measure: "lateness", value: duration(HOUR), state: "held", at: due, tone: "warn" },
      HOUR,
      NOW,
      t,
    );
    expect(reading.standing).toBe("1h at the stop");
    expect(reading.held).toBe(true);
    expect(reading.fraction).toBe(1);
  });

  it("says a delivered load arrived on time, or how late, with no track", () => {
    const onTime = measureReading(
      { measure: "lateness", value: duration(0), state: "delivered", at: due, tone: "" },
      HOUR,
      NOW,
      t,
    );
    expect(onTime.standing).toBe("on time");
    expect(onTime.against).toBe(`Delivered ${cardMomentAfter(NOW, due)}`);
    expect(onTime.fraction).toBe(0);

    const late = measureReading(
      { measure: "lateness", value: duration(20 * 60), state: "delivered", at: null, tone: "" },
      HOUR,
      NOW,
      t,
    );
    expect(late.standing).toBe("20m late");
    expect(late.against).toBe("Delivered");
  });

  it("says what an invoice still owes, or that it was paid", () => {
    expect(
      measureReading(
        { measure: "amountDue", value: money("1200"), state: "open", at: null, tone: "" },
        0,
        NOW,
        t,
      ).standing,
    ).toBe("$1,200.00 due");
    expect(
      measureReading(
        { measure: "amountDue", value: money("0"), state: "settled", at: null, tone: "" },
        0,
        NOW,
        t,
      ).standing,
    ).toBe("paid");
  });
});

describe("furthestOf", () => {
  it("is the longest duration among the rows, ignoring rows without one", () => {
    expect(
      furthestOf([
        shipmentRow(),
        shipmentRow({ id: "shp_2", measure: null }),
        shipmentRow({
          id: "shp_3",
          measure: {
            measure: "lateness",
            value: duration(3 * HOUR),
            state: "late",
            at: null,
            tone: "",
          },
        }),
      ]),
    ).toBe(3 * HOUR);
  });
});

describe("checkGroups", () => {
  it("groups checks worst first and leaves out groups with none", () => {
    const groups = checkGroups(closeBoard().blocks[0].checks, t);
    expect(groups.map((group) => [group.tone, group.label, group.checks.length])).toEqual([
      ["bad", "Blocks it", 1],
      ["ok", "Clear", 1],
    ]);
  });
});

describe("boardReads", () => {
  it("lists each exhibit by its numeral and strikes the one that could not be read", () => {
    expect(boardReads(closeBoard(), t)).toEqual([
      { numeral: 1, phrase: "the checks above", struck: false },
      { numeral: 2, phrase: "the checks above", struck: false },
      { numeral: 3, phrase: "could not be read", struck: true },
    ]);
    expect(boardReads(lateBoard(), t).map((read) => read.phrase)).toEqual([
      "6 shipments",
      "3 shipments",
      "its history",
    ]);
  });
});

describe("small helpers", () => {
  it("takes initials from the first two words of a label", () => {
    expect(initialsOf("Swift Transportation")).toBe("ST");
    expect(initialsOf("J.B. Hunt")).toBe("JB");
    expect(initialsOf("Werner")).toBe("W");
  });

  it("knows only the board's tones", () => {
    expect(toneClass("warn")).toBe("dk-warn");
    expect(toneClass("loud")).toBe("");
  });

  it("stamps when the board was read", () => {
    expect(readStamp(lateBoard(), t)).toBe(`read ${cardTime(READ_AT)}`);
  });
});
