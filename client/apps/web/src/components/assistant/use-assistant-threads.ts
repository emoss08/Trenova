import { queries } from "@/lib/queries";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { threadListQuery, threadsOf } from "@/lib/thread-list";
import type { AssistantThread } from "@/types/assistant";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export type AssistantThreads = {
  threads: AssistantThread[];
  agents: AgentChoice[];
  agentsById: Map<string, AgentChoice>;
  /**
   * The open conversation: from the list, or read on its own when the list
   * does not carry it — a palette question that was never kept, opened from
   * the notice that its answer is in.
   */
  activeThread: AssistantThread | null;
  agentsUnavailable: boolean;
  isLoading: boolean;
};

/** The person's conversations and agents, and the one the panel has open. */
export function useAssistantThreads(activeThreadId: string | null): AssistantThreads {
  const threadsQuery = useInfiniteQuery(threadListQuery());
  const agentsQuery = useQuery(queries.assistant.myAgents());
  const threads = threadsOf(threadsQuery.data);
  const agents = useMemo(() => agentsQuery.data ?? [], [agentsQuery.data]);
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);

  const listed = useMemo(
    () => threads.find((thread) => thread.id === activeThreadId) ?? null,
    [activeThreadId, threads],
  );
  // Read on its own only once the list has said it does not have it, so a
  // listed conversation never costs a second request.
  const unlistedQuery = useQuery({
    ...queries.assistant.thread(activeThreadId ?? ""),
    enabled: activeThreadId !== null && threadsQuery.isSuccess && listed === null,
    retry: false,
  });
  const activeThread =
    listed ??
    (activeThreadId !== null && unlistedQuery.data?.id === activeThreadId
      ? unlistedQuery.data
      : null);

  return {
    threads,
    agents,
    agentsById,
    activeThread,
    agentsUnavailable: agentsQuery.isError,
    isLoading: threadsQuery.isLoading || agentsQuery.isLoading,
  };
}
