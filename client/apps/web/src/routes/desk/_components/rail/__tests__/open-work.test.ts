import type { DeskRailRow, DeskRailShelf } from "@/components/desk-chat/rail/desk-rail-rows";
import {
  DESK_SHELF_KEYS,
  deskShelfEdges,
  type DeskShelfKey,
} from "@/components/desk-chat/rail/desk-threads";
import type { DeskAttentionGroup } from "@/lib/graphql/desk-attention";
import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { railCaseRows } from "../open-work";

const NOW = Date.UTC(2026, 10, 10, 15) / 1000;
const EDGES = deskShelfEdges(NOW, "UTC");
const QUIET = { loading: false, hasMore: false, failed: false };

function group(kind: string, id: string, label: string, at = NOW, readable = true) {
  return {
    key: id,
    record: { kind, noun: kind, id, label, subtitle: "", status: "", readable },
    items: [{ occurredAt: at }],
  } as unknown as DeskAttentionGroup;
}

function thread(id: string, overrides: Partial<AssistantThread> = {}): AssistantThread {
  return {
    id,
    title: "Missing proof of delivery",
    subjectType: "Shipment",
    subjectId: "shp_1",
    agentDefinitionId: "agdef_1",
    ...overrides,
  } as AssistantThread;
}

function shelves(threads: Partial<Record<DeskShelfKey, AssistantThread[]>> = {}): DeskRailShelf[] {
  return DESK_SHELF_KEYS.map((key) => ({
    key,
    count: threads[key]?.length ?? 0,
    threads: threads[key] ?? [],
    paging: QUIET,
  }));
}

/** The rows as a reader sees them: a shelf's name, or a case's name and whether it waits. */
const drawn = (rows: DeskRailRow[]) =>
  rows.map((row) =>
    row.kind === "shelf"
      ? `# ${row.key}`
      : row.kind === "case"
        ? `${row.case.title}${row.case.waiting ? " (waiting)" : ""}`
        : row.kind,
  );

describe("railCaseRows", () => {
  it("shelves a case by the day its conversation was last touched, as conversations are", () => {
    const rows = railCaseRows(
      shelves({
        today: [thread("athr_1", { subjectId: "shp_1", title: "Where is the driver?" })],
        week: [thread("athr_2", { subjectId: "shp_2", title: "Short on the BOL" })],
      }),
      [],
      [],
      EDGES,
      NOW,
    );

    expect(drawn(rows)).toEqual(["# today", "Where is the driver?", "# week", "Short on the BOL"]);
    expect(rows[1]).toMatchObject({ kind: "case", case: { to: "/desk/r/shipment/shp_1" } });
  });

  it("shelves a record that waits with no conversation yet on the day it was raised", () => {
    const rows = railCaseRows(
      shelves(),
      [
        group("invoice", "inv_1", "INV-6", EDGES.yesterday + 60),
        group("worker", "wrk_1", "Jane Doe"),
      ],
      [],
      EDGES,
      NOW,
    );

    expect(drawn(rows)).toEqual([
      "# today",
      "Jane Doe (waiting)",
      "# yesterday",
      "INV-6 (waiting)",
    ]);
  });

  it("is one row for a record however it got here, on its conversation's shelf", () => {
    const mine = thread("athr_1");
    const rows = railCaseRows(
      shelves({ yesterday: [mine] }),
      [group("shipment", "shp_1", "SHP-1")],
      [{ kind: "shipment", id: "shp_1", label: "SHP-1" }],
      EDGES,
      NOW,
    );

    expect(drawn(rows)).toEqual(["# yesterday", "SHP-1 (waiting)"]);
    expect(rows[1]).toMatchObject({ kind: "case", case: { thread: mine } });
  });

  it("lists what was opened this session under today, and a view as a view", () => {
    const rows = railCaseRows(
      shelves(),
      [],
      [
        { kind: "shipment", id: "shp_9", label: "SHP-9" },
        { kind: "view", id: "art_1", label: "Late today", path: "/desk/v/athr_1/art_1" },
      ],
      EDGES,
      NOW,
    );

    expect(drawn(rows)).toEqual(["# today", "SHP-9", "Late today"]);
    expect(rows[2]).toMatchObject({ kind: "case", case: { view: true, recordId: "" } });
  });

  it("leaves out a record nobody may read or that has no name", () => {
    const rows = railCaseRows(
      shelves(),
      [group("shipment", "shp_1", "SHP-1", NOW, false), group("shipment", "shp_2", "")],
      [],
      EDGES,
      NOW,
    );

    expect(rows).toEqual([]);
  });

  it("keeps the place the rest of a shelf arrives", () => {
    const paged = shelves({ today: [thread("athr_1")] }).map((shelf) =>
      shelf.key === "today" ? { ...shelf, paging: { ...QUIET, hasMore: true } } : shelf,
    );

    expect(drawn(railCaseRows(paged, [], [], EDGES, NOW))).toEqual([
      "# today",
      "Missing proof of delivery",
      "more",
    ]);
  });

  it("leaves a case the person put away off the rail until it waits on them again", () => {
    const away = {
      state: "Snoozed",
      record: { label: "SEED-DET-002" },
    } as AssistantThread["case"];
    const put = thread("athr_1", { subjectId: "shp_1", title: "Detention at Phoenix", case: away });
    const kept = thread("athr_2", { subjectId: "shp_2", title: "Short on the BOL" });

    expect(drawn(railCaseRows(shelves({ today: [put, kept] }), [], [], EDGES, NOW))).toEqual([
      "# today",
      "Short on the BOL",
    ]);

    const waitingAgain = railCaseRows(
      shelves({ today: [put, kept] }),
      [group("shipment", "shp_1", "SEED-DET-002")],
      [],
      EDGES,
      NOW,
    );
    expect(drawn(waitingAgain)).toEqual(["# today", "SEED-DET-002 (waiting)", "Short on the BOL"]);
  });
});
