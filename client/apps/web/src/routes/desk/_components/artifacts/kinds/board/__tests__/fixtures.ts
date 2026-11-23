import type {
  DeskBoardBlock,
  DeskBoardCard,
  DeskBoardRow,
  DeskBoardSegment,
  DeskBoardValue,
} from "@/lib/graphql/desk-board-card";
import type { AssistantArtifact } from "@/types/assistant";

export const ANSWERED_AT = 1_793_114_000;
export const READ_AT = ANSWERED_AT + 25 * 60;
export const HOUR = 3600;

export function count(n: number): DeskBoardValue {
  return { kind: "count", number: String(n), currency: "", text: "" };
}

export function duration(seconds: number): DeskBoardValue {
  return { kind: "duration", number: String(seconds), currency: "", text: "" };
}

export function money(amount: string, currency = "USD"): DeskBoardValue {
  return { kind: "money", number: amount, currency, text: "" };
}

export function shipmentRow(overrides: Partial<DeskBoardRow> = {}): DeskBoardRow {
  return {
    resource: "shipment",
    id: "shp_1",
    readable: true,
    label: "S-24117",
    subtitle: "Dallas, TX → Memphis, TN",
    status: "InTransit",
    measure: {
      measure: "lateness",
      value: duration(100 * 60),
      state: "late",
      at: ANSWERED_AT - 2 * HOUR,
      place: "",
      tone: "warn",
    },
    ...overrides,
  };
}

export function unreadableRow(resource = "shipment"): DeskBoardRow {
  return {
    resource,
    id: "",
    readable: false,
    label: "",
    subtitle: "",
    status: "",
    measure: null,
  };
}

export function block(overrides: Partial<DeskBoardBlock>): DeskBoardBlock {
  return {
    type: "figures",
    title: "",
    half: false,
    numeral: 0,
    caption: [],
    state: "ready",
    tone: "",
    measure: "",
    noteTone: "",
    source: { kind: "", resource: "", count: 0, readable: 0, title: "" },
    figures: [],
    rows: [],
    compare: null,
    table: null,
    chart: null,
    events: [],
    checks: [],
    meter: null,
    text: [],
    chips: [],
    work: [],
    actions: [],
    ...overrides,
  };
}

/** Six late loads: figures that moved, rows with one the reader may not read, a note and a log. */
export function lateBoard(): DeskBoardCard {
  return {
    artifactId: "art_board",
    threadId: "athr_1",
    title: "Six loads are running late on the Dallas lanes",
    author: "Dispatch Desk",
    question: "Which Dallas loads are late and why?",
    answeredAt: ANSWERED_AT,
    readAt: READ_AT,
    place: "beside",
    readings: [],
    markdown: "# Six loads are running late on the Dallas lanes\n\nFour are held at the dock.\n",
    verdict: [
      { text: "Four are ", numeral: 0, tone: "", struck: false, reading: "" },
      { text: "held at the dock", numeral: 1, tone: "warn", struck: false, reading: "" },
      { text: " and two are on the road.", numeral: 0, tone: "", struck: false, reading: "" },
    ],
    blocks: [
      block({
        type: "figures",
        title: "Where they stand",
        numeral: 1,
        source: { kind: "records", resource: "shipment", count: 6, readable: 6, title: "" },
        figures: [
          { label: "Late now", value: count(6), was: count(5), tone: "warn" },
          { label: "Held at a stop", value: count(4), was: null, tone: "" },
          { label: "Revenue at risk", value: null, was: null, tone: "" },
        ],
      }),
      block({
        type: "records",
        title: "The late loads",
        numeral: 2,
        half: true,
        measure: "lateness",
        source: { kind: "records", resource: "shipment", count: 3, readable: 2, title: "" },
        rows: [
          shipmentRow(),
          shipmentRow({
            id: "shp_2",
            label: "S-24130",
            subtitle: "Dallas, TX → Tulsa, OK",
            measure: {
              measure: "lateness",
              value: duration(50 * 60),
              state: "held",
              at: ANSWERED_AT - HOUR,
              place: "",
              tone: "warn",
            },
          }),
          unreadableRow(),
        ],
      }),
      block({
        type: "note",
        title: "Call the Memphis consignee",
        half: true,
        noteTone: "warn",
        text: [{ text: "They close at five.", numeral: 0, tone: "", struck: false, reading: "" }],
        chips: [shipmentRow()],
      }),
      block({
        type: "timeline",
        title: "What happened to S-24117",
        numeral: 3,
        source: { kind: "activity", resource: "shipment", count: 1, readable: 1, title: "" },
        events: [
          {
            at: ANSWERED_AT - 3 * HOUR,
            operation: "update",
            actor: "Ana Ruiz",
            principalType: "user",
            comment: "Driver reported a flat tire",
            changes: null,
          },
          {
            at: ANSWERED_AT - 2 * HOUR,
            operation: "update",
            actor: "",
            principalType: "api_key",
            comment: "Appointment moved",
            changes: null,
          },
        ],
      }),
      block({
        type: "prose",
        text: [
          { text: "The rest are on schedule.", numeral: 0, tone: "", struck: false, reading: "" },
        ],
      }),
    ],
  };
}

/** A month-end close: checks, the meter they fill, and a block whose source is gone. */
export function closeBoard(): DeskBoardCard {
  return {
    ...lateBoard(),
    title: "October can close once two entries post",
    verdict: [
      { text: "Two entries still wait.", numeral: 0, tone: "", struck: false, reading: "" },
    ],
    blocks: [
      block({
        type: "checks",
        title: "Before the close",
        numeral: 1,
        source: { kind: "checks", resource: "", count: 3, readable: 3, title: "" },
        checks: [
          {
            title: "Bank reconciled",
            tone: "ok",
            detail: "",
            count: count(0),
            was: null,
            records: [],
            hidden: 0,
          },
          {
            title: "Entries waiting to post",
            tone: "bad",
            detail: "Both are over a week old.",
            count: count(2),
            was: count(3),
            records: [
              {
                resource: "journal_entry",
                id: "je_1",
                readable: true,
                label: "JE-1042",
                subtitle: "",
                status: "",
                measure: null,
              },
            ],
            hidden: 1,
          },
        ],
      }),
      block({
        type: "meter",
        title: "How close it is",
        numeral: 2,
        source: { kind: "checks", resource: "", count: 2, readable: 2, title: "" },
        meter: { clear: 1, total: 2, tones: ["ok", "bad"], percent: null },
      }),
      block({
        type: "chart",
        title: "Revenue by week",
        numeral: 3,
        state: "unreadable",
        source: { kind: "analysis", resource: "", count: 0, readable: 0, title: "Weekly revenue" },
      }),
    ],
  };
}

export function segment(text: string, fields: Partial<DeskBoardSegment> = {}): DeskBoardSegment {
  return { text, numeral: 0, tone: "", struck: false, reading: "", ...fields };
}

const LATE_LOADS = { kind: "records", resource: "shipment", count: 6, readable: 4, title: "" };

/**
 * Six late loads, placed in the conversation: three readings in the verdict
 * (one read now, one that moved since the answer, one the reader may not read
 * because two of its loads are not theirs), a list of eight rows of which one
 * is unread, and an action. A reading segment's text is its value as answered
 * (`Filled.WriteReadings`), empty where it could not be read.
 */
export function conversationBoard(): DeskBoardCard {
  const rows = Array.from({ length: 8 }, (_, index) =>
    index === 3
      ? unreadableRow()
      : shipmentRow({ id: `shp_${index + 1}`, label: `S-2411${index}` }),
  );
  return {
    ...lateBoard(),
    artifactId: "art_inline",
    title: "Six late loads on the Dallas lanes",
    place: "conversation",
    readings: [
      {
        key: "late",
        label: "Late now",
        value: count(6),
        was: null,
        source: LATE_LOADS,
        measure: "lateness",
        aggregate: "count",
      },
      {
        key: "furthest",
        label: "Furthest behind",
        value: duration(100 * 60),
        was: duration(190 * 60),
        source: LATE_LOADS,
        measure: "lateness",
        aggregate: "max",
      },
      {
        key: "owed",
        label: "Still owed on them",
        value: null,
        was: null,
        source: LATE_LOADS,
        measure: "amountDue",
        aggregate: "sum",
      },
    ],
    verdict: [
      segment("There are "),
      segment("6", { reading: "late" }),
      segment(" late loads, the furthest "),
      segment("3h 10m", { reading: "furthest" }),
      segment(" behind, with "),
      segment("", { reading: "owed" }),
      segment(" owed. "),
      segment("Four wait on one driver's hours", { numeral: 1, tone: "warn" }),
      segment("."),
    ],
    blocks: [
      block({
        type: "records",
        title: "The late loads, furthest behind first",
        numeral: 1,
        measure: "lateness",
        source: { ...LATE_LOADS, count: 8, readable: 7 },
        rows,
      }),
      block({
        type: "actions",
        actions: [
          {
            label: "Ask the Memphis dock",
            ask: "Ask the Memphis dock for an unload time",
            records: [],
          },
        ],
      }),
    ],
  };
}

/** A board as a thread's artifact summary lists it: `place` kept, the layout left out. */
export function summaryBoard(overrides: Partial<AssistantArtifact> = {}): AssistantArtifact {
  return {
    ...boardArtifact(),
    id: "art_inline",
    title: "Six late loads on the Dallas lanes",
    payload: { place: "conversation" },
    sourceToolCallId: "call_inline",
    ...overrides,
  };
}

/** The show_board call that laid the conversation board out, as its arguments were saved. */
export const CONVERSATION_SPEC = {
  title: "Six late loads on the Dallas lanes",
  place: "conversation",
  verdict:
    "There are {late} late loads, the furthest {furthest} behind, with {owed} owed. [Four wait on one driver's hours][1].",
  readings: [
    {
      key: "late",
      label: "Late now",
      records: { resource: "shipment", ids: [] },
      aggregate: "count",
    },
    {
      key: "furthest",
      label: "Furthest behind",
      records: { resource: "shipment", ids: [] },
      aggregate: "max",
    },
    {
      key: "owed",
      label: "Still owed on them",
      records: { resource: "shipment", ids: [] },
      aggregate: "sum",
    },
  ],
  blocks: [
    {
      type: "records",
      title: "The late loads, furthest behind first",
      records: { resource: "shipment", ids: ["shp_1", "shp_2", "shp_3"] },
    },
    { type: "actions", actions: [{ label: "Ask the Memphis dock", ask: "Ask the Memphis dock" }] },
  ],
};

export function boardArtifact(): AssistantArtifact {
  return {
    id: "art_board",
    threadId: "athr_1",
    messageId: "amsg_1",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "board",
    status: "Ready",
    title: "Six loads are running late on the Dallas lanes",
    payload: {
      board: {
        title: "Six loads are running late on the Dallas lanes",
        verdict: "Four are [held at the dock][1] and two are on the road.",
        blocks: [
          {
            type: "figures",
            title: "Where they stand",
            figures: [{ label: "Late now" }, { label: "Held at a stop" }, { label: "At risk" }],
          },
          {
            type: "records",
            title: "The late loads",
            half: true,
            records: { resource: "shipment", ids: ["shp_1", "shp_2", "shp_3"] },
          },
          {
            type: "note",
            title: "Call the Memphis consignee",
            half: true,
            text: "They close at five.",
          },
        ],
      },
      author: "Dispatch Desk",
    },
    sourceToolCallId: "call_board",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: "six-loads-late",
    turn: "Which Dallas loads are late and why?",
    createdAt: ANSWERED_AT,
    updatedAt: ANSWERED_AT,
  } as AssistantArtifact;
}
