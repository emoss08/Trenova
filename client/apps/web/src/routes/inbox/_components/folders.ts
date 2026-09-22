import type { InboundClassification, InboundMessageFilter } from "@/lib/graphql/inbox";
import { INBOUND_CLASSIFICATIONS } from "./classification";
import { LANE_STATUSES, isLaneKey, type LaneKey } from "./lanes";

/**
 * Where a person is looking in the inbox.
 *
 * A lane asks "what state is it in", a kind asks "what is it", a mailbox asks
 * "where did it arrive". They are three ways into the same mail, like a mail
 * client's folders, labels and accounts, and the address carries whichever
 * one is open so a link and the back button both land where they look.
 */
export type InboxFolder =
  | { kind: "lane"; lane: LaneKey }
  | { kind: "classification"; classification: InboundClassification }
  | { kind: "mailbox"; mailboxId: string };

const FOLDER_PARAMS = ["lane", "kind", "mailbox"] as const;

export const DEFAULT_FOLDER: InboxFolder = { kind: "lane", lane: "waiting" };

function isClassification(value: string | null): value is InboundClassification {
  return value !== null && (INBOUND_CLASSIFICATIONS as readonly string[]).includes(value);
}

export function parseFolder(searchParams: URLSearchParams): InboxFolder {
  const kind = searchParams.get("kind");
  if (isClassification(kind)) {
    return { kind: "classification", classification: kind };
  }

  const mailboxId = searchParams.get("mailbox");
  if (mailboxId !== null && mailboxId.trim() !== "") {
    return { kind: "mailbox", mailboxId };
  }

  const lane = searchParams.get("lane");
  if (isLaneKey(lane)) {
    return { kind: "lane", lane };
  }

  return DEFAULT_FOLDER;
}

/**
 * The address for another folder. The open message is dropped — it is very
 * likely not in the folder being moved to — and the search is kept, because
 * narrowing the same words to another folder is the usual next step.
 */
export function folderParams(current: URLSearchParams, folder: InboxFolder): URLSearchParams {
  const next = new URLSearchParams(current);
  for (const key of FOLDER_PARAMS) {
    next.delete(key);
  }
  next.delete("message");

  switch (folder.kind) {
    case "lane":
      next.set("lane", folder.lane);
      break;
    case "classification":
      next.set("kind", folder.classification);
      break;
    case "mailbox":
      next.set("mailbox", folder.mailboxId);
      break;
  }

  return next;
}

export function folderKey(folder: InboxFolder): string {
  switch (folder.kind) {
    case "lane":
      return `lane:${folder.lane}`;
    case "classification":
      return `kind:${folder.classification}`;
    case "mailbox":
      return `mailbox:${folder.mailboxId}`;
  }
}

export function isSameFolder(a: InboxFolder, b: InboxFolder): boolean {
  return folderKey(a) === folderKey(b);
}

export type InboxQueryFilter = Required<
  Pick<InboundMessageFilter, "statuses" | "classification" | "mailboxId" | "query">
>;

export function folderFilter(folder: InboxFolder, search: string): InboxQueryFilter {
  const query = search.trim() === "" ? null : search.trim();

  switch (folder.kind) {
    case "lane":
      return { statuses: LANE_STATUSES[folder.lane], classification: null, mailboxId: null, query };
    case "classification":
      return { statuses: [], classification: folder.classification, mailboxId: null, query };
    case "mailbox":
      return { statuses: [], classification: null, mailboxId: folder.mailboxId, query };
  }
}
