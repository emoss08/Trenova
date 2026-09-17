import {
  AgentDefinitionCardFieldsFragmentDoc,
  AgentDefinitionCardsDocument,
  type AgentDefinitionCardFieldsFragment,
  type FieldFilterInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentDefinitionRow = AgentDefinitionCardFieldsFragment;

type RequestOptions = { signal?: AbortSignal };

export type AgentDefinitionFilter = {
  /** Only agents that can be used right now. */
  enabledOnly?: boolean;
  /** Only agents a person can talk to, as opposed to scheduled or event-driven ones. */
  chatOnly?: boolean;
};

/**
 * The organization's agents in one request. There are a handful per
 * organization, never pages, so the cards and the assistant's picker both read
 * the whole set and filter in the query rather than in the client.
 */
export async function fetchAgentDefinitions(
  filter: AgentDefinitionFilter = {},
  options?: RequestOptions,
): Promise<AgentDefinitionRow[]> {
  const fieldFilters: FieldFilterInput[] = [];
  if (filter.enabledOnly) {
    fieldFilters.push({ field: "enabled", operator: "eq", value: true });
  }
  if (filter.chatOnly) {
    fieldFilters.push({ field: "triggerMode", operator: "eq", value: "Chat" });
  }

  const data = await requestGraphQL({
    document: AgentDefinitionCardsDocument,
    operationName: "AgentDefinitionCards",
    variables: {
      input: {
        first: 100,
        fieldFilters,
        sort: [{ field: "name", direction: "asc" }],
      },
      includeTotalCount: false,
    },
    signal: options?.signal,
  });

  return data.agentDefinitions.edges.map((edge) =>
    getFragmentData(AgentDefinitionCardFieldsFragmentDoc, edge.node),
  );
}
