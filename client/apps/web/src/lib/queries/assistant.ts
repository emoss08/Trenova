import { fetchAgentCapabilities } from "@/lib/graphql/agent-capabilities";
import {
  fetchAgentChoicesByIds,
  fetchAgentDefinitions,
  fetchMyAgents,
  type AgentChoiceQuery,
  type AgentChoiceSource,
} from "@/lib/graphql/agent-definition";
import {
  fetchAgentAccessPreview,
  fetchRoleAgents,
  fetchSuggestedAgentAudience,
  type AgentAccessPreviewRequest,
} from "@/lib/graphql/agent-access";
import { apiService } from "@/services/api";
import type { ArtifactListParams } from "@/services/assistant";
import type { DeskSearchKind, MentionPageKind } from "@/types/assistant";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const assistant = createQueryKeys("assistant", {
  // Paged by cursor, pinned first then newest first. Read as an infinite
  // query through threadListQuery (lib/thread-list.ts); every reader of the
  // list goes through the helpers there.
  threads: () => ({
    queryKey: ["assistant-threads"],
  }),
  thread: (id: string) => ({
    queryKey: ["assistant-thread", id],
    queryFn: () => apiService.assistantService.getThread(id),
  }),
  // Paged newest-first: the page param is the sequence to read above, and
  // nothing for the newest page. useThreadHistory drives it as an infinite
  // query; the key is here so the turn hook can append to the same cache.
  messages: (threadId: string) => ({
    queryKey: ["assistant-messages", threadId],
    queryFn: ({ pageParam, signal }: { pageParam?: unknown; signal?: AbortSignal }) =>
      apiService.assistantService.listMessages(threadId, {
        before: typeof pageParam === "number" ? pageParam : undefined,
        signal,
      }),
  }),
  // Every reply the person's conversations are still writing. Kept fresh by
  // the "assistant_turns" realtime event and by the turn hooks, not polled.
  threadBudget: (threadId: string) => ({
    queryKey: ["assistant-thread-budget", threadId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.threadBudget(threadId, { signal }),
  }),
  // The requests scheduled in a conversation, for its schedule cards. Kept
  // fresh by the "conversation_schedules" realtime event, not polled.
  schedules: (threadId: string) => ({
    queryKey: ["assistant-schedules", threadId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.listThreadSchedules(threadId, { signal }),
  }),
  // What the person left for a conversation while its agent worked, in send
  // order. Kept fresh by the "assistant_queue" realtime event, not polled.
  queue: (threadId: string) => ({
    queryKey: ["assistant-queue", threadId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.listQueue(threadId, { signal }),
  }),
  // What the conversation's agent is waiting on. Kept fresh by the
  // "agent_waits" realtime event, not polled.
  waits: (threadId: string) => ({
    queryKey: ["assistant-waits", threadId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.listWaits(threadId, { signal }),
  }),
  // Where the conversation's case stands and its checklist. Kept fresh by
  // the "assistant_case" and "agent_waits" events and by changes to the
  // record the case is about (lib/case-realtime.ts).
  case: (threadId: string) => ({
    queryKey: ["assistant-case", threadId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.getCase(threadId, { signal }),
  }),
  // Records the person can make a conversation a case about, one kind at a
  // time, paged by the case picker as it scrolls (use-case-records.ts).
  caseRecords: (query: string, kind: MentionPageKind) => ({
    queryKey: [query, kind],
  }),
  activeTurns: () => ({
    queryKey: ["assistant-active-turns"],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.listActiveTurns({ signal }),
  }),
  deskSearch: (query: string, kind: DeskSearchKind) => ({
    queryKey: [query, kind],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.searchDesk(query, kind, { signal }),
  }),
  providers: () => ({
    queryKey: ["assistant-providers"],
    queryFn: () => apiService.assistantService.listProviders(),
  }),
  proposals: (threadId: string) => ({
    queryKey: ["assistant-proposals", threadId],
    queryFn: () => apiService.assistantService.listProposals(threadId),
  }),
  plans: (threadId: string) => ({
    queryKey: ["assistant-plans", threadId],
    queryFn: () => apiService.assistantService.listPlans(threadId),
  }),
  // The first page of a conversation's artifacts by lineage, pinned and
  // newest first, every version with its payload. Every artifact query of a
  // thread is one of its context queries, so they share this key's prefix and
  // one invalidation of it refreshes them all.
  artifacts: (threadId: string) => ({
    queryKey: ["assistant-artifacts", threadId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.assistantService.listArtifacts(threadId, { signal, limit: ARTIFACT_FIRST_PAGE }),
    contextQueries: {
      // The same page with each payload cut to what names and draws an
      // artifact in a list: what the Desk's transcript, counts and stack read.
      // The contents come with their lineage when one is shown.
      summary: {
        queryKey: null,
        queryFn: ({ signal }: { signal?: AbortSignal }) =>
          apiService.assistantService.listArtifacts(threadId, {
            signal,
            limit: ARTIFACT_FIRST_PAGE,
            summary: true,
          }),
      },
      // Every version of one lineage with its payload: what an artifact is
      // drawn from, and how one a link or the store names is found.
      lineage: (artifactId: string) => ({
        queryKey: [artifactId],
        queryFn: ({ signal }: { signal?: AbortSignal }) =>
          apiService.assistantService.artifactLineage(threadId, artifactId, { signal }),
      }),
      slug: (slug: string) => ({
        queryKey: [slug],
        queryFn: ({ signal }: { signal?: AbortSignal }) =>
          apiService.assistantService.artifactBySlug(threadId, slug, { signal }),
      }),
      // The list of every artifact to search, paged by useInfiniteQuery.
      browse: (filters: Pick<ArtifactListParams, "q" | "kind" | "pinned">) => ({
        queryKey: [filters],
      }),
    },
  }),
  // The organization's agents with everything an administrator configures;
  // AI Control only. Chat surfaces read myAgents.
  agents: (enabledOnly: boolean, chatOnly = false) => ({
    queryKey: ["agent-definitions", enabledOnly, chatOnly],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAgentDefinitions({ enabledOnly, chatOnly }, { signal }),
  }),
  // The agents the person may ask, up to a hundred, for naming the agent
  // behind each conversation.
  myAgents: () => ({
    queryKey: ["my-agents"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchMyAgents({ signal }),
  }),
  // What an agent can do, for its capabilities page.
  agentCapabilities: (agentId: string) => ({
    queryKey: ["agent-capabilities", agentId],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentCapabilities(agentId, { signal }),
  }),
  // Paged: read by useAgentChoices as an infinite query, one cursor at a time.
  agentChoices: (query: AgentChoiceQuery, source: AgentChoiceSource, pageSize: number) => ({
    queryKey: [
      "agent-choices",
      source,
      query.search?.trim() ?? "",
      query.origin ?? "all",
      [...(query.excludeIds ?? [])],
      pageSize,
    ],
  }),
  agentChoicesByIds: (ids: readonly string[]) => ({
    queryKey: ["agent-choices-by-id", [...ids]],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentChoicesByIds(ids, { signal }),
  }),
  // Each role's coverage of an agent's tools, for choosing who may use it.
  agentAudience: (agentId: string) => ({
    queryKey: ["agent-audience", agentId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchSuggestedAgentAudience(agentId, { signal }),
  }),
  // The same, worked out from the agent as the form holds it rather than as
  // saved. The request is normalized, so it is its own key.
  agentAccessPreview: (request: AgentAccessPreviewRequest) => ({
    queryKey: ["agent-access-preview", request],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAgentAccessPreview(request, { signal }),
  }),
  // The agents a role is granted, for the role editor.
  roleAgents: (roleId: string) => ({
    queryKey: ["role-agents", roleId],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchRoleAgents(roleId, { signal }),
  }),
  systemAgent: (systemKey: string) => ({
    queryKey: ["agent-definition-system", systemKey],
    queryFn: () => apiService.agentDefinitionService.getBySystemKey(systemKey),
  }),
  toolCatalog: () => ({
    queryKey: ["agent-tool-catalog"],
    queryFn: () => apiService.agentDefinitionService.tools(),
  }),
  eventKinds: () => ({
    queryKey: ["agent-event-kinds"],
    queryFn: () => apiService.agentDefinitionService.eventKinds(),
  }),
  agent: (id: string) => ({
    queryKey: ["agent-definition", id],
    queryFn: () => apiService.agentDefinitionService.get(id),
  }),
  agentBudget: (id: string) => ({
    queryKey: ["agent-budget", id],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentDefinitionService.budget(id, { signal }),
  }),
  agentTrust: (id: string) => ({
    queryKey: ["agent-trust", id],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.agentDefinitionService.trust(id, { signal }),
  }),
  agentTemplates: () => ({
    queryKey: ["agent-templates"],
    queryFn: () => apiService.agentDefinitionService.templates(),
  }),
});

/** How many lineages the first page of a conversation's artifacts holds. */
export const ARTIFACT_FIRST_PAGE = 100;
