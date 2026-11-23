import { boardPlaceOf } from "@/lib/board-place";
import { describe, expect, it } from "vitest";
import {
  answerLayoutFrom,
  boardFrom,
  boardLayoutFrom,
  callLayoutFrom,
  isBoardExhibit,
} from "../../../artifact-payloads";
import { CONVERSATION_SPEC, boardArtifact } from "./fixtures";

/*
The stored payload is what Go's board.Payload writes: the layout under `board`
(with `board.place` and `board.readings`), the same place again at the top,
the snapshot and the author. A board stored before boards had a place has
neither and is beside the conversation.
*/
describe("boardFrom", () => {
  it("reads a conversation board's place and the readings its words place", () => {
    const artifact = boardArtifact();
    artifact.payload = {
      board: {
        ...CONVERSATION_SPEC,
        readings: [
          {
            key: "late",
            label: "Late now",
            records: { resource: "shipment", ids: ["shp_1"] },
            measure: "lateness",
            aggregate: "count",
          },
          { key: "", label: "No key" },
          "not a reading",
          { key: "open", metric: "ar_open" },
        ],
      },
      place: "conversation",
      snapshot: { at: 1_793_114_000, values: { "r.late": { kind: "count", number: "6" } } },
      author: "Dispatch desk",
    };

    expect(boardFrom(artifact)).toEqual({
      title: "Six late loads on the Dallas lanes",
      verdict:
        "There are {late} late loads, the furthest {furthest} behind, with {owed} owed. [Four wait on one driver's hours][1].",
      author: "Dispatch desk",
      place: "conversation",
      readings: [
        { key: "late", label: "Late now" },
        { key: "open", label: "" },
      ],
      blocks: [
        {
          type: "records",
          title: "The late loads, furthest behind first",
          half: false,
          size: 3,
          caption: "",
          measure: "lateness",
        },
        { type: "actions", title: "", half: false, size: 1, caption: "", measure: "" },
      ],
    });
  });

  it("reads the measure a records block is read by under the name the server keeps, and none for other blocks", () => {
    const layout = boardLayoutFrom(
      {
        blocks: [
          {
            type: "records",
            records: { resource: "invoice", ids: ["inv_1"] },
            measure: "balanceDue",
          },
          { type: "records", records: { resource: "shipment", ids: ["shp_1"] } },
          { type: "figures", measure: "lateness", figures: [{ label: "Late" }] },
        ],
      },
      { title: "", author: "" },
    );

    expect(layout.blocks.map((block) => block.measure)).toEqual(["amountDue", "", ""]);
  });

  it("reads a board stored with no place and no readings as beside, with none", () => {
    const layout = boardFrom(boardArtifact());

    expect(layout.place).toBe("beside");
    expect(layout.readings).toEqual([]);
    expect(layout.blocks.map((block) => block.type)).toEqual(["figures", "records", "note"]);
  });

  it("takes the place from the board when the top-level one is missing, and refuses any other value", () => {
    const artifact = boardArtifact();
    artifact.payload = { board: { place: "conversation", blocks: [] } };
    expect(boardFrom(artifact).place).toBe("conversation");

    artifact.payload = { board: { place: "inline" }, place: "elsewhere" };
    expect(boardFrom(artifact).place).toBe("beside");
  });

  it("keeps work and actions blocks, and numbers only the exhibits", () => {
    const artifact = boardArtifact();
    artifact.payload = {
      board: {
        blocks: [
          { type: "work", records: { resource: "shipment", ids: ["shp_1", "shp_2"] } },
          { type: "actions", actions: [{ label: "A" }, { label: "B" }] },
          { type: "prose", text: "Words." },
          { type: "nonsense" },
        ],
      },
    };

    const blocks = boardFrom(artifact).blocks;
    expect(blocks.map((block) => [block.type, block.size])).toEqual([
      ["work", 2],
      ["actions", 2],
      ["prose", 0],
    ]);
    expect(blocks.map((block) => isBoardExhibit(block.type))).toEqual([true, false, false]);
  });
});

describe("boardLayoutFrom", () => {
  it("reads a show_board call's arguments as the layout, with the fallback's title when it has none", () => {
    const layout = boardLayoutFrom(
      { ...CONVERSATION_SPEC, title: "" },
      { title: "From the artifact", author: "" },
    );

    expect(layout.title).toBe("From the artifact");
    expect(layout.place).toBe("conversation");
    expect(layout.readings.map((reading) => reading.key)).toEqual(["late", "furthest", "owed"]);
  });

  it("draws nothing from arguments it cannot read", () => {
    expect(boardLayoutFrom({}, { title: "Board", author: "" })).toEqual({
      title: "Board",
      verdict: "",
      author: "",
      place: "beside",
      readings: [],
      blocks: [],
    });
  });
});

describe("boardPlaceOf", () => {
  it("reads a summary's top-level place first, then the board's", () => {
    expect(boardPlaceOf({ place: "conversation" })).toBe("conversation");
    expect(boardPlaceOf({ place: "beside", board: { place: "conversation" } })).toBe("beside");
    expect(boardPlaceOf({ board: { place: "conversation" } })).toBe("conversation");
    expect(boardPlaceOf({})).toBe("beside");
  });
});

const FALLBACK = { title: "From the artifact", author: "Dispatch desk" };

/*
A show_answer call is not a board spec: its arguments are a title, the answer,
one or two groups of records and up to two actions (runtimetools/answer.go).
The server builds a board from them, and the reply lays that board out from
the call while it is read: the answer as the verdict, a records block per
group with its note as the block's caption, side by side when there are two,
and an actions block. Its figures are a fixed vocabulary, never declared.
*/
describe("answerLayoutFrom", () => {
  it("lays one group out as a full-width records block with its note as the caption", () => {
    expect(
      answerLayoutFrom(
        {
          title: "Late loads on the Dallas lanes",
          answer: "There are {count} late loads, the furthest {largest} behind.",
          measure: "lateness",
          groups: [
            {
              title: "The late loads",
              records: ["shp_1", "shp_2", "shp_3"],
              note: "All three are past their window.",
            },
          ],
        },
        FALLBACK,
      ),
    ).toEqual({
      title: "Late loads on the Dallas lanes",
      verdict: "There are {count} late loads, the furthest {largest} behind.",
      author: "Dispatch desk",
      place: "conversation",
      readings: [
        { key: "count", label: "" },
        { key: "largest", label: "" },
      ],
      blocks: [
        {
          type: "records",
          title: "The late loads",
          half: false,
          size: 3,
          caption: "All three are past their window.",
          measure: "lateness",
        },
      ],
    });
  });

  it("sets two groups side by side and adds an actions block, with the readings their notes place", () => {
    const layout = answerLayoutFrom(
      {
        title: "Who they wait on",
        answer: "Four wait on hours and two are at the dock.",
        groups: [
          {
            title: "Waiting on hours",
            records: ["shp_1", "shp_2", "shp_3", "shp_4"],
            note: "{count_1} wait on Marcus Bell, the furthest {largest_1} behind.",
          },
          { title: "At the dock", records: ["shp_5", "shp_6"] },
        ],
        actions: [
          { label: "Ask the dock", ask: "Ask the Memphis dock for an unload time." },
          { label: "Call Marcus", ask: "Call Marcus Bell." },
        ],
      },
      FALLBACK,
    );

    expect(layout.blocks).toEqual([
      {
        type: "records",
        title: "Waiting on hours",
        half: true,
        size: 4,
        caption: "{count_1} wait on Marcus Bell, the furthest {largest_1} behind.",
        measure: "",
      },
      { type: "records", title: "At the dock", half: true, size: 2, caption: "", measure: "" },
      { type: "actions", title: "", half: false, size: 2, caption: "", measure: "" },
    ]);
    expect(layout.readings.map((reading) => reading.key)).toEqual(["count_1", "largest_1"]);
    expect(layout.blocks.map((block) => isBoardExhibit(block.type))).toEqual([true, true, false]);
  });

  it("drops what it cannot read: an empty or malformed group, a fourth group, malformed actions, unknown figures", () => {
    const layout = answerLayoutFrom(
      {
        answer: "There are {count} late, {nonsense} and {Count}.",
        groups: [
          "not a group",
          { title: "Empty", records: [] },
          { title: "First", records: ["shp_1"] },
          { title: "Second", records: ["shp_2"], note: 7 },
          { title: "Third", records: ["shp_3"] },
          { title: "Fourth", records: ["shp_4"] },
        ],
        actions: ["not an action", null],
      },
      FALLBACK,
    );

    expect(layout.title).toBe("From the artifact");
    expect(layout.blocks).toEqual([
      { type: "records", title: "First", half: false, size: 1, caption: "", measure: "" },
      { type: "records", title: "Second", half: false, size: 1, caption: "", measure: "" },
      { type: "records", title: "Third", half: false, size: 1, caption: "", measure: "" },
    ]);
    expect(layout.readings).toEqual([{ key: "count", label: "" }]);
  });

  it("reads one record's own figure by its measure and number, in either order, and keys it as the server does", () => {
    const layout = answerLayoutFrom(
      {
        title: "SEED-SHP-010 and SEED-SHP-008 compared",
        answer:
          "SEED-SHP-010 is {lateness:SEED-SHP-010} behind and brings in {revenue:SEED-SHP-010}, against {SEED-SHP-008:lateness} and {totalCharge:SEED-SHP-008}.",
        measure: "lateness",
        groups: [
          { title: "SEED-SHP-010", records: ["SEED-SHP-010", "SEED-SHP-011"] },
          { title: "SEED-SHP-008", records: ["SEED-SHP-008"] },
        ],
      },
      FALLBACK,
    );

    expect(layout.verdict).toBe(
      "SEED-SHP-010 is {lateness_of_1} behind and brings in {revenue_of_1}, against {lateness_of_3} and {revenue_of_3}.",
    );
    expect(layout.readings.map((reading) => reading.key)).toEqual([
      "lateness_of_3",
      "lateness_of_1",
      "revenue_of_1",
      "revenue_of_3",
    ]);
  });

  it("reads a figure that names its measure after a colon, in the answer and in a group's note", () => {
    const layout = answerLayoutFrom(
      {
        answer:
          "Billed {total:revenue}, the most {largest:lateness} behind, {count:anything} loads.",
        measure: "lateness",
        groups: [
          {
            title: "Held",
            records: ["SEED-SHP-010", "SEED-SHP-011"],
            note: "{total_1:revenue} billed and {largest_1:lateness} behind.",
          },
          { title: "Dock", records: ["SEED-SHP-008", "SEED-SHP-009"] },
        ],
      },
      FALLBACK,
    );

    expect(layout.verdict).toBe(
      "Billed {total_revenue}, the most {largest} behind, {count} loads.",
    );
    expect(layout.blocks[0].caption).toBe("{total_1_revenue} billed and {largest_1} behind.");
    expect(layout.readings.map((reading) => reading.key)).toEqual([
      "count",
      "total_revenue",
      "largest",
      "total_1_revenue",
      "largest_1",
    ]);
  });

  it("blanks a record's own figure written by number when the groups hold the records by id, one key per record and measure", () => {
    const layout = answerLayoutFrom(
      {
        answer:
          "SEED-SHP-010 is {lateness:SEED-SHP-010} behind and brings in {revenue:SEED-SHP-010}, against {lateness:SEED-SHP-008} and {SEED-SHP-008:revenue}, {revenue:SEED-SHP-010} again.",
        measure: "lateness",
        groups: [
          {
            title: "A",
            records: ["shp_01JTEN00000000000000000001", "shp_01JTEN00000000000000000002"],
          },
          { title: "B", records: ["shp_01JEIGHT000000000000000001"] },
        ],
      },
      FALLBACK,
    );

    expect(layout.verdict).toBe(
      "SEED-SHP-010 is {lateness_for_1} behind and brings in {revenue_for_1}, against {lateness_for_2} and {revenue_for_2}, {revenue_for_1} again.",
    );
    expect(layout.readings.map((reading) => reading.key)).toEqual([
      "lateness_for_2",
      "lateness_for_1",
      "revenue_for_2",
      "revenue_for_1",
    ]);
  });

  it("leaves what the server would not read as a figure as words", () => {
    const answer =
      "{total:bogus} {total_3} {count_2} {nonsense:SEED-SHP-010} {revenue:} {total_1:revenue:x}";
    const layout = answerLayoutFrom(
      {
        answer,
        measure: "lateness",
        groups: [{ title: "Held", records: ["SEED-SHP-010"] }],
      },
      FALLBACK,
    );

    expect(layout.verdict).toBe(answer);
    expect(layout.readings).toEqual([]);
  });

  it("lays nothing out from arguments that are not a show_answer call's, and is always in the conversation", () => {
    expect(answerLayoutFrom({}, FALLBACK)).toEqual({
      title: "From the artifact",
      verdict: "",
      author: "Dispatch desk",
      place: "conversation",
      readings: [],
      blocks: [],
    });
  });

  it("is chosen for a show_answer call, and a board's own reader for a show_board call", () => {
    const answer = { answer: "Two are late.", groups: [{ title: "Late", records: ["shp_1"] }] };

    expect(callLayoutFrom("show_answer", answer, FALLBACK).verdict).toBe("Two are late.");
    expect(callLayoutFrom("show_board", answer, FALLBACK).verdict).toBe("");
    expect(
      callLayoutFrom(
        "show_board",
        { verdict: "Six are late.", blocks: [{ type: "records", caption: "Past due." }] },
        FALLBACK,
      ).blocks[0].caption,
    ).toBe("Past due.");
  });
});
