type ThreadActivity = {
  agentDefinitionId: string;
  lastMessageAt: number;
  createdAt: number;
};

export type AgentRecency = {
  /** Agents the person has talked to, most recent first. */
  ids: string[];
  /** When the person last talked to each agent, in Unix seconds. */
  lastUsedAt: ReadonlyMap<string, number>;
};

export const EMPTY_AGENT_RECENCY: AgentRecency = { ids: [], lastUsedAt: new Map() };

/**
 * Which agents a person reaches for, read from their conversations.
 *
 * A conversation counts from its last message, or from when it was opened
 * when nothing has been said in it yet. The agent the person last asked is
 * put first even when that question was never kept as a conversation — a
 * quick question from the corner panel — because it is the one they are
 * most likely to ask again.
 */
export function agentRecency(
  threads: readonly ThreadActivity[],
  options: { limit: number; preferId?: string | null },
): AgentRecency {
  const lastUsedAt = new Map<string, number>();
  for (const thread of threads) {
    if (thread.agentDefinitionId === "") {
      continue;
    }
    const touched = thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
    const known = lastUsedAt.get(thread.agentDefinitionId);
    if (known === undefined || touched > known) {
      lastUsedAt.set(thread.agentDefinitionId, touched);
    }
  }

  const ordered = [...lastUsedAt.entries()]
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .map(([id]) => id);
  const preferId = options.preferId ?? "";
  const ids = preferId === "" ? ordered : [preferId, ...ordered.filter((id) => id !== preferId)];

  return { ids: ids.slice(0, Math.max(0, options.limit)), lastUsedAt };
}
