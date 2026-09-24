import { AgentChoicesDocument, MyAgentsDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  agentChoiceFilters,
  fetchAgentChoices,
  fetchAgentChoicesByIds,
  fetchMyAgents,
  myAgentsInput,
  MY_AGENTS_LIMIT,
} from "../agent-definition";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables: { input: Record<string, unknown>; includeTotalCount?: boolean };
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

const agent = (id: string, name: string) => ({
  id,
  name,
  description: "",
  template: null,
  icon: "",
  accent: "",
  toolNames: [],
  systemKey: "",
  starters: [{ label: "Build a report", prompt: "Build a report." }],
});

type PageInfo = { hasNextPage: boolean; endCursor: string | null };

const connection = (
  nodes: ReturnType<typeof agent>[],
  pageInfo: PageInfo,
  totalCount?: number | null,
) => ({
  edges: nodes.map((node) => ({ node })),
  pageInfo,
  ...(totalCount === undefined ? {} : { totalCount }),
});

const myAgents = (
  nodes: ReturnType<typeof agent>[],
  pageInfo: PageInfo,
  totalCount?: number | null,
) => ({
  myAgents: connection(nodes, pageInfo, totalCount),
});

const organizationAgents = (
  nodes: ReturnType<typeof agent>[],
  pageInfo: PageInfo,
  totalCount?: number | null,
) => ({
  agentDefinitions: connection(nodes, pageInfo, totalCount),
});

const ASKABLE = [
  { field: "enabled", operator: "eq", value: true },
  { field: "triggerMode", operator: "eq", value: "Chat" },
];

describe("myAgentsInput", () => {
  it("sends only the page and the origin when nothing narrows the list", () => {
    expect(myAgentsInput({}, { first: 24 })).toEqual({ first: 24, origin: "All" });
  });

  it("maps each origin onto the schema's enum", () => {
    expect(myAgentsInput({ origin: "template" }, { first: 1 }).origin).toBe("Template");
    expect(myAgentsInput({ origin: "custom" }, { first: 1 }).origin).toBe("Custom");
    expect(myAgentsInput({ origin: "all" }, { first: 1 }).origin).toBe("All");
  });

  it("trims the search and leaves a blank one out rather than sending it empty", () => {
    expect(myAgentsInput({ search: "  report  " }, { first: 1 }).search).toBe("report");
    expect(myAgentsInput({ search: "   " }, { first: 1 })).not.toHaveProperty("search");
    expect(myAgentsInput({ search: "" }, { first: 1 })).not.toHaveProperty("search");
  });

  it("sends a cursor only when there is one", () => {
    expect(myAgentsInput({}, { first: 1, after: "cursor-1" }).after).toBe("cursor-1");
    expect(myAgentsInput({}, { first: 1, after: null })).not.toHaveProperty("after");
    expect(myAgentsInput({}, { first: 1, after: "" })).not.toHaveProperty("after");
  });

  it("leaves out agents already listed, and only when there are some", () => {
    expect(myAgentsInput({ excludeIds: ["agdef_1", "agdef_2"] }, { first: 1 }).excludeIds).toEqual([
      "agdef_1",
      "agdef_2",
    ]);
    expect(myAgentsInput({ excludeIds: [] }, { first: 1 })).not.toHaveProperty("excludeIds");
  });

  it("names the ids asked for, and never the enabled or trigger filters the server applies", () => {
    const input = myAgentsInput({ ids: ["agdef_b", "agdef_a"] }, { first: 2 });

    expect(input.ids).toEqual(["agdef_b", "agdef_a"]);
    expect(input).not.toHaveProperty("fieldFilters");
  });
});

describe("agentChoiceFilters", () => {
  it("keeps the organization's list to enabled agents a person starts", () => {
    expect(agentChoiceFilters()).toEqual(ASKABLE);
  });

  it("narrows by where the agent came from", () => {
    expect(agentChoiceFilters({ origin: "template" })).toContainEqual({
      field: "template",
      operator: "isnotnull",
      value: null,
    });
    expect(agentChoiceFilters({ origin: "custom" })).toContainEqual({
      field: "template",
      operator: "isnull",
      value: null,
    });
    expect(agentChoiceFilters({ origin: "all" })).toEqual(ASKABLE);
  });

  it("leaves out agents already listed above the page, and only when there are some", () => {
    expect(agentChoiceFilters({ excludeIds: ["agdef_1", "agdef_2"] })).toContainEqual({
      field: "id",
      operator: "notin",
      value: ["agdef_1", "agdef_2"],
    });
    expect(agentChoiceFilters({ excludeIds: [] })).toEqual(ASKABLE);
  });
});

describe("fetchAgentChoices", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reads the person's own agents from myAgents by default", async () => {
    requestGraphQLMock.mockResolvedValue(
      myAgents(
        [agent("agdef_1", "Report Builder"), agent("agdef_2", "Help")],
        { hasNextPage: true, endCursor: "cursor-2" },
        42,
      ),
    );
    const controller = new AbortController();

    const page = await fetchAgentChoices(
      { search: "  report ", origin: "custom", excludeIds: ["agdef_9"] },
      { first: 30, after: "cursor-1", includeTotalCount: true },
      { signal: controller.signal },
    );

    const call = lastCall();
    expect(call.document).toBe(MyAgentsDocument);
    expect(call.operationName).toBe("MyAgents");
    expect(call.signal).toBe(controller.signal);
    expect(call.variables.includeTotalCount).toBe(true);
    expect(call.variables.input).toEqual({
      first: 30,
      after: "cursor-1",
      search: "report",
      origin: "Custom",
      excludeIds: ["agdef_9"],
    });
    expect(page.items.map((item) => item.id)).toEqual(["agdef_1", "agdef_2"]);
    expect(page.items[0].starters).toEqual([
      { label: "Build a report", prompt: "Build a report." },
    ]);
    expect(page).toMatchObject({ endCursor: "cursor-2", hasNextPage: true, totalCount: 42 });
  });

  it("reads an absent count as unknown and a null cursor as the end", async () => {
    requestGraphQLMock.mockResolvedValue(myAgents([], { hasNextPage: false, endCursor: null }));

    const page = await fetchAgentChoices({ search: "   " }, { first: 30 });

    expect(lastCall().variables.input).toEqual({ first: 30, origin: "All" });
    expect(lastCall().variables.includeTotalCount).toBe(false);
    expect(page).toEqual({ items: [], endCursor: null, hasNextPage: false, totalCount: null });
  });

  it("reads a null count, which the server sends when it skipped the count, as unknown", async () => {
    requestGraphQLMock.mockResolvedValue(
      myAgents([agent("agdef_1", "Help")], { hasNextPage: false, endCursor: "c" }, null),
    );

    const page = await fetchAgentChoices({}, { first: 5, includeTotalCount: true });

    expect(page.totalCount).toBeNull();
  });

  it("reads the organization's agents from agentDefinitions only when AI Control asks", async () => {
    requestGraphQLMock.mockResolvedValue(
      organizationAgents([agent("agdef_3", "Payroll")], { hasNextPage: false, endCursor: null }, 1),
    );

    const page = await fetchAgentChoices(
      { search: "pay", origin: "template" },
      { first: 10, includeTotalCount: true },
      { source: "organization" },
    );

    const call = lastCall();
    expect(call.document).toBe(AgentChoicesDocument);
    expect(call.operationName).toBe("AgentChoices");
    expect(call.variables.input).toEqual({
      first: 10,
      after: undefined,
      query: "pay",
      fieldFilters: [...ASKABLE, { field: "template", operator: "isnotnull", value: null }],
      sort: [{ field: "name", direction: "asc" }],
    });
    expect(page).toEqual({
      items: [agent("agdef_3", "Payroll")],
      endCursor: null,
      hasNextPage: false,
      totalCount: 1,
    });
  });

  // A role can be granted a scheduled agent, whose proposals its members then
  // decide, or a disabled one ahead of switching it on.
  it("lists every agent a role could be granted, whatever its trigger or state", async () => {
    requestGraphQLMock.mockResolvedValue(
      organizationAgents([agent("agdef_4", "Nightly digest")], {
        hasNextPage: false,
        endCursor: null,
      }),
    );

    await fetchAgentChoices(
      { search: "", excludeIds: ["agdef_1"] },
      { first: 24 },
      { source: "grantable" },
    );

    const call = lastCall();
    expect(call.document).toBe(AgentChoicesDocument);
    expect(call.variables.input).toEqual({
      first: 24,
      after: undefined,
      query: undefined,
      fieldFilters: [{ field: "id", operator: "notin", value: ["agdef_1"] }],
      sort: [{ field: "name", direction: "asc" }],
    });
  });
});

describe("fetchAgentChoicesByIds", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("asks for nothing when there are no ids", async () => {
    await expect(fetchAgentChoicesByIds([])).resolves.toEqual([]);
    expect(requestGraphQLMock).not.toHaveBeenCalled();
  });

  it("returns the agents in the order asked, dropping ones that are gone or no longer theirs", async () => {
    requestGraphQLMock.mockResolvedValue(
      myAgents([agent("agdef_a", "Alpha"), agent("agdef_c", "Charlie")], {
        hasNextPage: false,
        endCursor: null,
      }),
    );

    const agents = await fetchAgentChoicesByIds(["agdef_c", "agdef_gone", "agdef_a"]);

    expect(agents.map((item) => item.id)).toEqual(["agdef_c", "agdef_a"]);
    expect(lastCall().document).toBe(MyAgentsDocument);
    expect(lastCall().variables.input).toEqual({
      first: 3,
      origin: "All",
      ids: ["agdef_c", "agdef_gone", "agdef_a"],
    });
  });

  it("asks about at most as many ids as the server accepts", async () => {
    requestGraphQLMock.mockResolvedValue(myAgents([], { hasNextPage: false, endCursor: null }));
    const ids = Array.from({ length: MY_AGENTS_LIMIT + 5 }, (_, index) => `agdef_${index}`);

    await fetchAgentChoicesByIds(ids);

    const input = lastCall().variables.input as { first: number; ids: string[] };
    expect(input.first).toBe(MY_AGENTS_LIMIT);
    expect(input.ids).toHaveLength(MY_AGENTS_LIMIT);
    expect(input.ids[0]).toBe("agdef_0");
  });
});

describe("fetchMyAgents", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("reads the whole list the person may ask in one page, without a count", async () => {
    requestGraphQLMock.mockResolvedValue(
      myAgents([agent("agdef_1", "Help")], { hasNextPage: false, endCursor: null }),
    );

    const agents = await fetchMyAgents();

    expect(lastCall().document).toBe(MyAgentsDocument);
    expect(lastCall().variables).toEqual({
      input: { first: MY_AGENTS_LIMIT, origin: "All" },
      includeTotalCount: false,
    });
    expect(agents.map((item) => item.name)).toEqual(["Help"]);
  });

  // The thread's agent carries the delegates the person may use, so an old
  // hand-off that saved no mark of its own still draws the delegate's.
  it("keeps each agent's delegates, in the order the server gave them", async () => {
    const delegates = [
      { id: "agdef_rb", name: "Report Builder", icon: "receipt", accent: "teal", template: null },
      {
        id: "agdef_bill",
        name: "Billing",
        icon: "",
        accent: "amber",
        template: "BillingAssistant",
      },
    ];
    requestGraphQLMock.mockResolvedValue(
      myAgents([{ ...agent("agdef_1", "Widgets"), delegates } as ReturnType<typeof agent>], {
        hasNextPage: false,
        endCursor: null,
      }),
    );

    const [widgets] = await fetchMyAgents();

    expect(widgets.delegates).toEqual(delegates);
  });
});
