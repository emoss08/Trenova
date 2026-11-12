import {
  deskShelfAt,
  type DeskShelfEdges,
  type DeskShelfKey,
} from "@/components/desk-chat/rail/desk-threads";
import type {
  DeskRailCase,
  DeskRailRow,
  DeskRailShelf,
} from "@/components/desk-chat/rail/desk-rail-rows";
import { conversationPath } from "@/lib/conversation-path";
import type { DeskAttentionGroup } from "@/lib/graphql/desk-attention";
import type { DeskOpenWork } from "@/stores/desk-margin-store";
import type { AssistantThread } from "@/types/assistant";
import { casePath } from "../attention/attention-record";
import { caseOfThread } from "../record-case/case-subject";

/**
 * How many waiting decisions the rail asks for. More than it draws: a decision
 * about no record, or about one the person may not read, is not a row.
 */
export const RAIL_WAITING_CASES = 24;

export type RailCaseRow = DeskRailCase;

/** A case that is on the rail without a conversation of its own, and when it last moved. */
type ShelvedCase = { row: RailCaseRow; at: number };

function caseOfConversation(thread: AssistantThread, waiting: ReadonlySet<string>): RailCaseRow {
  const record = caseOfThread(thread);
  const to = record ? casePath(record.kind, record.id) : conversationPath(thread.id);

  return {
    to,
    recordId: record?.id ?? "",
    title: thread.case?.record.label || thread.title,
    waiting: waiting.has(to),
    view: false,
    thread,
  };
}

/**
 * The rail's cases as rows, shelved by day the way conversations are: every
 * record the person has asked an agent about on its case, on the shelf its
 * conversation was last touched; and, with no conversation yet, every record
 * a plan or a single change waits on a person for, on the shelf it was raised,
 * and what was opened this session, under Today.
 *
 * A case put away is left out until its time is up or something about it
 * waits on the person again.
 *
 * A record is one row however it got here. One that waits says so on its row
 * and stays on its shelf, so a case is found where it was left. A record
 * nobody may read is counted in Needs you and left out here, where a row is a
 * name.
 */
export function railCaseRows(
  shelves: readonly DeskRailShelf[],
  waiting: readonly DeskAttentionGroup[],
  session: readonly DeskOpenWork[],
  edges: DeskShelfEdges,
  now: number,
): DeskRailRow[] {
  const waits = new Map<string, { label: string; recordId: string; at: number }>();
  for (const group of waiting) {
    const record = group.record;
    if (!record || !record.readable || record.label === "") {
      continue;
    }
    const to = casePath(record.kind, record.id);
    if (!waits.has(to)) {
      const raised = group.items.reduce((latest, item) => Math.max(latest, item.occurredAt), 0);
      waits.set(to, { label: record.label, recordId: record.id, at: raised || now });
    }
  }
  const waitingPaths = new Set(waits.keys());

  // What the person has a conversation on is shelved by that conversation;
  // each record once, by its newest.
  const listed = new Set<string>();
  const byShelf = new Map<DeskShelfKey, ShelvedCase[]>();
  const shelve = (key: DeskShelfKey, entry: ShelvedCase) => {
    if (listed.has(entry.row.to)) {
      return;
    }
    listed.add(entry.row.to);
    const shelf = byShelf.get(key);
    if (shelf) {
      shelf.push(entry);
    } else {
      byShelf.set(key, [entry]);
    }
  };
  const names = new Map<string, string>();
  for (const work of session) {
    names.set(work.path ?? casePath(work.kind, work.id), work.label);
  }
  for (const [to, wait] of waits) {
    names.set(to, wait.label);
  }
  for (const shelf of shelves) {
    for (const thread of shelf.threads) {
      const row = caseOfConversation(thread, waitingPaths);
      // A case the person put away is off their rail until it comes back,
      // unless something about it now waits on them.
      if (thread.case?.state === "Snoozed" && !row.waiting) {
        continue;
      }
      // The page and Needs you know the record's own name; a conversation's
      // title is what was asked about it.
      row.title = names.get(row.to) ?? row.title;
      shelve(shelf.key, { row, at: Number.MAX_SAFE_INTEGER });
    }
  }
  for (const [to, wait] of waits) {
    shelve(deskShelfAt(wait.at, edges), {
      row: {
        to,
        recordId: wait.recordId,
        title: wait.label,
        waiting: true,
        view: false,
        thread: null,
      },
      at: wait.at,
    });
  }
  for (const work of session) {
    const view = work.path !== undefined;
    shelve("today", {
      row: {
        to: work.path ?? casePath(work.kind, work.id),
        recordId: view ? "" : work.id,
        title: work.label,
        waiting: false,
        view,
        thread: null,
      },
      at: now,
    });
  }

  const rows: DeskRailRow[] = [];
  for (const shelf of shelves) {
    const cases = byShelf.get(shelf.key) ?? [];
    const { loading, hasMore, failed } = shelf.paging;
    const more = loading || hasMore || failed;
    if (cases.length === 0 && !more) {
      continue;
    }
    rows.push({ kind: "shelf", key: shelf.key });
    for (const entry of cases) {
      rows.push({ kind: "case", case: entry.row });
    }
    if (more) {
      rows.push({ kind: "more", shelf: shelf.key, loading, failed });
    }
  }

  return rows;
}
