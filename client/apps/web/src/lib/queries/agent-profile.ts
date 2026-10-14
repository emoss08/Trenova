import {
  agentProfileHistoryQueryKey,
  agentProfileQueryKey,
  fetchAgentProfile,
  fetchAgentProfileHistory,
  type AgentProfileOwner,
} from "@/lib/graphql/agent-profile";

type Signal = { signal?: AbortSignal };

/**
 * The profile agents are told about the organization or about the signed-in person,
 * keyed under the root the realtime resource `agent_profile` invalidates. The Desk
 * memory route warms the person's own on navigation, so the card paints with the page
 * instead of after it.
 */
export function agentProfileQueryOptions(owner: AgentProfileOwner) {
  return {
    queryKey: agentProfileQueryKey(owner),
    queryFn: ({ signal }: Signal) => fetchAgentProfile(owner, { signal }),
  };
}

export function agentProfileHistoryQueryOptions() {
  return {
    queryKey: agentProfileHistoryQueryKey,
    queryFn: ({ signal }: Signal) => fetchAgentProfileHistory({ signal }),
  };
}
