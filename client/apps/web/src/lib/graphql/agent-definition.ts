import {
  AgentChoiceFieldsFragmentDoc,
  AgentChoicesDocument,
  AgentDefinitionCardFieldsFragmentDoc,
  AgentDefinitionCardsDocument,
  DataTablePageInfoFieldsFragmentDoc,
  MyAgentFieldsFragmentDoc,
  MyAgentsDocument,
  type AgentDefinitionCardFieldsFragment,
  type FieldFilterInput,
  type MyAgentFieldsFragment,
  type MyAgentOrigin,
  type MyAgentsInput,
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
 * The organization's agents in one request, up to a hundred, with everything
 * an administrator configures. Only AI Control reads this: it needs
 * permission to read agent definitions, which a person who only talks to
 * agents does not have. Chat surfaces read `fetchMyAgents` instead.
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

/**
 * An agent as the person asking it sees it: who it is and what to ask it.
 * Chat surfaces read it from `myAgents`, which needs only the assistant and
 * the agent's own grant; AI Control reads the same shape from the
 * organization's full list.
 */
export type AgentChoice = Omit<MyAgentFieldsFragment, " $fragmentName">;
export type AgentStarter = AgentChoice["starters"][number];

/** Where an agent came from: one of the platform's templates, or built by hand. */
export type AgentOrigin = "all" | "template" | "custom";

/**
 * Whose agents a list reads. `mine` is the agents the person may ask, and is
 * what every chat surface reads. `organization` is every enabled chat agent
 * the organization has, whoever may use it; only AI Control reads it, to
 * choose who an agent may hand a task to.
 */
export type AgentChoiceSource = "mine" | "organization";

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

/** The most agents `myAgents` returns in one page, and the most ids it takes. */
export const MY_AGENTS_LIMIT = 100;

const MY_AGENT_ORIGINS: Record<AgentOrigin, MyAgentOrigin> = {
  all: "All",
  template: "Template",
  custom: "Custom",
};

/**
 * The `myAgents` input for a query and a page. The server already keeps the
 * list to enabled chat agents the person may use, so only the narrowing is
 * sent: a blank search, an empty exclusion list and a missing cursor are
 * left out rather than sent empty.
 */
export function myAgentsInput(
  query: AgentChoiceQuery & { ids?: readonly string[] },
  page: { first: number; after?: string | null },
): MyAgentsInput {
  const input: MyAgentsInput = {
    first: page.first,
    origin: MY_AGENT_ORIGINS[query.origin ?? "all"],
  };
  const search = query.search?.trim();
  if (search) {
    input.search = search;
  }
  if (page.after) {
    input.after = page.after;
  }
  if (query.excludeIds && query.excludeIds.length > 0) {
    input.excludeIds = [...query.excludeIds];
  }
  if (query.ids) {
    input.ids = [...query.ids];
  }

  return input;
}

async function requestMyAgents(
  input: MyAgentsInput,
  includeTotalCount: boolean,
  options?: RequestOptions,
): Promise<AgentChoicePage> {
  const data = await requestGraphQL({
    document: MyAgentsDocument,
    operationName: "MyAgents",
    variables: { input, includeTotalCount },
    signal: options?.signal,
  });
  const connection = data.myAgents;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);

  return {
    items: connection.edges.map((edge) => getFragmentData(MyAgentFieldsFragmentDoc, edge.node)),
    endCursor: pageInfo.endCursor ?? null,
    hasNextPage: pageInfo.hasNextPage,
    totalCount: connection.totalCount ?? null,
  };
}

/**
 * The filters that make an agent in the organization's list one a person
 * can ask: enabled, and started by a person rather than by a schedule or an
 * event. `myAgents` applies the same rule on the server, so only the
 * organization's list sends them.
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

async function requestOrganizationAgents(
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

/**
 * One page of agents, alphabetical, searched and filtered on the server: the
 * person's own unless AI Control asks for the organization's.
 */
export function fetchAgentChoices(
  query: AgentChoiceQuery,
  page: AgentChoicePageRequest,
  options?: RequestOptions & { source?: AgentChoiceSource },
): Promise<AgentChoicePage> {
  const includeTotalCount = page.includeTotalCount ?? false;
  if (options?.source === "organization") {
    return requestOrganizationAgents(
      {
        first: page.first,
        after: page.after,
        query: query.search?.trim(),
        fieldFilters: agentChoiceFilters(query),
      },
      includeTotalCount,
      options,
    );
  }

  return requestMyAgents(myAgentsInput(query, page), includeTotalCount, options);
}

/**
 * The agents among the given ids the person may ask, in the order the ids
 * were given. An id that is gone — the agent was disabled or deleted, or the
 * person lost access to it — is simply absent. `myAgents` takes at most a
 * hundred ids, so only the first hundred are asked about.
 */
export async function fetchAgentChoicesByIds(
  ids: readonly string[],
  options?: RequestOptions,
): Promise<AgentChoice[]> {
  const wanted = ids.slice(0, MY_AGENTS_LIMIT);
  if (wanted.length === 0) {
    return [];
  }
  const page = await requestMyAgents(
    myAgentsInput({ ids: wanted }, { first: wanted.length }),
    false,
    options,
  );
  const byId = new Map(page.items.map((agent) => [agent.id, agent]));

  return wanted.flatMap((id) => {
    const agent = byId.get(id);
    return agent ? [agent] : [];
  });
}

/**
 * Every agent the person may ask, alphabetical, up to a hundred. The
 * conversation list and the thread read it to name and draw the agent behind
 * each conversation; anything a person browses reads the paged
 * `fetchAgentChoices` instead.
 */
export async function fetchMyAgents(options?: RequestOptions): Promise<AgentChoice[]> {
  const page = await requestMyAgents(myAgentsInput({}, { first: MY_AGENTS_LIMIT }), false, options);

  return page.items;
}
