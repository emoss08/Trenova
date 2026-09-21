import type { AssistantMessage } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import type { ThreadEntry } from "../thread-view";
import { arrivedSince, highestSequence, withDayMarkers } from "../thread-rows";

const HOUR = 60 * 60;
const DAY = 24 * HOUR;
// 2026-09-21 12:00 UTC
const NOON = 1_789_992_000;

function message(sequence: number, createdAt: number): AssistantMessage {
  return {
    id: `amsg_${sequence}`,
    threadId: "thr_1",
    sequence,
    role: "User",
    content: "",
    toolCalls: null,
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    createdAt,
  };
}

function entry(sequence: number, createdAt: number): ThreadEntry {
  return { kind: "user", message: message(sequence, createdAt) };
}

describe("withDayMarkers", () => {
  it("puts one marker before the first message of each day", () => {
    const rows = withDayMarkers(
      [
        entry(0, NOON - 2 * DAY),
        entry(1, NOON - 2 * DAY + HOUR),
        entry(2, NOON - DAY),
        entry(3, NOON),
      ],
      NOON,
      "UTC",
    );

    expect(rows.map((row) => row.kind)).toEqual([
      "day",
      "entry",
      "entry",
      "day",
      "entry",
      "day",
      "entry",
    ]);
    const markers = rows.filter((row) => row.kind === "day");
    expect(markers.map((marker) => marker.daysAgo)).toEqual([2, 1, 0]);
  });

  // 23:50 the night before is yesterday to the reader, however few hours ago.
  it("cuts days on the reader's calendar, not on 24-hour spans", () => {
    const lateLastNight = NOON - 12 * HOUR - 10 * 60;
    const rows = withDayMarkers([entry(0, lateLastNight), entry(1, NOON)], NOON, "UTC");

    expect(rows.filter((row) => row.kind === "day")).toHaveLength(2);
  });

  it("skips a marker for a message with no timestamp", () => {
    const rows = withDayMarkers([entry(0, 0), entry(1, NOON)], NOON, "UTC");

    expect(rows.map((row) => row.kind)).toEqual(["entry", "day", "entry"]);
  });
});

describe("arrivedSince", () => {
  // A windowed list mounts rows as they scroll into view and a page of history
  // mounts a screenful at once; neither is an arrival. Only what is numbered
  // past the thread as it was opened rises.
  it("marks only messages numbered past the opening point", () => {
    const messages = [message(3, NOON), message(4, NOON), message(5, NOON)];

    expect(arrivedSince(4, messages)).toEqual(new Set(["amsg_5"]));
    expect(arrivedSince(5, messages).size).toBe(0);
  });

  it("reads the opening point off the thread", () => {
    expect(highestSequence([message(2, NOON), message(9, NOON), message(4, NOON)])).toBe(9);
    expect(highestSequence([])).toBe(-1);
  });
});
