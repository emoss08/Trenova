import type { InboundMessageCounts, TriageView } from "@/lib/graphql/inbox";
import { describe, expect, it } from "vitest";
import {
  BRIEFING,
  folderFilter,
  folderHasWaiting,
  folderKey,
  folderParams,
  messageParams,
  parseFolder,
  parseTriageLocation,
  resolveSort,
  searchParamsWith,
  type TriageFolder,
} from "../folders";

function params(value: string): URLSearchParams {
  return new URLSearchParams(value);
}

const view: TriageView = { id: "tv_1", name: "Acme", query: "acme", position: 0, waiting: 2 };

describe("triage folders in the address", () => {
  it("lands on the Briefing when the address names nothing", () => {
    expect(parseFolder(params(""))).toEqual(BRIEFING);
    expect(parseFolder(params("brief="))).toEqual(BRIEFING);
  });

  it("opens Everything for a link that names only a message, so the message is in the list", () => {
    expect(parseTriageLocation(params("message=imsg_9"))).toMatchObject({
      folder: { kind: "lane", lane: "all" },
      messageId: "imsg_9",
    });
  });

  it("reads every kind of folder, and the old inbox's lane words", () => {
    expect(parseFolder(params("rules="))).toEqual({ kind: "rules" });
    expect(parseFolder(params("lane=waiting"))).toEqual({ kind: "lane", lane: "waiting" });
    expect(parseFolder(params("lane=them"))).toEqual({ kind: "lane", lane: "them" });
    expect(parseFolder(params("lane=mine"))).toEqual({ kind: "lane", lane: "mine" });
    expect(parseFolder(params("lane=snoozed"))).toEqual({ kind: "lane", lane: "snoozed" });
    expect(parseFolder(params("lane=all"))).toEqual({ kind: "lane", lane: "all" });
    expect(parseFolder(params("kind=Tender"))).toEqual({
      kind: "classification",
      classification: "Tender",
    });
    expect(parseFolder(params("mailbox=imbx_01"))).toEqual({
      kind: "mailbox",
      mailboxId: "imbx_01",
    });
    expect(parseFolder(params("view=tv_1"))).toEqual({ kind: "view", viewId: "tv_1" });
  });

  it("falls back rather than trusting a value it does not know", () => {
    expect(parseFolder(params("lane=archived"))).toEqual(BRIEFING);
    expect(parseFolder(params("kind=Spam"))).toEqual(BRIEFING);
    expect(parseFolder(params("mailbox=&view="))).toEqual(BRIEFING);
  });

  it("keeps no open message or search on the Briefing or Rules", () => {
    expect(parseTriageLocation(params("rules=&message=imsg_1&q=acme"))).toMatchObject({
      messageId: null,
      search: "",
    });
  });

  it("round-trips every folder, dropping the open message and keeping the search inside the queue", () => {
    const folders: TriageFolder[] = [
      { kind: "lane", lane: "ignored" },
      { kind: "lane", lane: "them" },
      { kind: "classification", classification: "ProofOfDelivery" },
      { kind: "mailbox", mailboxId: "imbx_02" },
      { kind: "view", viewId: "tv_3" },
      { kind: "rules" },
      BRIEFING,
    ];
    for (const folder of folders) {
      const next = folderParams(params("lane=waiting&message=imsg_9&q=pod&sort=newest"), folder);
      expect(parseFolder(next)).toEqual(folder);
      expect(next.has("message")).toBe(false);
      expect(next.get("sort")).toBe("newest");
      expect(next.get("q")).toBe(
        folder.kind === "rules" || folder.kind === "briefing" ? null : "pod",
      );
    }
  });

  it("sets and clears the open message and the search without touching the folder", () => {
    const opened = messageParams(params("lane=mine"), "imsg_4");
    expect(opened.toString()).toBe("lane=mine&message=imsg_4");
    expect(messageParams(opened, null).toString()).toBe("lane=mine");
    expect(searchParamsWith(params("lane=mine&q=x"), "  ").has("q")).toBe(false);
  });

  it("gives every folder a distinct key", () => {
    const keys = [
      folderKey(BRIEFING),
      folderKey({ kind: "rules" }),
      folderKey({ kind: "lane", lane: "waiting" }),
      folderKey({ kind: "lane", lane: "all" }),
      folderKey({ kind: "classification", classification: "Tender" }),
      folderKey({ kind: "mailbox", mailboxId: "imbx_01" }),
      folderKey({ kind: "view", viewId: "imbx_01" }),
    ];
    expect(new Set(keys).size).toBe(keys.length);
  });
});

describe("folderFilter", () => {
  const base = { search: "", sort: "Due" as const, views: [view] };

  it("asks for one row per thread in the folder's lane", () => {
    expect(folderFilter({ kind: "lane", lane: "them" }, base)).toEqual({
      sort: "Due",
      collapseThreads: true,
      lane: "WaitingOnThem",
      query: null,
    });
  });

  it("reads a kind as what is waiting of that kind, and a source as all of its mail", () => {
    expect(
      folderFilter(
        { kind: "classification", classification: "Invoice" },
        { ...base, search: "  acme " },
      ),
    ).toEqual({
      sort: "Due",
      collapseThreads: true,
      lane: "Waiting",
      classification: "Invoice",
      query: "acme",
    });
    expect(folderFilter({ kind: "mailbox", mailboxId: "imbx_3" }, base)).toMatchObject({
      lane: "Everything",
      mailboxId: "imbx_3",
    });
  });

  it("searches a view's words and the typed words together, and asks nothing for a view that is gone", () => {
    expect(folderFilter({ kind: "view", viewId: "tv_1" }, { ...base, search: "PO" })).toMatchObject(
      {
        lane: "Everything",
        query: "acme PO",
      },
    );
    expect(folderFilter({ kind: "view", viewId: "tv_gone" }, base)).toBeNull();
  });

  it("treats a search that has not arrived as no search", () => {
    const missing = { ...base, search: undefined as unknown as string };
    expect(folderFilter({ kind: "lane", lane: "waiting" }, missing)).toMatchObject({ query: null });
  });
});

describe("sorting a folder", () => {
  const counts: InboundMessageCounts = {
    waiting: 3,
    waitingOnThem: 0,
    snoozed: 1,
    assignedToMe: 0,
    overdue: 1,
    processing: 0,
    handled: 9,
    ignored: 2,
    quarantined: 0,
    total: 15,
    byClassification: [
      { classification: "Tender", total: 4, waiting: 2 },
      { classification: "Invoice", total: 5, waiting: 0 },
    ],
    byMailbox: [{ mailboxId: "imbx_1", total: 10, waiting: 0 }],
  };

  it("offers due first only where something waits", () => {
    expect(folderHasWaiting({ kind: "lane", lane: "waiting" }, counts, [view])).toBe(true);
    expect(folderHasWaiting({ kind: "lane", lane: "them" }, counts, [view])).toBe(false);
    expect(folderHasWaiting({ kind: "lane", lane: "handled" }, counts, [view])).toBe(false);
    expect(folderHasWaiting({ kind: "lane", lane: "all" }, counts, [view])).toBe(true);
    expect(
      folderHasWaiting({ kind: "classification", classification: "Invoice" }, counts, [view]),
    ).toBe(false);
    expect(folderHasWaiting({ kind: "mailbox", mailboxId: "imbx_1" }, counts, [view])).toBe(false);
    expect(folderHasWaiting({ kind: "view", viewId: "tv_1" }, counts, [view])).toBe(true);
  });

  it("reads due first by default, newest when asked or when nothing waits", () => {
    expect(resolveSort(null, true)).toBe("Due");
    expect(resolveSort("newest", true)).toBe("Newest");
    expect(resolveSort("due", false)).toBe("Newest");
  });
});
