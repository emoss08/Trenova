import type {
  InboundMessageCounts,
  InboundMessageFilter,
  InboundMessageRow,
} from "@/lib/graphql/inbox";
import { inbox, type InboxFolderPages } from "@/lib/queries/inbox";
import {
  adjustCounts,
  applyRows,
  folderMembership,
  insertIntoPages,
  placeInFolder,
  removeFromPages,
} from "@/lib/triage/cache";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { messageRow } from "./message-fixture";

const me = { meId: "usr_me", now: 1_003_600 };
const waiting: InboundMessageFilter = { lane: "Waiting", sort: "Due", collapseThreads: true };
const newest: InboundMessageFilter = { lane: "Everything", sort: "Newest", collapseThreads: true };

function pages(...chunks: Array<{ rows: InboundMessageRow[]; more?: boolean }>): InboxFolderPages {
  return {
    pages: chunks.map((chunk, index) => ({
      messages: chunk.rows,
      endCursor: chunk.more === true ? `cursor-${index}` : null,
      hasNextPage: chunk.more === true,
    })),
    pageParams: chunks.map((_, index) => (index === 0 ? null : `cursor-${index - 1}`)),
  };
}

function ids(data: InboxFolderPages): string[][] {
  return data.pages.map((page) => page.messages.map((message) => message.id));
}

function counts(overrides: Partial<InboundMessageCounts> = {}): InboundMessageCounts {
  return {
    waiting: 0,
    waitingOnThem: 0,
    snoozed: 0,
    assignedToMe: 0,
    overdue: 0,
    processing: 0,
    handled: 0,
    ignored: 0,
    quarantined: 0,
    total: 0,
    byClassification: [{ classification: "Tender", total: 0, waiting: 0 }],
    byMailbox: [],
    ...overrides,
  };
}

describe("folderMembership", () => {
  it("reads each lane the way the server does", () => {
    const snoozed = messageRow({
      snooze: {
        until: 2_000_000,
        event: null,
        watchShipment: null,
        documentTypeId: null,
        documentTypeName: null,
        snoozedAt: 1,
        snoozedBy: null,
      },
    });
    const awaiting = messageRow({ awaitingReply: { since: 1, until: 2 } });
    const mine = messageRow({ ownerId: "usr_me" });
    const quarantined = messageRow({ status: "Quarantined" });
    const handled = messageRow({ status: "Actioned", needsReview: false });

    expect(folderMembership(quarantined, { lane: "Waiting" }, me)).toBe(true);
    expect(folderMembership(snoozed, { lane: "Waiting" }, me)).toBe(false);
    expect(folderMembership(snoozed, { lane: "Snoozed" }, me)).toBe(true);
    expect(folderMembership(awaiting, { lane: "Waiting" }, me)).toBe(false);
    expect(folderMembership(awaiting, { lane: "WaitingOnThem" }, me)).toBe(true);
    expect(folderMembership(mine, { lane: "AssignedToMe" }, me)).toBe(true);
    expect(folderMembership(mine, { lane: "AssignedToMe" }, { ...me, meId: "usr_else" })).toBe(
      false,
    );
    expect(folderMembership(handled, { lane: "Handled" }, me)).toBe(true);
    expect(folderMembership(handled, { lane: "Waiting" }, me)).toBe(false);
    expect(folderMembership(handled, { lane: "Everything" }, me)).toBe(true);
  });

  it("is unknown when the folder also searches, but false whenever a part it can judge fails", () => {
    const row = messageRow();
    expect(folderMembership(row, { lane: "Waiting", query: "acme" }, me)).toBe("unknown");
    expect(folderMembership(row, { lane: "Handled", query: "acme" }, me)).toBe(false);
    expect(
      folderMembership(row, { lane: "Waiting", classification: "Invoice", query: "x" }, me),
    ).toBe(false);
    expect(folderMembership(row, { lane: "Waiting", query: "   " }, me)).toBe(true);
  });
});

describe("insertIntoPages", () => {
  it("puts a message where the due-first order puts it, the overdue first", () => {
    const data = pages({
      rows: [
        messageRow({ id: "a", dueAt: 100 }),
        messageRow({ id: "b", dueAt: 300, threadKey: "b" }),
      ],
    });
    const arriving = messageRow({ id: "n", dueAt: 200, threadKey: "n" });

    expect(ids(insertIntoPages(data, arriving, waiting))).toEqual([["a", "n", "b"]]);
  });

  it("leaves a message that sorts past the loaded rows for the next page to bring", () => {
    const data = pages({ rows: [messageRow({ id: "a", dueAt: 100 })], more: true });
    const late = messageRow({ id: "n", dueAt: 900, threadKey: "n" });

    expect(insertIntoPages(data, late, waiting)).toBe(data);
  });

  it("appends to the last page when there is nothing more to load", () => {
    const data = pages({ rows: [messageRow({ id: "a", dueAt: 100 })] });
    const late = messageRow({ id: "n", dueAt: 900, threadKey: "n" });

    expect(ids(insertIntoPages(data, late, waiting))).toEqual([["a", "n"]]);
  });

  it("lets the newer message of a thread stand for it in a collapsed folder", () => {
    const older = messageRow({ id: "old", receivedAt: 10, threadKey: "t" });
    const data = pages({ rows: [older] });
    const reply = messageRow({ id: "reply", receivedAt: 20, threadKey: "t" });

    expect(ids(insertIntoPages(data, reply, newest))).toEqual([["reply"]]);
    const stale = messageRow({ id: "stale", receivedAt: 5, threadKey: "t" });
    expect(insertIntoPages(data, stale, newest)).toBe(data);
  });

  it("orders newest first when the folder is read newest first", () => {
    const data = pages(
      { rows: [messageRow({ id: "b", receivedAt: 30, threadKey: "b" })] },
      {
        rows: [messageRow({ id: "a", receivedAt: 10, threadKey: "a" })],
      },
    );
    const middle = messageRow({ id: "m", receivedAt: 20, threadKey: "m" });

    expect(ids(insertIntoPages(data, middle, newest))).toEqual([["b"], ["m", "a"]]);
  });
});

describe("placeInFolder", () => {
  it("updates a message that still belongs, removes one that does not, adds one that now does", () => {
    const row = messageRow({ id: "a" });
    const data = pages({ rows: [row] });

    const updated = placeInFolder(data, { ...row, subject: "Changed" }, waiting, me);
    expect(updated.data.pages[0].messages[0].subject).toBe("Changed");
    expect(updated.inserted).toBe(false);

    const handled = placeInFolder(
      data,
      { ...row, status: "Actioned", needsReview: false },
      waiting,
      me,
    );
    expect(ids(handled.data)).toEqual([[]]);
    expect(handled.removed).toBe(true);

    const fresh = placeInFolder(pages({ rows: [] }), row, waiting, me);
    expect(ids(fresh.data)).toEqual([["a"]]);
    expect(fresh.inserted).toBe(true);
  });

  it("keeps but never adds a message in a searched folder", () => {
    const row = messageRow({ id: "a" });
    const searched = { ...waiting, query: "acme" };

    expect(placeInFolder(pages({ rows: [] }), row, searched, me).inserted).toBe(false);
    const kept = placeInFolder(pages({ rows: [row] }), { ...row, subject: "x" }, searched, me);
    expect(kept.data.pages[0].messages[0].subject).toBe("x");
  });

  it("returns the same pages when nothing about them changed", () => {
    const data = pages({ rows: [messageRow({ id: "a" })] });
    expect(removeFromPages(data, new Set(["zzz"]))).toBe(data);
  });
});

describe("adjustCounts", () => {
  it("moves a message from waiting to handled, by kind and by source, without going negative", () => {
    const before = messageRow({ ownerId: "usr_me", dueAt: 1_000_000 });
    const after = { ...before, status: "Actioned" as const, needsReview: false, dueAt: null };
    const start = counts({
      waiting: 1,
      assignedToMe: 1,
      overdue: 1,
      total: 1,
      byClassification: [{ classification: "Tender", total: 1, waiting: 1 }],
      byMailbox: [{ mailboxId: "imbx_1", total: 1, waiting: 1 }],
    });

    expect(adjustCounts(start, before, after, me)).toEqual(
      counts({
        waiting: 0,
        assignedToMe: 0,
        overdue: 0,
        handled: 1,
        total: 1,
        byClassification: [{ classification: "Tender", total: 1, waiting: 0 }],
        byMailbox: [{ mailboxId: "imbx_1", total: 1, waiting: 0 }],
      }),
    );
    expect(adjustCounts(counts(), before, undefined, me).waiting).toBe(0);
  });

  it("counts an arrival on a source with no mail yet", () => {
    const next = adjustCounts(counts(), undefined, messageRow({ mailboxId: "imbx_new" }), me);
    expect(next.byMailbox).toEqual([{ mailboxId: "imbx_new", total: 1, waiting: 1 }]);
    expect(next.waiting).toBe(1);
  });
});

describe("applyRows", () => {
  it("draws a change in every folder, the counts and the reader, and puts it back from the before rows", () => {
    const client = new QueryClient();
    const row = messageRow({ id: "a" });
    const waitingKey = inbox.messages(waiting).queryKey;
    const handledKey = inbox.messages({
      lane: "Handled",
      sort: "Newest",
      collapseThreads: true,
    }).queryKey;
    client.setQueryData(waitingKey, pages({ rows: [row] }));
    client.setQueryData(handledKey, pages({ rows: [] }));
    client.setQueryData(inbox.counts().queryKey, counts({ waiting: 1, total: 1 }));
    client.setQueryData(inbox.message("a").queryKey, { ...row, textBody: "Hello" });

    const handled = { ...row, status: "Actioned" as const, needsReview: false, dueAt: null };
    applyRows(client, [handled], new Map([["a", row]]), me);

    expect(ids(client.getQueryData<InboxFolderPages>(waitingKey)!)).toEqual([[]]);
    expect(ids(client.getQueryData<InboxFolderPages>(handledKey)!)).toEqual([["a"]]);
    expect(client.getQueryData<InboundMessageCounts>(inbox.counts().queryKey)).toMatchObject({
      waiting: 0,
      handled: 1,
    });
    expect(client.getQueryData(inbox.message("a").queryKey)).toMatchObject({
      status: "Actioned",
      textBody: "Hello",
    });

    applyRows(client, [row], new Map([["a", handled]]), me);

    expect(ids(client.getQueryData<InboxFolderPages>(waitingKey)!)).toEqual([["a"]]);
    expect(ids(client.getQueryData<InboxFolderPages>(handledKey)!)).toEqual([[]]);
    expect(client.getQueryData<InboundMessageCounts>(inbox.counts().queryKey)).toMatchObject({
      waiting: 1,
      handled: 0,
    });
  });
});
