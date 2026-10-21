import {
  AgentChoiceFieldsFragmentDoc,
  AgentChoicesDocument,
  AgentDefinitionCardFieldsFragmentDoc,
  AgentDefinitionCardsDocument,
  DataTablePageInfoFieldsFragmentDoc,
  MyAgentFieldsFragmentDoc,
  type AgentChoiceFieldsFragment,
  MyAgentMatchDocument,
  MyAgentsDocument,
  type AgentAccessMode,
  type AgentDefinitionCardFieldsFragment,
  type AgentMatchOutcome,
  type FieldFilterInput,
  type MyAgentFieldsFragment,
  type MyAgentOrigin,
  type MyAgentsInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { rootField } from "@trenova/shared/lib/reference-access";

type AgentDefinitionCardFields = Omit<AgentDefinitionCardFieldsFragment, " $fragmentName">;
type AgentDefinitionCardDelegate = AgentDefinitionCardFields["delegates"][number];

/**
 * An agent as AI Control reads it: the agent is the root record of every read that returns
 * it, so the fields the schema marks nullable for where an agent is linked from another
 * record (docs/engineering/reference-views.md) are its own here and never withheld. Its
 * delegates are agents too, read by someone who may read agents, so theirs are not either.
 */
export type AgentDefinitionRow = Omit<
  AgentDefinitionCardFields,
  "accessMode" | "description" | "enabled" | "systemKey" | "delegates"
> & {
  accessMode: AgentAccessMode;
  description: string;
  enabled: boolean;
  systemKey: string;
  delegates: Array<Omit<AgentDefinitionCardDelegate, "enabled"> & { enabled: boolean }>;
};

export function toAgentDefinitionRow(fields: AgentDefinitionCardFields): AgentDefinitionRow {
  return {
    ...fields,
    accessMode: rootField(fields.accessMode, "AgentDefinition.accessMode"),
    description: rootField(fields.description, "AgentDefinition.description"),
    enabled: rootField(fields.enabled, "AgentDefinition.enabled"),
    systemKey: rootField(fields.systemKey, "AgentDefinition.systemKey"),
    delegates: fields.delegates.map((delegate) => ({
      ...delegate,
      enabled: rootField(delegate.enabled, "AgentDefinition.enabled"),
    })),
  };
}

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
    toAgentDefinitionRow(getFragmentData(AgentDefinitionCardFieldsFragmentDoc, edge.node)),
  );
}

/**
 * An agent as the person asking it sees it: who it is and what to ask it.
 * Chat surfaces read it from `myAgents`, which needs only the assistant and
 * the agent's own grant; AI Control reads the same shape from the
 * organization's full list.
 */
type MyAgentFields = Omit<MyAgentFieldsFragment, " $fragmentName">;

/**
 * Another agent as a hand-off draws it: who it is and its mark, never what it
 * is set up to do.
 */
export type AgentDelegateMark = MyAgentFields["delegates"][number];

export type AgentChoice = Omit<MyAgentFields, "delegates"> & {
  /**
   * The agents this one may hand a task to that the person may use, in the
   * order configured, so an old hand-off in its conversation still draws the
   * delegate's mark. Read only from `myAgents`; the organization's lists,
   * which are an administrator's, leave it out.
   */
  delegates?: readonly AgentDelegateMark[];
};
export type AgentStarter = AgentChoice["starters"][number];

/** Where an agent came from: one of the platform's templates, or built by hand. */
export type AgentOrigin = "all" | "template" | "custom";

/**
 * Whose agents a list reads. `mine` is the agents the person may ask, and is
 * what every chat surface reads. The other two are an administrator's and
 * read the organization's full list, whoever may use each agent:
 * `organization` keeps to enabled chat agents, for choosing who an agent may
 * hand a task to; `grantable` is every agent whatever its trigger and
 * whether it is enabled, for choosing what a role is granted.
 */
export type AgentChoiceSource = "mine" | "organization" | "grantable";

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
 * An agent from the organization's list, which reads each agent as its root record, so its
 * description and system key are never withheld (docs/engineering/reference-views.md).
 */
function toAgentChoice(fields: AgentChoiceFieldsFragment): AgentChoice {
  return {
    id: fields.id,
    name: fields.name,
    description: rootField(fields.description, "AgentDefinition.description"),
    template: fields.template,
    icon: fields.icon,
    accent: fields.accent,
    toolNames: fields.toolNames,
    systemKey: rootField(fields.systemKey, "AgentDefinition.systemKey"),
    starters: fields.starters,
  };
}

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
 * The filters for the organization's list. By default they keep it to
 * agents a person can ask: enabled, and started by a person rather than by a
 * schedule or an event, the rule `myAgents` applies on the server. A list of
 * what a role may be granted leaves that rule out and narrows only by what
 * was asked.
 */
export function agentChoiceFilters(
  query: AgentChoiceQuery = {},
  { askableOnly = true }: { askableOnly?: boolean } = {},
): FieldFilterInput[] {
  const filters: FieldFilterInput[] = askableOnly
    ? [
        { field: "enabled", operator: "eq", value: true },
        { field: "triggerMode", operator: "eq", value: "Chat" },
      ]
    : [];
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
    items: connection.edges.map((edge) =>
      toAgentChoice(getFragmentData(AgentChoiceFieldsFragmentDoc, edge.node)),
    ),
    endCursor: pageInfo.endCursor ?? null,
    hasNextPage: pageInfo.hasNextPage,
    totalCount: connection.totalCount ?? null,
  };
}

/**
 * One page of agents, alphabetical, searched and filtered on the server: the
 * person's own unless an administrator's screen asks for the organization's.
 */
export function fetchAgentChoices(
  query: AgentChoiceQuery,
  page: AgentChoicePageRequest,
  options?: RequestOptions & { source?: AgentChoiceSource },
): Promise<AgentChoicePage> {
  const includeTotalCount = page.includeTotalCount ?? false;
  const source = options?.source ?? "mine";
  if (source === "mine") {
    return requestMyAgents(myAgentsInput(query, page), includeTotalCount, options);
  }

  return requestOrganizationAgents(
    {
      first: page.first,
      after: page.after,
      query: query.search?.trim(),
      fieldFilters: agentChoiceFilters(query, { askableOnly: source === "organization" }),
    },
    includeTotalCount,
    options,
  );
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

/**
 * How well the person's agents fit a question before it is asked, among the
 * agents they may start a conversation with, judged by the tools each holds
 * that the person may use. `BetterAgent` names another agent holding the tools
 * the question needs; `CurrentAgent` says the agent it would go to holds them,
 * itself or through an agent it hands work to; `NoAgent` says tools match but
 * none of the person's agents holds them; `NoMatch` says no tool matches.
 * `tools` names, best first, at most three of the tools that matched.
 */
export type AgentMatch = {
  outcome: AgentMatchOutcome;
  agent: AgentChoice | null;
  tools: AgentMatchTool[];
};

export type AgentMatchTool = { name: string; summary: string };

/** The most characters of a question the server matches to an agent. */
export const AGENT_RECOMMENDATION_MAX_LENGTH = 4000;

/** The most kinds of record a question can be about when it is matched. */
export const AGENT_MATCH_MAX_CONTEXT_KINDS = 11;

export type AgentMatchInput = {
  text: string;
  currentAgentId?: string | null;
  /** The kinds of record the question is about; tools for them rank higher. */
  contextKinds?: readonly string[];
};

/** Asks the server how well the person's agents fit a question. */
export async function fetchAgentMatch(
  input: AgentMatchInput,
  options?: RequestOptions,
): Promise<AgentMatch> {
  const data = await requestGraphQL({
    document: MyAgentMatchDocument,
    operationName: "MyAgentMatch",
    variables: {
      input: {
        text: input.text,
        ...(input.currentAgentId ? { currentAgentId: input.currentAgentId } : {}),
        ...(input.contextKinds && input.contextKinds.length > 0
          ? { contextKinds: [...input.contextKinds] }
          : {}),
      },
    },
    signal: options?.signal,
  });
  const match = data.myAgentMatch;

  return {
    outcome: match.outcome,
    agent: match.agent ? getFragmentData(MyAgentFieldsFragmentDoc, match.agent) : null,
    tools: match.tools.map((tool) => ({ name: tool.name, summary: tool.summary })),
  };
}
