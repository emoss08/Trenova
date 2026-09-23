import {
  AgentChoiceFieldsFragmentDoc,
  AgentChoicesDocument,
  AgentDefinitionCardFieldsFragmentDoc,
  AgentDefinitionCardsDocument,
  DataTablePageInfoFieldsFragmentDoc,
  type AgentChoiceFieldsFragment,
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
 * The organization's agents in one request, up to a hundred. The admin
 * cards and the lookups by id read this; anything that lets a person browse
 * agents reads the paged `fetchAgentChoices` instead, because an
 * organization can hold more agents than one screen should draw.
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

export type AgentChoice = AgentChoiceFieldsFragment;
export type AgentStarter = AgentChoice["starters"][number];

/** Where an agent came from: one of the platform's templates, or built by hand. */
export type AgentOrigin = "all" | "template" | "custom";

export type AgentChoiceQuery = {
  /** Matched against the agent's name and description on the server. */
  search?: string;
  origin?: AgentOrigin;
  /** Agents already shown above the list, so a page never repeats them. */
  excludeIds?: readonly string[];
};

export type AgentChoicePageRequest = {
  first: number;
  after?: string | null;
  includeTotalCount?: boolean;
};

export type AgentChoicePage = {
  items: AgentChoice[];
  endCursor: string | null;
  hasNextPage: boolean;
  /** Only read when the page asked for it; the count is a second query on the server. */
  totalCount: number | null;
};

/**
 * The filters that make an agent one a person can ask: enabled, and started
 * by a person rather than by a schedule or an event. Everything the chooser
 * shows goes through this, so a scheduled digest never turns up in a picker
 * that would open a conversation with it.
 */
export function agentChoiceFilters(query: AgentChoiceQuery = {}): FieldFilterInput[] {
  const filters: FieldFilterInput[] = [
    { field: "enabled", operator: "eq", value: true },
    { field: "triggerMode", operator: "eq", value: "Chat" },
  ];
  if (query.origin === "template") {
    filters.push({ field: "template", operator: "isnotnull", value: null });
  }
  if (query.origin === "custom") {
    filters.push({ field: "template", operator: "isnull", value: null });
  }
  if (query.excludeIds && query.excludeIds.length > 0) {
    filters.push({ field: "id", operator: "notin", value: [...query.excludeIds] });
  }

  return filters;
}

async function requestAgentChoices(
  input: { first: number; after?: string | null; query?: string; fieldFilters: FieldFilterInput[] },
  includeTotalCount: boolean,
  options?: RequestOptions,
): Promise<AgentChoicePage> {
  const data = await requestGraphQL({
    document: AgentChoicesDocument,
    operationName: "AgentChoices",
    variables: {
      input: {
        first: input.first,
        after: input.after || undefined,
        query: input.query || undefined,
        fieldFilters: input.fieldFilters,
        sort: [{ field: "name", direction: "asc" }],
      },
      includeTotalCount,
    },
    signal: options?.signal,
  });
  const connection = data.agentDefinitions;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);

  return {
    items: connection.edges.map((edge) => getFragmentData(AgentChoiceFieldsFragmentDoc, edge.node)),
    endCursor: pageInfo.endCursor ?? null,
    hasNextPage: pageInfo.hasNextPage,
    totalCount: connection.totalCount ?? null,
  };
}

/** One page of the agents a person can ask, alphabetical, searched and filtered on the server. */
export function fetchAgentChoices(
  query: AgentChoiceQuery,
  page: AgentChoicePageRequest,
  options?: RequestOptions,
): Promise<AgentChoicePage> {
  return requestAgentChoices(
    {
      first: page.first,
      after: page.after,
      query: query.search?.trim(),
      fieldFilters: agentChoiceFilters(query),
    },
    page.includeTotalCount ?? false,
    options,
  );
}

/**
 * The askable agents among the given ids, in the order the ids were given.
 * An id that is gone — the agent was disabled or deleted — is simply absent.
 */
export async function fetchAgentChoicesByIds(
  ids: readonly string[],
  options?: RequestOptions,
): Promise<AgentChoice[]> {
  if (ids.length === 0) {
    return [];
  }
  const page = await requestAgentChoices(
    {
      first: ids.length,
      fieldFilters: [...agentChoiceFilters(), { field: "id", operator: "in", value: [...ids] }],
    },
    false,
    options,
  );
  const byId = new Map(page.items.map((agent) => [agent.id, agent]));

  return ids.flatMap((id) => {
    const agent = byId.get(id);
    return agent ? [agent] : [];
  });
}
