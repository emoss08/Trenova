import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  cardDay,
  cardMoment,
  cardMomentAfter,
  cardTime,
} from "../../../record-card/record-card-values";
import {
  boardArrival,
  boardReads,
  checkGroups,
  cutRows,
  figureRows,
  formatBoardValue,
  furthestOf,
  halfBlocks,
  initialsOf,
  lockedReading,
  measureMoment,
  measureReading,
  readableIds,
  readingSource,
  readStamp,
  replyFootPlace,
  sourceLine,
  toneClass,
  wordPieces,
  wordRuns,
  writtenSegments,
} from "../board-card";
import {
  ANSWERED_AT,
  HOUR,
  READ_AT,
  closeBoard,
  conversationBoard,
  count,
  duration,
  lateBoard,
  money,
  segment,
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
      {
        measure: "lateness",
        value: duration(50 * 60),
        state: "late",
        at: due,
        place: "",
        tone: "warn",
      },
      100 * 60,
      NOW,
      t,
    );
    expect(reading).toEqual({
      against: `Due ${cardMomentAfter(NOW, due)}`,
      shown: `Due ${cardMomentAfter(NOW, due)}`,
      at: null,
      standing: "50m behind",
      tone: "warn",
      fraction: 0.5,
      held: false,
      projected: false,
    });
  });

  it("says how late the estimate runs past a window still open", () => {
    const reading = measureReading(
      {
        measure: "lateness",
        value: duration(6 * 60),
        state: "projected",
        at: due,
        place: "",
        tone: "warn",
      },
      60 * 60,
      NOW,
      t,
    );
    expect(reading).toEqual({
      against: `Due ${cardMomentAfter(NOW, due)}`,
      shown: `Due ${cardMomentAfter(NOW, due)}`,
      at: null,
      standing: "estimated 6m late",
      tone: "warn",
      fraction: 0.1,
      held: false,
      projected: true,
    });
  });

  it("names the stop a load is measured at, so a multi-stop route says which window it missed", () => {
    const late = measureReading(
      {
        measure: "lateness",
        value: duration(2 * HOUR),
        state: "late",
        at: due,
        place: "Seattle, WA",
        tone: "warn",
      },
      2 * HOUR,
      NOW,
      t,
    );
    expect(late.against).toBe(`Due ${cardMomentAfter(NOW, due)} at Seattle, WA`);
    expect(late.shown).toBe(`Due ${cardMomentAfter(NOW, due)}`);
    expect(late.at).toEqual({ verb: "due", when: cardMomentAfter(NOW, due), place: "Seattle, WA" });

    const delivered = measureReading(
      {
        measure: "lateness",
        value: duration(0),
        state: "delivered",
        at: due,
        place: "Memphis, TN",
        tone: "",
      },
      HOUR,
      NOW,
      t,
    );
    expect(delivered.against).toBe(`Delivered ${cardMomentAfter(NOW, due)} at Memphis, TN`);
    expect(delivered.at).toEqual({
      verb: "delivered",
      when: cardMomentAfter(NOW, due),
      place: "Memphis, TN",
    });
  });

  it("says a load due with no estimate is not estimated rather than on time", () => {
    const reading = measureReading(
      { measure: "lateness", value: duration(0), state: "due", at: due, place: "", tone: "" },
      HOUR,
      NOW,
      t,
    );
    expect(reading.standing).toBe("not estimated");
    expect(reading.against).toBe(`Due ${cardMomentAfter(NOW, due)}`);
    expect(reading.fraction).toBe(0);
    expect(reading.projected).toBe(false);
  });

  it("says a held load waits at the stop", () => {
    const reading = measureReading(
      {
        measure: "lateness",
        value: duration(HOUR),
        state: "held",
        at: due,
        place: "",
        tone: "warn",
      },
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
      { measure: "lateness", value: duration(0), state: "delivered", at: due, place: "", tone: "" },
      HOUR,
      NOW,
      t,
    );
    expect(onTime.standing).toBe("on time");
    expect(onTime.against).toBe(`Delivered ${cardMomentAfter(NOW, due)}`);
    expect(onTime.fraction).toBe(0);

    const late = measureReading(
      {
        measure: "lateness",
        value: duration(20 * 60),
        state: "delivered",
        at: null,
        place: "",
        tone: "",
      },
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
        {
          measure: "amountDue",
          value: money("1200"),
          state: "open",
          at: null,
          place: "",
          tone: "",
        },
        0,
        NOW,
        t,
      ).standing,
    ).toBe("$1,200.00 due");
    expect(
      measureReading(
        {
          measure: "amountDue",
          value: money("0"),
          state: "settled",
          at: null,
          place: "",
          tone: "",
        },
        0,
        NOW,
        t,
      ).standing,
    ).toBe("paid");
  });

  // An invoice falls due on a day (`invoice.DueDateFromPaymentTerm` adds whole days to the
  // invoice date), so its due moment's time of day means nothing and is never said.
  const LAST_YEAR = Date.UTC(2025, 8, 15, 12) / 1000;
  const dayAndYear = (at: number) =>
    new Date(at * 1000).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  const amountDue = (at: number) =>
    measureReading(
      { measure: "amountDue", value: money("2531.25"), state: "open", at, place: "", tone: "warn" },
      0,
      NOW,
      t,
    );

  it("says the day an amount falls due, without its time, and the year only when it is not this year's", () => {
    const thisYear = amountDue(due);
    expect(thisYear.against).toBe(`Due ${cardDay(due)}`);
    expect(thisYear.against).not.toContain(cardTime(due));
    expect(thisYear.standing).toBe("$2,531.25 due");

    const lastYear = amountDue(LAST_YEAR);
    expect(lastYear.against).toBe(`Due ${dayAndYear(LAST_YEAR)}`);
    expect(lastYear.against).toContain("2025");
    expect(lastYear.against).not.toContain(cardTime(LAST_YEAR));
  });

  it("says the day a record is dated for its age, without its time", () => {
    const reading = measureReading(
      { measure: "age", value: duration(3 * 24 * HOUR), state: "", at: due, place: "", tone: "" },
      3 * 24 * HOUR,
      NOW,
      t,
    );
    expect(reading.against).toBe(`Dated ${cardDay(due)}`);
  });

  it("writes a load's appointment within a week of now with its time, and the whole moment and stop in what it is held against", () => {
    const sixDaysAgo = NOW - 6 * 24 * HOUR;
    const reading = measureReading(
      {
        measure: "lateness",
        value: duration(HOUR),
        state: "late",
        at: sixDaysAgo,
        place: "Rockford, IL",
        tone: "",
      },
      HOUR,
      NOW,
      t,
    );
    expect(reading.shown).toBe(`Due ${cardMoment(sixDaysAgo)}`);
    expect(reading.at?.when).toBe(cardMoment(sixDaysAgo));
    expect(reading.against).toBe(`Due ${cardMoment(sixDaysAgo)} at Rockford, IL`);
  });

  it("writes a load's appointment more than a week from now, either way, as its day, keeping the whole moment in what it is held against", () => {
    for (const at of [NOW - 8 * 24 * HOUR, NOW + 8 * 24 * HOUR]) {
      const reading = measureReading(
        { measure: "lateness", value: duration(HOUR), state: "late", at, place: "", tone: "" },
        HOUR,
        NOW,
        t,
      );
      expect(reading.shown).toBe(`Due ${cardDay(at)}`);
      expect(reading.shown).not.toContain(cardTime(at));
      expect(reading.against).toBe(`Due ${cardMoment(at)}`);
    }
  });
});

describe("measureMoment", () => {
  const LAST_YEAR = Date.UTC(2025, 2, 19, 12) / 1000;

  it("keeps the time of a moment up to seven days from now on either side, and gives the day past it", () => {
    const week = 7 * 24 * HOUR;
    expect(measureMoment(NOW, NOW - week, "moment")).toBe(cardMomentAfter(NOW, NOW - week));
    expect(measureMoment(NOW, NOW + week, "moment")).toBe(cardMomentAfter(NOW, NOW + week));
    expect(measureMoment(NOW, NOW - week - 1, "moment")).toBe(cardDay(NOW - week - 1));
    expect(measureMoment(NOW, NOW + week + 1, "moment")).toBe(cardDay(NOW + week + 1));
  });

  it("gives a date its day however near, with the year only when it is not now's, and nothing for none", () => {
    expect(measureMoment(NOW, NOW - HOUR, "date")).toBe(cardDay(NOW - HOUR));
    expect(measureMoment(NOW, LAST_YEAR, "moment")).toBe(
      new Date(LAST_YEAR * 1000).toLocaleDateString(undefined, {
        month: "short",
        day: "numeric",
        year: "numeric",
      }),
    );
    expect(measureMoment(NOW, 0, "moment")).toBe("");
  });
});

describe("wordRuns", () => {
  const join = (text: string) =>
    wordRuns(text)
      .map((run) => run.text)
      .join("");

  it("keeps a record's number and a hyphenated date whole, and the words around them plain", () => {
    expect(wordRuns("SEED-INV-0006 is due 2026-11-29, after SEED-SHP-010.")).toEqual([
      { text: "SEED-INV-0006", whole: true },
      { text: " is due ", whole: false },
      { text: "2026-11-29", whole: true },
      { text: ", after ", whole: false },
      { text: "SEED-SHP-010", whole: true },
      { text: ".", whole: false },
    ]);
  });

  it("leaves plain words, hyphenated words without a digit, amounts and a trailing hyphen alone", () => {
    for (const text of ["a well-known lane", "$2,531.25 owed", "SEED- then more", "three loads"]) {
      expect(wordRuns(text)).toEqual([{ text, whole: false }]);
    }
  });

  it("gives back the words unchanged when its runs are joined", () => {
    for (const text of [
      "",
      "S-24110",
      "Pay SEED-INV-0006 and SEED-INV-0007 by 2026-12-01 — a well-known lane.",
      "SEED- INV-0006-",
    ]) {
      expect(join(text)).toBe(text);
    }
    expect(wordRuns("")).toEqual([]);
  });
});

describe("replyFootPlace", () => {
  const exhibit = (type: string) => type !== "actions" && type !== "prose" && type !== "note";

  it("stands inside the last exhibit when it is drawn full width", () => {
    expect(replyFootPlace([{ type: "records" }], [false], exhibit)).toBe("inside");
    expect(
      replyFootPlace(
        [{ type: "records" }, { type: "records" }, { type: "figures" }],
        [true, true, false],
        exhibit,
      ),
    ).toBe("inside");
  });

  it("stands under the desk when the last row is two halves, the last block is words or buttons, or there are none", () => {
    expect(replyFootPlace([{ type: "records" }, { type: "records" }], [true, true], exhibit)).toBe(
      "under",
    );
    expect(
      replyFootPlace([{ type: "records" }, { type: "actions" }], [false, false], exhibit),
    ).toBe("under");
    expect(replyFootPlace([], [], exhibit)).toBe("under");
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
            place: "",
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
      ["ok", "All clear", 1],
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

/*
A figures block holds up to eight, and the card sets at most four side by
side. More are set in balanced rows, the longer first, so five do not leave a
lone figure on a second row.
*/
describe("figureRows", () => {
  it("sets up to four in one row and more in balanced rows", () => {
    expect(figureRows(0)).toEqual([]);
    expect(figureRows(1)).toEqual([1]);
    expect(figureRows(4)).toEqual([4]);
    expect(figureRows(5)).toEqual([3, 2]);
    expect(figureRows(6)).toEqual([3, 3]);
    expect(figureRows(7)).toEqual([4, 3]);
    expect(figureRows(8)).toEqual([4, 4]);
  });
});

/*
A reply's board is drawn from the words the agent wrote while it is read: the
client splits them as the server does (board.Segments), so a cite becomes the
exhibit's numeral and a {key} the reading's blank, in the same order.
*/
describe("writtenSegments", () => {
  it("splits the words into cites and readings as the server does", () => {
    expect(
      writtenSegments("There are {late} late loads. [Four wait on hours][1] and {far} behind."),
    ).toEqual([
      segment("There are "),
      segment("", { reading: "late" }),
      segment(" late loads. "),
      segment("Four wait on hours", { numeral: 1 }),
      segment(" and "),
      segment("", { reading: "far" }),
      segment(" behind."),
    ]);
  });

  it("leaves words with no cite or reading as one run, and a cite of 0 as words", () => {
    expect(writtenSegments("Nothing is late.")).toEqual([segment("Nothing is late.")]);
    expect(writtenSegments("[Not a cite][0].")).toEqual([
      segment("Not a cite", { numeral: 0 }),
      segment("."),
    ]);
    expect(writtenSegments("{2late} stays")).toEqual([segment("{2late} stays")]);
    expect(writtenSegments("{lateSix} late")).toEqual([
      segment("", { reading: "lateSix" }),
      segment(" late"),
    ]);
  });
});

describe("wordPieces", () => {
  it("cuts the words into the phrases they arrive by, a reading and a cite with their clause", () => {
    const { pieces, phrases } = wordPieces(conversationBoard().verdict);

    expect(phrases).toBe(4);
    expect(
      pieces.map((piece) =>
        piece.kind === "text" ? [piece.phrase, piece.text] : [piece.phrase, piece.kind],
      ),
    ).toEqual([
      [0, "There are "],
      [0, "reading"],
      [0, " late loads, "],
      [1, "the furthest "],
      [1, "reading"],
      [1, " behind, "],
      [2, "with "],
      [2, "reading"],
      [2, " owed. "],
      [3, "cite"],
      [3, "."],
    ]);
  });

  it("counts a last phrase that ends without punctuation", () => {
    expect(wordPieces([segment("Two wait, "), segment("one moved")]).phrases).toBe(2);
    expect(wordPieces([]).phrases).toBe(0);
  });
});

describe("readingSource", () => {
  it("says where a reading was read from, with its aggregate in words", () => {
    const [late, furthest, owed] = conversationBoard().readings;

    expect(readingSource(late, t)).toBe("Counted from 6 shipments · 4 you may read");
    expect(readingSource({ ...furthest, source: { ...furthest.source, readable: 6 } }, t)).toBe(
      "The largest of 6 shipments",
    );
    expect(readingSource({ ...owed, source: { ...owed.source, readable: 6 } }, t)).toBe(
      "Summed from 6 shipments",
    );
  });

  it("names a metric's or an analysis's source as an exhibit's source line does", () => {
    const [late] = conversationBoard().readings;
    const metric = {
      ...late,
      measure: "",
      aggregate: "",
      source: { kind: "metrics", resource: "", count: 0, readable: 0, title: "" },
    };
    expect(readingSource(metric, t)).toBe("Read from Trenova's figures");
    expect(
      readingSource(
        { ...metric, source: { ...metric.source, kind: "analysis", title: "Weekly revenue" } },
        t,
      ),
    ).toBe("Read from the analysis “Weekly revenue”");
  });
});

describe("lockedReading", () => {
  it("says a reading the reader may not read in words, by what it reads", () => {
    const [late, furthest, owed] = conversationBoard().readings;

    expect(lockedReading(owed, t)).toBe("an amount you may not read");
    expect(lockedReading(late, t)).toBe("a count you may not read");
    expect(lockedReading(furthest, t)).toBe("a time you may not read");
  });

  it("says one that could not be read when every record behind it is the reader's", () => {
    const [, , owed] = conversationBoard().readings;
    const metric = {
      ...owed,
      measure: "",
      aggregate: "",
      source: { kind: "metrics", resource: "", count: 0, readable: 0, title: "" },
    };

    expect(lockedReading({ ...owed, source: { ...owed.source, readable: 6 } }, t)).toBe(
      "an amount that could not be read",
    );
    expect(lockedReading(metric, t)).toBe("a figure that could not be read");
  });
});

/*
The conversation draws a list to six rows and says how many more the board
holds; a list of six or fewer is drawn whole.
*/
describe("cutRows", () => {
  it("draws six rows and counts the rest", () => {
    const rows = conversationBoard().blocks[0].rows;
    const { shown, more } = cutRows(rows);

    expect(shown.map((row) => row.label)).toEqual([
      "S-24110",
      "S-24111",
      "S-24112",
      "",
      "S-24114",
      "S-24115",
    ]);
    expect(more).toBe(2);
  });

  it("draws a list of six or fewer whole", () => {
    expect(cutRows([1, 2, 3, 4, 5, 6])).toEqual({ shown: [1, 2, 3, 4, 5, 6], more: 0 });
    expect(cutRows([])).toEqual({ shown: [], more: 0 });
  });
});

describe("boardArrival", () => {
  it("times the words, then each block, then the rows and the readings, a beat apart", () => {
    const arrival = boardArrival(3, 2);

    expect(arrival.exhibitAt(0)).toBe(720);
    expect(arrival.exhibitAt(1)).toBe(960);
    expect(arrival.rowsAt(0)).toBe(1440);
    expect(arrival.rowsAt(1)).toBe(1680);
    expect(arrival.landAt).toBe(1440);
  });
});

/*
A record opened from a list steps through the records of that list a person may
read, in the order drawn; one they may not read is never a stop.
*/
describe("readableIds", () => {
  it("lists the ids of the readable records in order, leaving out one that cannot be read", () => {
    const rows = conversationBoard().blocks[0].rows;

    expect(readableIds(rows)).toEqual([
      "shp_1",
      "shp_2",
      "shp_3",
      "shp_5",
      "shp_6",
      "shp_7",
      "shp_8",
    ]);
    expect(readableIds([])).toEqual([]);
  });
});
