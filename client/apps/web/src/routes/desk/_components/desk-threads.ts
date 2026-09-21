import type { AssistantThread } from "@/types/assistant";

export type DeskThreadGroup = {
  kind: "pinned" | "agent";
  /** The agent's name for an agent group; empty when the agent is gone. */
  label: string;
  agentId: string | null;
  threads: AssistantThread[];
};

function touchedAt(thread: AssistantThread): number {
  return thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
}

function newestFirst(a: AssistantThread, b: AssistantThread): number {
  return touchedAt(b) - touchedAt(a);
}

/**
 * The rail's order: what the person pinned on top, then the rest grouped by
 * the agent they were talking to, the busiest agent first and the newest
 * conversation first inside each group. A day's work with one desk reads
 * as one shelf rather than being shuffled among the others by time alone.
 */
export function groupDeskThreads(
  threads: readonly AssistantThread[],
  agentNames: ReadonlyMap<string, string>,
): DeskThreadGroup[] {
  if (threads.length === 0) {
    return [];
  }

  const pinned = threads.filter((thread) => thread.pinned).sort(newestFirst);
  const byAgent = new Map<string, AssistantThread[]>();
  for (const thread of threads) {
    if (thread.pinned) {
      continue;
    }
    const shelf = byAgent.get(thread.agentDefinitionId);
    if (shelf) {
      shelf.push(thread);
    } else {
      byAgent.set(thread.agentDefinitionId, [thread]);
    }
  }

  const groups: DeskThreadGroup[] = [];
  if (pinned.length > 0) {
    groups.push({ kind: "pinned", label: "", agentId: null, threads: pinned });
  }

  const agentGroups = [...byAgent.entries()]
    .map(([agentId, shelf]) => ({
      kind: "agent" as const,
      label: agentNames.get(agentId) ?? "",
      agentId,
      threads: shelf.sort(newestFirst),
    }))
    .sort(
      (a, b) =>
        b.threads.length - a.threads.length ||
        touchedAt(b.threads[0]) - touchedAt(a.threads[0]) ||
        a.label.localeCompare(b.label),
    );

  return [...groups, ...agentGroups];
}

/** Whether a conversation's title or agent contains the search, case ignored. */
export function matchesThreadSearch(
  thread: AssistantThread,
  agentName: string,
  query: string,
): boolean {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return true;
  }

  return `${thread.title} ${agentName}`.toLowerCase().includes(needle);
}
