import type { TriageBriefing } from "@/lib/graphql/inbox";
import { describe, expect, it } from "vitest";
import {
  carryOverRows,
  deckEntries,
  firstDayStage,
  novaLine,
  ringProgress,
  riverTicks,
  riverWindow,
  riverX,
  tipAlign,
  yourDotState,
} from "../briefing-model";

type Card = TriageBriefing["deck"][number];
type Item = Card["items"][number];
type Point = TriageBriefing["yours"][number];

const DAY = 1_791_763_200;
const HOUR = 3600;
const NOW = DAY + 13 * HOUR + 39 * 60;

function item(messageId: string, dueAt: number | null = null): Item {
  return {
    messageId,
    fromName: messageId,
    fromAddress: `${messageId}@example.com`,
    subject: "",
    classification: "Other",
    receivedAt: NOW - HOUR,
    dueAt,
    action: null,
    owner: null,
  };
}

function card(group: Card["group"], ids: string[], estimateMinutes: number): Card {
  return {
    group,
    count: ids.length,
    estimateMinutes,
    messageIds: ids,
    items: ids.slice(0, 4).map((id) => item(id)),
  };
}

function point(messageId: string, done: boolean, dueAt: number | null): Point {
  return {
    messageId,
    fromName: "",
    fromAddress: `${messageId}@example.com`,
    subject: "",
    classification: "Tender",
    receivedAt: NOW - HOUR,
    dueAt,
    done,
  };
}

describe("the ring", () => {
  it("fills as the morning's messages clear and is full when nothing waits", () => {
    expect(ringProgress(0, 8)).toBe(0);
    expect(ringProgress(6, 8)).toBeCloseTo(6 / 14);
    expect(ringProgress(6, 0)).toBe(1);
    expect(ringProgress(0, 0)).toBe(1);
  });

  it("never reads a negative count as progress", () => {
    expect(ringProgress(-3, 4)).toBe(0);
  });
});

describe("the river", () => {
  const window = riverWindow(DAY, NOW);

  it("runs from before six in the morning to now", () => {
    expect(window.end).toBe(NOW);
    expect(window.start).toBeLessThan(DAY + 6 * HOUR);
    expect(riverX(DAY + 6 * HOUR, window)).toBeGreaterThan(8);
    expect(riverX(NOW, window)).toBe(98);
  });

  it("clamps a message from before the window to the left edge and a later one to now", () => {
    expect(riverX(DAY - 3 * 24 * HOUR, window)).toBe(2);
    expect(riverX(NOW + HOUR, window)).toBe(98);
  });

  it("places messages in time order, left to right", () => {
    const xs = [8, 9, 11, 13].map((hour) => riverX(DAY + hour * HOUR, window));
    expect([...xs].sort((a, b) => a - b)).toEqual(xs);
  });

  it("keeps a window of at least two hours before six in the morning", () => {
    const early = riverWindow(DAY, DAY + 6 * HOUR + 30 * 60);
    expect(early.end - early.start).toBeGreaterThanOrEqual(2 * HOUR);
    expect(riverX(DAY + 6 * HOUR, early)).toBeGreaterThan(2);
  });

  it("marks every two hours from six and drops a mark that would crowd now", () => {
    const ticks = riverTicks(window, DAY);
    expect(ticks.map((tick) => (tick.at - DAY) / HOUR)).toEqual([6, 8, 10, 12]);
    const crowded = riverTicks(riverWindow(DAY, DAY + 12 * HOUR + 5 * 60), DAY);
    expect(crowded.map((tick) => (tick.at - DAY) / HOUR)).toEqual([6, 8, 10]);
  });

  it("keeps a tooltip inside the river at either end", () => {
    expect(tipAlign(4)).toBe("start");
    expect(tipAlign(50)).toBe("center");
    expect(tipAlign(95)).toBe("end");
  });

  it("draws a dot as done, over its target, or still open", () => {
    expect(yourDotState(point("a", true, NOW - HOUR), NOW)).toBe("done");
    expect(yourDotState(point("b", false, NOW - 60), NOW)).toBe("over");
    expect(yourDotState(point("c", false, NOW + HOUR), NOW)).toBe("open");
    expect(yourDotState(point("d", false, null), NOW)).toBe("open");
  });
});

describe("the deck", () => {
  it("orders the cards file, draft, decide, link, held whatever order the server sent", () => {
    const deck = [
      card("Held", ["h1"], 1),
      card("Link", ["l1"], 2),
      card("File", ["f1", "f2"], 2),
      card("Decide", ["d1"], 4),
      card("Draft", ["n1"], 2),
    ];
    expect(deckEntries(deck, new Set()).map((entry) => entry.group)).toEqual([
      "File",
      "Draft",
      "Decide",
      "Link",
      "Held",
    ]);
  });

  it("leaves out what was already acted on, and a card with nothing left", () => {
    const deck = [card("File", ["f1", "f2", "f3"], 3), card("Draft", ["n1", "n2"], 4)];
    const entries = deckEntries(deck, new Set(["f1", "n1", "n2"]));
    expect(entries.map((entry) => [entry.group, entry.count, entry.messageIds])).toEqual([
      ["File", 2, ["f2", "f3"]],
    ]);
    expect(entries[0].items.map((row) => row.messageId)).toEqual(["f2", "f3"]);
  });

  it("scales the estimate to what is left and never shows less than a minute", () => {
    const [file] = deckEntries([card("File", ["f1", "f2", "f3", "f4"], 8)], new Set(["f1"]));
    expect(file.estimateMinutes).toBe(6);
    const [held] = deckEntries([card("Held", ["h1", "h2"], 1)], new Set(["h1"]));
    expect(held.estimateMinutes).toBe(1);
  });

  it("counts every message in a card, not only the ones it lists", () => {
    const ids = ["a", "b", "c", "d", "e", "f"];
    const [entry] = deckEntries([card("Decide", ids, 24)], new Set());
    expect(entry.count).toBe(6);
    expect(entry.items).toHaveLength(4);
  });
});

describe("Nova's line", () => {
  const base = {
    phase: "Day" as const,
    waiting: 8,
    handledByNovaToday: 29,
    closedByPeopleToday: 6,
    estimateMinutes: 17,
    longestWaiting: item("partner", NOW - 23 * HOUR),
    carryOver: [] as Item[],
    onCall: null,
  };

  it("says what Nova did, what is yours and who waited longest during the day", () => {
    expect(novaLine(base)).toEqual({
      kind: "day",
      handled: 29,
      waiting: 8,
      longest: base.longestWaiting,
      minutes: 17,
    });
  });

  it("says you are clear when nothing waits", () => {
    expect(novaLine({ ...base, waiting: 0, longestWaiting: null })).toEqual({ kind: "clear" });
  });

  it("wraps up at the end of the day with what carries over and who is on call", () => {
    const onCall = { id: "u1", name: "Marcus Lee", emailAddress: "", thumbnailUrl: "" };
    const carry = [item("a"), item("b")];
    expect(novaLine({ ...base, phase: "EndOfDay", carryOver: carry, onCall, waiting: 2 })).toEqual({
      kind: "endOfDay",
      handled: 29,
      closed: 6,
      carry: 2,
      onCall,
    });
  });

  it("wraps up even when nothing waits at the end of the day", () => {
    expect(novaLine({ ...base, phase: "EndOfDay", waiting: 0 })).toEqual({
      kind: "endOfDay",
      handled: 29,
      closed: 6,
      carry: 0,
      onCall: null,
    });
  });
});

describe("carrying over", () => {
  it("lists the soonest due first and the ones without a target last", () => {
    const rows = carryOverRows([item("late", NOW + HOUR), item("none"), item("over", NOW - HOUR)]);
    expect(rows.map((row) => row.messageId)).toEqual(["over", "late", "none"]);
  });
});

describe("the first day", () => {
  const empty = { total: 0, processing: 0 };

  it("waits for a first source until one is connected", () => {
    expect(
      firstDayStage({
        firstDay: true,
        connectedAt: null,
        counts: empty,
        now: NOW,
        finished: false,
      }),
    ).toBe("empty");
  });

  it("is an ordinary briefing once mail has arrived, unless the backlog is being read here", () => {
    expect(
      firstDayStage({
        firstDay: false,
        connectedAt: null,
        counts: { total: 9, processing: 2 },
        now: NOW,
        finished: false,
      }),
    ).toBe("briefing");
  });

  it("reads the backlog after a source is connected, while any of it is still being read", () => {
    const connectedAt = NOW - 5;
    expect(
      firstDayStage({ firstDay: true, connectedAt, counts: empty, now: NOW, finished: false }),
    ).toBe("reading");
    expect(
      firstDayStage({
        firstDay: false,
        connectedAt,
        counts: { total: 12, processing: 4 },
        now: NOW,
        finished: false,
      }),
    ).toBe("reading");
  });

  it("says it is done reading, then shows the first briefing", () => {
    const connectedAt = NOW - 30;
    const read = { total: 57, processing: 0 };
    expect(
      firstDayStage({ firstDay: false, connectedAt, counts: read, now: NOW, finished: false }),
    ).toBe("done");
    expect(
      firstDayStage({ firstDay: false, connectedAt, counts: read, now: NOW, finished: true }),
    ).toBe("briefing");
  });

  it("stops waiting when nothing was sitting at the address", () => {
    expect(
      firstDayStage({
        firstDay: true,
        connectedAt: NOW - 25,
        counts: empty,
        now: NOW,
        finished: false,
      }),
    ).toBe("quiet");
  });
});
