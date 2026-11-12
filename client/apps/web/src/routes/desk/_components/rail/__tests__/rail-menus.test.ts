import type { AssistantThread } from "@/types/assistant";
import { isItem, normalizeEntries } from "@trenova/shared/components/context-menu/menu-spec";
import { describe, expect, it, vi } from "vitest";
import type { RailCaseRow } from "../open-work";
import { threadMenu, workMenu, type RailThreadActions, type RailWorkActions } from "../rail-menus";

const t = (message: string | null | undefined) => message ?? "";

function thread(overrides: Partial<AssistantThread> = {}): AssistantThread {
  return {
    id: "athr_1",
    title: "Late loads today",
    pinned: false,
    ...overrides,
  } as AssistantThread;
}

function threadActions(overrides: Partial<RailThreadActions> = {}): RailThreadActions {
  return {
    open: vi.fn(),
    openBeside: vi.fn(),
    openInNewTab: vi.fn(),
    copyLink: vi.fn(),
    setPinned: vi.fn(),
    rename: vi.fn(),
    download: vi.fn(),
    remove: vi.fn(),
    ...overrides,
  };
}

function workActions(overrides: Partial<RailWorkActions> = {}): RailWorkActions {
  return {
    open: vi.fn(),
    openInNewTab: vi.fn(),
    copyLink: vi.fn(),
    recordPath: () => "/shipments?item=shp_1",
    forget: vi.fn(),
    ...overrides,
  };
}

function work(overrides: Partial<RailCaseRow> = {}): RailCaseRow {
  return {
    to: "/desk/r/shipment/shp_1",
    recordId: "shp_1",
    title: "SHP-1",
    waiting: false,
    view: false,
    thread: null,
    ...overrides,
  };
}

const labels = (spec: { entries: Parameters<typeof normalizeEntries>[0] }) =>
  normalizeEntries(spec.entries)
    .filter(isItem)
    .map((entry) => entry.label);

describe("a conversation's menu on the rail", () => {
  it("offers what can be done to it, with delete last", () => {
    const spec = threadMenu(t, thread(), threadActions());

    // Named for assistive technology, with no heading drawn over the row it opened on.
    expect(spec.label).toBe("Late loads today");
    expect(spec.title).toBeUndefined();
    expect(spec.subtitle).toBeUndefined();
    expect(labels(spec)).toEqual([
      "Open",
      "Open beside the page",
      "Open in a new tab",
      "Pin conversation",
      "Rename",
      "Download transcript",
      "Copy link",
      "Delete conversation",
    ]);
  });

  it("unpins a pinned conversation and leaves out the margin where it would not open there", () => {
    const actions = threadActions({ openBeside: undefined });
    const pinned = thread({ pinned: true });
    const spec = threadMenu(t, pinned, actions);

    expect(labels(spec)).not.toContain("Open beside the page");
    normalizeEntries(spec.entries)
      .filter(isItem)
      .find((entry) => entry.label === "Unpin conversation")
      ?.onSelect?.();

    expect(actions.setPinned).toHaveBeenCalledWith(pinned, false);
  });
});

describe("a case conversation's menu on the rail", () => {
  it("opens the case first, and the conversation on its own second", () => {
    const actions = threadActions();
    const spec = threadMenu(t, thread({ subjectType: "Shipment", subjectId: "shp_1" }), actions);

    expect(labels(spec).slice(0, 2)).toEqual(["Open the case", "Open the conversation"]);
    normalizeEntries(spec.entries)
      .filter(isItem)
      .find((entry) => entry.label === "Open the case")
      ?.onSelect?.();

    expect(actions.open).toHaveBeenCalledWith("/desk/r/shipment/shp_1");
  });
});

describe("a case's menu on the rail", () => {
  it("opens the case or its record, and takes a row the session added off the list", () => {
    const actions = workActions();
    const row = work();
    const spec = workMenu(t, row, actions);

    expect(spec.label).toBe(row.title);
    expect(spec.title).toBeUndefined();
    expect(labels(spec)).toEqual([
      "Open",
      "Open in a new tab",
      "Open the record",
      "Copy link",
      "Remove from Cases",
    ]);

    normalizeEntries(spec.entries)
      .filter(isItem)
      .find((entry) => entry.label === "Remove from Cases")
      ?.onSelect?.();
    expect(actions.forget).toHaveBeenCalledWith(row);
  });

  it("cannot take off a case that waits on the person, and offers no record a view has none of", () => {
    const waiting = workMenu(t, work({ waiting: true }), workActions());
    const view = workMenu(
      t,
      work({ view: true, recordId: "", to: "/desk/v/athr_1/art_1" }),
      workActions({ recordPath: () => null }),
    );

    expect(waiting.title).toBeUndefined();
    expect(labels(waiting)).not.toContain("Remove from Cases");
    expect(labels(view)).not.toContain("Open the record");
  });
});
