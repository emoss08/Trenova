import type {
  InboundMessageCounts,
  InboundMessageDetail,
  InboundMessageFilter,
  InboundMessagePage,
  InboundMessageRow,
  TriageLane,
} from "@/lib/graphql/inbox";
import { inbox, type InboxFolderPages } from "@/lib/queries/inbox";
import type { QueryClient } from "@tanstack/react-query";
import { isAwaitingReply, isSnoozed, isWaiting } from "@/lib/triage/message-state";

/**
 * Keeping the cached queue true without asking the server for it again.
 *
 * A write a person makes is drawn at once (optimistic), the server's answer
 * replaces the guess, an Undo or a refusal puts the message back as it was,
 * and a realtime notice places a changed message where it now belongs. All
 * four are the same operation — "this is what the message looks like now" —
 * applied to every cached folder, the counts and the message itself, so they
 * cannot disagree about where a message is.
 */

/** Who is looking and when, for the lanes that depend on it. */
export type MembershipContext = { meId: string | null; now: number };

/**
 * Whether a message belongs in a folder, as far as the client can tell.
 * "unknown" when every part the client can judge holds but the folder also
 * filters on something only the server can (search words, one thread, a
 * shipment, a date) — such a folder keeps a message it already shows and is
 * not given one it does not.
 */
export type Membership = boolean | "unknown";

export function laneHolds(
  message: InboundMessageRow,
  lane: TriageLane,
  meId: string | null,
): boolean {
  switch (lane) {
    case "Waiting":
      return isWaiting(message);
    case "WaitingOnThem":
      return isAwaitingReply(message);
    case "AssignedToMe":
      return isWaiting(message) && meId !== null && message.ownerId === meId;
    case "Snoozed":
      return isSnoozed(message);
    case "Handled":
      return message.status === "Actioned";
    case "Ignored":
      return message.status === "Ignored";
    case "Everything":
      return true;
  }
}

export function folderMembership(
  message: InboundMessageRow,
  filter: InboundMessageFilter,
  { meId }: MembershipContext,
): Membership {
  if (filter.lane && !laneHolds(message, filter.lane, meId)) {
    return false;
  }
  if (filter.classification && message.classification !== filter.classification) {
    return false;
  }
  if (filter.mailboxId && message.mailboxId !== filter.mailboxId) {
    return false;
  }
  if (filter.ownerId && message.ownerId !== filter.ownerId) {
    return false;
  }
  if (filter.statuses && filter.statuses.length > 0 && !filter.statuses.includes(message.status)) {
    return false;
  }
  if (
    (filter.query && filter.query.trim() !== "") ||
    filter.threadKey ||
    filter.shipmentId ||
    filter.since
  ) {
    return "unknown";
  }

  return true;
}

function newestFirst(a: InboundMessageRow, b: InboundMessageRow): number {
  if (a.receivedAt !== b.receivedAt) {
    return b.receivedAt - a.receivedAt;
  }

  return a.id < b.id ? 1 : a.id > b.id ? -1 : 0;
}

/**
 * The server's order: due first puts the soonest response target at the top
 * (the overdue ones first, since theirs has passed) and anything without one
 * after; newest is by arrival. Ties go to the newer message.
 */
export function compareForFolder(
  a: InboundMessageRow,
  b: InboundMessageRow,
  filter: InboundMessageFilter,
): number {
  if (filter.sort === "Due") {
    const aDue = a.dueAt ?? Number.POSITIVE_INFINITY;
    const bDue = b.dueAt ?? Number.POSITIVE_INFINITY;
    if (aDue !== bDue) {
      return aDue < bDue ? -1 : 1;
    }
  }

  return newestFirst(a, b);
}

export function findInPages(
  data: InboxFolderPages | undefined,
  id: string,
): InboundMessageRow | undefined {
  if (data === undefined) {
    return undefined;
  }
  for (const page of data.pages) {
    const found = page.messages.find((message) => message.id === id);
    if (found !== undefined) {
      return found;
    }
  }

  return undefined;
}

function mapPages(
  data: InboxFolderPages,
  map: (messages: InboundMessageRow[]) => InboundMessageRow[],
): InboxFolderPages {
  let changed = false;
  const pages = data.pages.map((page): InboundMessagePage => {
    const messages = map(page.messages);
    if (messages === page.messages) {
      return page;
    }
    changed = true;
    return { ...page, messages };
  });

  return changed ? { ...data, pages } : data;
}

/** The pages without these messages; the same object when none was there. */
export function removeFromPages(
  data: InboxFolderPages,
  ids: ReadonlySet<string>,
): InboxFolderPages {
  return mapPages(data, (messages) =>
    messages.some((message) => ids.has(message.id))
      ? messages.filter((message) => !ids.has(message.id))
      : messages,
  );
}

/** The pages with a message's row replaced; the same object when it was not there. */
export function replaceInPages(
  data: InboxFolderPages,
  message: InboundMessageRow,
): InboxFolderPages {
  return mapPages(data, (messages) => {
    const index = messages.findIndex((row) => row.id === message.id);
    if (index === -1 || messages[index] === message) {
      return messages;
    }
    const next = [...messages];
    next[index] = message;
    return next;
  });
}

/**
 * The pages with a message added where the server would list it. A message
 * that sorts past the last loaded row of a folder with more to load is left
 * for that page to bring, so a row never appears out of order. In a folder
 * collapsed to threads, the newer of two messages of one thread stands for it.
 */
export function insertIntoPages(
  data: InboxFolderPages,
  message: InboundMessageRow,
  filter: InboundMessageFilter,
): InboxFolderPages {
  if (data.pages.length === 0) {
    return data;
  }

  let current = data;
  if (filter.collapseThreads) {
    const sibling = data.pages
      .flatMap((page) => page.messages)
      .find((row) => row.threadKey === message.threadKey && row.id !== message.id);
    if (sibling !== undefined) {
      if (sibling.receivedAt > message.receivedAt) {
        return data;
      }
      current = removeFromPages(data, new Set([sibling.id]));
    }
  }

  const pages = current.pages;
  for (let pageIndex = 0; pageIndex < pages.length; pageIndex += 1) {
    const page = pages[pageIndex];
    const at = page.messages.findIndex((row) => compareForFolder(message, row, filter) < 0);
    if (at !== -1) {
      const messages = [...page.messages];
      messages.splice(at, 0, message);
      return { ...current, pages: pages.map((p, i) => (i === pageIndex ? { ...p, messages } : p)) };
    }
  }

  const lastIndex = pages.length - 1;
  const last = pages[lastIndex];
  if (last.hasNextPage) {
    return current;
  }

  return {
    ...current,
    pages: pages.map((p, i) => (i === lastIndex ? { ...p, messages: [...p.messages, message] } : p)),
  };
}

export type Placement = {
  data: InboxFolderPages;
  /** The message was added to a folder that did not show it. */
  inserted: boolean;
  /** The message was taken out of a folder that showed it. */
  removed: boolean;
};

/**
 * Puts one message where it now belongs in one cached folder: updated where
 * it stays, taken out where it no longer belongs, added where it now does.
 */
export function placeInFolder(
  data: InboxFolderPages,
  message: InboundMessageRow,
  filter: InboundMessageFilter,
  context: MembershipContext,
): Placement {
  const membership = folderMembership(message, filter, context);
  const present = findInPages(data, message.id) !== undefined;

  if (membership === false) {
    return present
      ? { data: removeFromPages(data, new Set([message.id])), inserted: false, removed: true }
      : { data, inserted: false, removed: false };
  }
  if (present) {
    return { data: replaceInPages(data, message), inserted: false, removed: false };
  }
  if (membership === "unknown") {
    return { data, inserted: false, removed: false };
  }

  const next = insertIntoPages(data, message, filter);

  return { data: next, inserted: next !== data, removed: false };
}

type CountFields = Pick<
  InboundMessageRow,
  | "status"
  | "needsReview"
  | "snooze"
  | "awaitingReply"
  | "ownerId"
  | "dueAt"
  | "classification"
  | "mailboxId"
>;

const BEING_READ = new Set(["Received", "Processing", "Classified"]);

function addTo(
  counts: InboundMessageCounts,
  message: CountFields,
  sign: 1 | -1,
  { meId, now }: MembershipContext,
): InboundMessageCounts {
  const waiting = isWaiting(message);
  const step = (value: number, holds: boolean) => Math.max(0, value + (holds ? sign : 0));

  return {
    ...counts,
    waiting: step(counts.waiting, waiting),
    waitingOnThem: step(counts.waitingOnThem, isAwaitingReply(message)),
    snoozed: step(counts.snoozed, isSnoozed(message)),
    assignedToMe: step(counts.assignedToMe, waiting && meId !== null && message.ownerId === meId),
    overdue: step(counts.overdue, waiting && message.dueAt !== null && message.dueAt < now),
    processing: step(counts.processing, BEING_READ.has(message.status)),
    handled: step(counts.handled, message.status === "Actioned"),
    ignored: step(counts.ignored, message.status === "Ignored"),
    quarantined: step(counts.quarantined, message.status === "Quarantined"),
    total: step(counts.total, true),
    byClassification: counts.byClassification.map((row) =>
      row.classification === message.classification
        ? { ...row, total: step(row.total, true), waiting: step(row.waiting, waiting) }
        : row,
    ),
    byMailbox: counts.byMailbox.some((row) => row.mailboxId === message.mailboxId)
      ? counts.byMailbox.map((row) =>
          row.mailboxId === message.mailboxId
            ? { ...row, total: step(row.total, true), waiting: step(row.waiting, waiting) }
            : row,
        )
      : sign === 1
        ? [...counts.byMailbox, { mailboxId: message.mailboxId, total: 1, waiting: waiting ? 1 : 0 }]
        : counts.byMailbox,
  };
}

/**
 * The counts once a message has moved from one state to another. Either side
 * may be missing: a message that arrived has no before, one that was deleted
 * or became invisible has no after.
 */
export function adjustCounts(
  counts: InboundMessageCounts,
  before: CountFields | undefined,
  after: CountFields | undefined,
  context: MembershipContext,
): InboundMessageCounts {
  let next = counts;
  if (before !== undefined) {
    next = addTo(next, before, -1, context);
  }
  if (after !== undefined) {
    next = addTo(next, after, 1, context);
  }

  return next;
}

/** Every cached folder, with the filter it was asked with. */
export function cachedFolders(
  queryClient: QueryClient,
): Array<{ queryKey: readonly unknown[]; filter: InboundMessageFilter; data: InboxFolderPages }> {
  return queryClient
    .getQueriesData<InboxFolderPages>({ queryKey: inbox.messages._def })
    .flatMap(([queryKey, data]) => {
      const filter = queryKey[2];
      if (data === undefined || typeof filter !== "object" || filter === null) {
        return [];
      }
      return [{ queryKey, filter: filter as InboundMessageFilter, data }];
    });
}

/** The message as the cache last knew it: a folder row, else the reader's copy. */
export function knownRow(queryClient: QueryClient, id: string): InboundMessageRow | undefined {
  for (const folder of cachedFolders(queryClient)) {
    const row = findInPages(folder.data, id);
    if (row !== undefined) {
      return row;
    }
  }

  return queryClient.getQueryData<InboundMessageDetail>(inbox.message(id).queryKey);
}

export type AppliedRows = {
  /** Messages added to a folder that did not show them. */
  inserted: ReadonlySet<string>;
};

/**
 * Writes what messages look like now into every cached folder, the counts
 * and each message's own entry. `before` is what the counts are adjusted
 * from; a message missing from it is counted as new. A list row merges into
 * a cached detail rather than replacing it, since a row carries fewer fields.
 */
export function applyRows(
  queryClient: QueryClient,
  rows: readonly InboundMessageRow[],
  before: ReadonlyMap<string, InboundMessageRow | undefined>,
  context: MembershipContext,
): AppliedRows {
  const inserted = new Set<string>();
  for (const folder of cachedFolders(queryClient)) {
    let data = folder.data;
    for (const row of rows) {
      const placed = placeInFolder(data, row, folder.filter, context);
      data = placed.data;
      if (placed.inserted) {
        inserted.add(row.id);
      }
    }
    if (data !== folder.data) {
      queryClient.setQueryData(folder.queryKey, data);
    }
  }

  queryClient.setQueryData<InboundMessageCounts>(inbox.counts().queryKey, (counts) => {
    if (counts === undefined) {
      return counts;
    }
    let next = counts;
    for (const row of rows) {
      next = adjustCounts(next, before.get(row.id), row, context);
    }
    return next;
  });

  for (const row of rows) {
    queryClient.setQueryData<InboundMessageDetail>(inbox.message(row.id).queryKey, (detail) =>
      detail === undefined ? detail : { ...detail, ...row },
    );
  }

  return { inserted };
}

/** Takes messages out of every folder and the counts: deleted, or no longer visible. */
export function forgetRows(
  queryClient: QueryClient,
  rows: readonly InboundMessageRow[],
  context: MembershipContext,
): void {
  const ids = new Set(rows.map((row) => row.id));
  for (const folder of cachedFolders(queryClient)) {
    const data = removeFromPages(folder.data, ids);
    if (data !== folder.data) {
      queryClient.setQueryData(folder.queryKey, data);
    }
  }
  queryClient.setQueryData<InboundMessageCounts>(inbox.counts().queryKey, (counts) =>
    counts === undefined
      ? counts
      : rows.reduce((next, row) => adjustCounts(next, row, undefined, context), counts),
  );
  for (const id of ids) {
    queryClient.removeQueries({ queryKey: inbox.message(id).queryKey, exact: true });
  }
}
