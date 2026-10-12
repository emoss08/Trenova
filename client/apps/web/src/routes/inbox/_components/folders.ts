import type {
  InboundClassification,
  InboundMessageCounts,
  InboundMessageFilter,
  TriageSort,
  TriageView,
} from "@/lib/graphql/inbox";
import { INBOUND_CLASSIFICATIONS } from "./classification";
import { LANE_ENUM, isLaneKey, isWaitingLane, laneCount, type LaneKey } from "./lanes";

/**
 * Where a person is in Triage.
 *
 * The Briefing and Rules are pages of their own. Everything else is a folder
 * of the queue, reached four ways: a lane asks "what state is it in", a kind
 * "what is it" (only what is waiting), a source "where did it arrive", and a
 * view is a search someone saved. The address carries whichever is open, so a
 * link and the back button both land where they look.
 */
export type QueueFolder =
  | { kind: "lane"; lane: LaneKey }
  | { kind: "classification"; classification: InboundClassification }
  | { kind: "mailbox"; mailboxId: string }
  | { kind: "view"; viewId: string };

export type TriageFolder = { kind: "briefing" } | { kind: "rules" } | QueueFolder;

/** The order a person chose; the server's `TriageSort` once it is resolved. */
export type FolderSort = "due" | "newest";

export type TriageLocation = {
  folder: TriageFolder;
  /** The open message, if any. */
  messageId: string | null;
  /** The search typed over the folder, untrimmed. */
  search: string;
  /** The order a person picked; null leaves it to the folder. */
  sort: FolderSort | null;
};

export const BRIEFING: TriageFolder = { kind: "briefing" };
export const RULES: TriageFolder = { kind: "rules" };
export const WAITING: QueueFolder = { kind: "lane", lane: "waiting" };

const PARAM = {
  briefing: "brief",
  rules: "rules",
  lane: "lane",
  kind: "kind",
  mailbox: "mailbox",
  view: "view",
  message: "message",
  search: "q",
  sort: "sort",
} as const;

const FOLDER_PARAMS = [
  PARAM.briefing,
  PARAM.rules,
  PARAM.lane,
  PARAM.kind,
  PARAM.mailbox,
  PARAM.view,
] as const;

function isClassification(value: string | null): value is InboundClassification {
  return value !== null && (INBOUND_CLASSIFICATIONS as readonly string[]).includes(value);
}

function present(value: string | null): value is string {
  return value !== null && value.trim() !== "";
}

function parseSort(value: string | null): FolderSort | null {
  return value === "due" || value === "newest" ? value : null;
}

/**
 * The folder an address names. Nothing named is the Briefing, Triage's
 * landing page — unless a message is named on its own, as an old link or a
 * record link does, which opens Everything so the message is in the list.
 * A value the page does not know is ignored rather than trusted.
 */
export function parseFolder(params: URLSearchParams): TriageFolder {
  if (params.has(PARAM.briefing)) {
    return BRIEFING;
  }
  if (params.has(PARAM.rules)) {
    return RULES;
  }

  const kind = params.get(PARAM.kind);
  if (isClassification(kind)) {
    return { kind: "classification", classification: kind };
  }

  const mailboxId = params.get(PARAM.mailbox);
  if (present(mailboxId)) {
    return { kind: "mailbox", mailboxId };
  }

  const viewId = params.get(PARAM.view);
  if (present(viewId)) {
    return { kind: "view", viewId };
  }

  const lane = params.get(PARAM.lane);
  if (isLaneKey(lane)) {
    return { kind: "lane", lane };
  }

  return present(params.get(PARAM.message)) ? { kind: "lane", lane: "all" } : BRIEFING;
}

export function parseTriageLocation(params: URLSearchParams): TriageLocation {
  const folder = parseFolder(params);
  const messageId = params.get(PARAM.message);

  return {
    folder,
    messageId: isQueueFolder(folder) && present(messageId) ? messageId : null,
    search: isQueueFolder(folder) ? (params.get(PARAM.search) ?? "") : "",
    sort: parseSort(params.get(PARAM.sort)),
  };
}

/**
 * The address for another folder. The open message is dropped — it is very
 * likely not in the folder being moved to — and the search is kept for
 * another folder of the queue, because narrowing the same words elsewhere is
 * the usual next step.
 */
export function folderParams(current: URLSearchParams, folder: TriageFolder): URLSearchParams {
  const next = new URLSearchParams(current);
  for (const key of FOLDER_PARAMS) {
    next.delete(key);
  }
  next.delete(PARAM.message);

  switch (folder.kind) {
    case "briefing":
      next.delete(PARAM.search);
      break;
    case "rules":
      next.delete(PARAM.search);
      next.set(PARAM.rules, "");
      break;
    case "lane":
      next.set(PARAM.lane, folder.lane);
      break;
    case "classification":
      next.set(PARAM.kind, folder.classification);
      break;
    case "mailbox":
      next.set(PARAM.mailbox, folder.mailboxId);
      break;
    case "view":
      next.set(PARAM.view, folder.viewId);
      break;
  }

  return next;
}

export function messageParams(current: URLSearchParams, messageId: string | null): URLSearchParams {
  const next = new URLSearchParams(current);
  if (messageId === null) {
    next.delete(PARAM.message);
  } else {
    next.set(PARAM.message, messageId);
  }

  return next;
}

export function searchParamsWith(current: URLSearchParams, search: string): URLSearchParams {
  const next = new URLSearchParams(current);
  if (search.trim() === "") {
    next.delete(PARAM.search);
  } else {
    next.set(PARAM.search, search);
  }

  return next;
}

export function sortParams(current: URLSearchParams, sort: FolderSort): URLSearchParams {
  const next = new URLSearchParams(current);
  next.set(PARAM.sort, sort);

  return next;
}

export function isQueueFolder(folder: TriageFolder): folder is QueueFolder {
  return folder.kind !== "briefing" && folder.kind !== "rules";
}

export function folderKey(folder: TriageFolder): string {
  switch (folder.kind) {
    case "briefing":
      return "briefing";
    case "rules":
      return "rules";
    case "lane":
      return `lane:${folder.lane}`;
    case "classification":
      return `kind:${folder.classification}`;
    case "mailbox":
      return `mailbox:${folder.mailboxId}`;
    case "view":
      return `view:${folder.viewId}`;
  }
}

export function isSameFolder(a: TriageFolder, b: TriageFolder): boolean {
  return folderKey(a) === folderKey(b);
}

/**
 * Whether anything in a folder is waiting on a person. Only then does "due
 * first" mean anything; a folder of settled mail is read newest first.
 */
export function folderHasWaiting(
  folder: QueueFolder,
  counts: InboundMessageCounts | undefined,
  views: readonly TriageView[],
): boolean {
  if (counts === undefined) {
    return true;
  }
  switch (folder.kind) {
    case "lane":
      return isWaitingLane(folder.lane)
        ? laneCount(counts, folder.lane) > 0
        : folder.lane === "all" && counts.waiting > 0;
    case "classification":
      return (
        (counts.byClassification.find((row) => row.classification === folder.classification)
          ?.waiting ?? 0) > 0
      );
    case "mailbox":
      return (counts.byMailbox.find((row) => row.mailboxId === folder.mailboxId)?.waiting ?? 0) > 0;
    case "view":
      return (views.find((view) => view.id === folder.viewId)?.waiting ?? 0) > 0;
  }
}

/** The order the server is asked for: due first by default, newest when nothing waits. */
export function resolveSort(sort: FolderSort | null, hasWaiting: boolean): TriageSort {
  if (!hasWaiting) {
    return "Newest";
  }

  return sort === "newest" ? "Newest" : "Due";
}

function joinQuery(...parts: Array<string | null | undefined>): string | null {
  const words = parts
    .map((part) => (typeof part === "string" ? part.trim() : ""))
    .filter((part) => part !== "");

  return words.length === 0 ? null : words.join(" ");
}

export type FolderFilterOptions = {
  search: string;
  sort: TriageSort;
  views: readonly TriageView[];
};

/**
 * Exactly the slice of the queue a folder shows, as the server is asked for
 * it. One row per conversation: the newest message of each thread in the
 * folder, with the thread's size on the row. Null for a view that no longer
 * exists, which has nothing to ask for.
 */
export function folderFilter(
  folder: QueueFolder,
  { search, sort, views }: FolderFilterOptions,
): InboundMessageFilter | null {
  const base = { sort, collapseThreads: true };

  switch (folder.kind) {
    case "lane":
      return { ...base, lane: LANE_ENUM[folder.lane], query: joinQuery(search) };
    case "classification":
      return {
        ...base,
        lane: "Waiting",
        classification: folder.classification,
        query: joinQuery(search),
      };
    case "mailbox":
      return {
        ...base,
        lane: "Everything",
        mailboxId: folder.mailboxId,
        query: joinQuery(search),
      };
    case "view": {
      const view = views.find((row) => row.id === folder.viewId);
      if (view === undefined) {
        return null;
      }

      return { ...base, lane: "Everything", query: joinQuery(view.query, search) };
    }
  }
}
