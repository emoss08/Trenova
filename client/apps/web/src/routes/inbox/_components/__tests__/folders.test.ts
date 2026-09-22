import { describe, expect, it } from "vitest";
import { folderFilter, folderKey, folderParams, parseFolder, type InboxFolder } from "../folders";

function params(value: string): URLSearchParams {
  return new URLSearchParams(value);
}

describe("inbox folders", () => {
  it("opens on the waiting lane when the address names no folder", () => {
    expect(parseFolder(params(""))).toEqual({ kind: "lane", lane: "waiting" });
  });

  it("reads a lane, a kind and a mailbox out of the address", () => {
    expect(parseFolder(params("lane=handled"))).toEqual({ kind: "lane", lane: "handled" });
    expect(parseFolder(params("kind=Tender"))).toEqual({
      kind: "classification",
      classification: "Tender",
    });
    expect(parseFolder(params("mailbox=imbx_01"))).toEqual({
      kind: "mailbox",
      mailboxId: "imbx_01",
    });
  });

  it("falls back to waiting rather than trusting a value it does not know", () => {
    // A link from an old notification, or a hand-edited address, must still
    // open an inbox — never a list filtered on a status the server refuses.
    expect(parseFolder(params("lane=archived"))).toEqual({ kind: "lane", lane: "waiting" });
    expect(parseFolder(params("kind=Spam"))).toEqual({ kind: "lane", lane: "waiting" });
    expect(parseFolder(params("mailbox="))).toEqual({ kind: "lane", lane: "waiting" });
  });

  it("round-trips every folder through the address and keeps the open message and search", () => {
    const folders: InboxFolder[] = [
      { kind: "lane", lane: "ignored" },
      { kind: "classification", classification: "ProofOfDelivery" },
      { kind: "mailbox", mailboxId: "imbx_02" },
    ];
    for (const folder of folders) {
      const next = folderParams(params("lane=waiting&message=imsg_9&q=pod"), folder);
      expect(parseFolder(next)).toEqual(folder);
      expect(next.get("q")).toBe("pod");
      expect(next.has("message")).toBe(false);
    }
  });

  it("asks the server for exactly the folder's slice", () => {
    expect(folderFilter({ kind: "lane", lane: "waiting" }, "")).toEqual({
      statuses: ["InReview", "Quarantined"],
      classification: null,
      mailboxId: null,
      query: null,
    });
    expect(folderFilter({ kind: "classification", classification: "Invoice" }, "  acme ")).toEqual({
      statuses: [],
      classification: "Invoice",
      mailboxId: null,
      query: "acme",
    });
    expect(folderFilter({ kind: "mailbox", mailboxId: "imbx_03" }, "")).toEqual({
      statuses: [],
      classification: null,
      mailboxId: "imbx_03",
      query: null,
    });
  });

  it("gives every folder a distinct key", () => {
    const keys = [
      folderKey({ kind: "lane", lane: "waiting" }),
      folderKey({ kind: "lane", lane: "all" }),
      folderKey({ kind: "classification", classification: "Tender" }),
      folderKey({ kind: "mailbox", mailboxId: "imbx_01" }),
    ];
    expect(new Set(keys).size).toBe(keys.length);
  });
});
