import { AgentChoicesDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { agentChoiceFilters, fetchAgentChoices, fetchAgentChoicesByIds } from "../agent-definition";

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

const connection = (
  nodes: ReturnType<typeof agent>[],
  pageInfo: { hasNextPage: boolean; endCursor: string | null },
  totalCount?: number | null,
) => ({
  agentDefinitions: {
    edges: nodes.map((node) => ({ node })),
    pageInfo,
    ...(totalCount === undefined ? {} : { totalCount }),
  },
});

const ASKABLE = [
  { field: "enabled", operator: "eq", value: true },
  { field: "triggerMode", operator: "eq", value: "Chat" },
];

describe("agentChoiceFilters", () => {
  it("keeps to enabled agents a person starts", () => {
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

  it("sends the search, the cursor and the count flag, and reads the page back", async () => {
    requestGraphQLMock.mockResolvedValue(
      connection(
        [agent("agdef_1", "Report Builder"), agent("agdef_2", "Help")],
        {
          hasNextPage: true,
          endCursor: "cursor-2",
        },
        42,
      ),
    );
    const controller = new AbortController();

    const page = await fetchAgentChoices(
      { search: "  report ", origin: "custom" },
      { first: 30, after: "cursor-1", includeTotalCount: true },
      { signal: controller.signal },
    );

    const call = lastCall();
    expect(call.document).toBe(AgentChoicesDocument);
    expect(call.operationName).toBe("AgentChoices");
    expect(call.signal).toBe(controller.signal);
    expect(call.variables.includeTotalCount).toBe(true);
    expect(call.variables.input).toEqual({
      first: 30,
      after: "cursor-1",
      query: "report",
      fieldFilters: [...ASKABLE, { field: "template", operator: "isnull", value: null }],
      sort: [{ field: "name", direction: "asc" }],
    });
    expect(page.items.map((item) => item.id)).toEqual(["agdef_1", "agdef_2"]);
    expect(page.items[0].starters).toEqual([
      { label: "Build a report", prompt: "Build a report." },
    ]);
    expect(page).toMatchObject({ endCursor: "cursor-2", hasNextPage: true, totalCount: 42 });
  });

  it("sends no search or cursor when there is none, and reads an absent count as unknown", async () => {
    requestGraphQLMock.mockResolvedValue(connection([], { hasNextPage: false, endCursor: null }));

    const page = await fetchAgentChoices({ search: "   " }, { first: 30 });

    expect(lastCall().variables.input.query).toBeUndefined();
    expect(lastCall().variables.input.after).toBeUndefined();
    expect(lastCall().variables.includeTotalCount).toBe(false);
    expect(page).toEqual({ items: [], endCursor: null, hasNextPage: false, totalCount: null });
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

  it("returns the agents in the order asked, dropping ones that are gone", async () => {
    requestGraphQLMock.mockResolvedValue(
      connection([agent("agdef_a", "Alpha"), agent("agdef_c", "Charlie")], {
        hasNextPage: false,
        endCursor: null,
      }),
    );

    const agents = await fetchAgentChoicesByIds(["agdef_c", "agdef_gone", "agdef_a"]);

    expect(agents.map((item) => item.id)).toEqual(["agdef_c", "agdef_a"]);
    expect(lastCall().variables.input).toMatchObject({
      first: 3,
      fieldFilters: [
        ...ASKABLE,
        { field: "id", operator: "in", value: ["agdef_c", "agdef_gone", "agdef_a"] },
      ],
    });
  });
});
