import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentReflectionRowFieldsFragmentDoc,
  AgentReflectionsDocument,
  type AgentReflectionRowFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentReflection = AgentReflectionRowFieldsFragment;

export const AGENT_REFLECTIONS_KEY = "agent-reflections";

const RECENT_REFLECTIONS = 20;

/**
 * The latest times an agent looked back over its work and decided something:
 * kept, offered or refused a lesson, or could not finish. Stretches skipped
 * because nothing called for a look are left out; they are most of them.
 */
export async function fetchRecentAgentReflections(options?: {
  signal?: AbortSignal;
}): Promise<AgentReflection[]> {
  const data = await requestGraphQL({
    document: AgentReflectionsDocument,
    operationName: "AgentReflections",
    variables: {
      input: {
        first: RECENT_REFLECTIONS,
        fieldFilters: [{ field: "status", operator: "in", value: ["Completed", "Failed"] }],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
      includeTotalCount: false,
    },
    signal: options?.signal,
  });

  return data.agentReflections.edges.map((edge) =>
    getFragmentData(AgentReflectionRowFieldsFragmentDoc, edge.node),
  );
}
