import {
  AgentSafetyDocument,
  AgentToolPolicyConnectionDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  ALL_TOOL_POLICIES,
  fetchAgentSafety,
  fetchToolPolicyPage,
  toolPolicyConnectionInput,
} from "../agent-safety";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables: Record<string, unknown>;
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("toolPolicyConnectionInput", () => {
  // An open filter is left out rather than sent as "all" or "": the server
  // reads an absent filter as every tool, and a page keyed on what is sent
  // is cached once however the filter was spelled.
  it("sends only the page size for an open filter on the first page", () => {
    expect(toolPolicyConnectionInput(ALL_TOOL_POLICIES, { first: 25, after: null })).toEqual({
      first: 25,
    });
    expect(
      toolPolicyConnectionInput({ ...ALL_TOOL_POLICIES, search: "   " }, { first: 25, after: "" }),
    ).toEqual({ first: 25 });
  });

  it("sends every filter that narrows, and the cursor to read after", () => {
    expect(
      toolPolicyConnectionInput(
        {
          search: "  email customer ",
          egress: "ExternalRecipient",
          resource: "general",
          kind: "Action",
          attendance: "alone",
        },
        { first: 50, after: "cursor-25" },
      ),
    ).toEqual({
      first: 50,
      after: "cursor-25",
      query: "email customer",
      egress: "ExternalRecipient",
      resource: "general",
      kind: "Action",
      runsWithoutPerson: true,
    });
  });

  // Attended is a filter of its own, not the absence of one: false keeps the
  // tools that never run unattended, which "all" would not.
  it("keeps attended apart from unfiltered", () => {
    expect(
      toolPolicyConnectionInput(
        { ...ALL_TOOL_POLICIES, attendance: "attended" },
        { first: 25, after: null },
      ),
    ).toEqual({ first: 25, runsWithoutPerson: false });
  });
});

describe("fetchToolPolicyPage", () => {
  it("asks for the count only when told to and reads the page back", async () => {
    const signal = new AbortController().signal;
    requestGraphQLMock.mockResolvedValue({
      agentToolPolicyConnection: {
        edges: [
          { cursor: "c1", node: { name: "assign_move" } },
          { cursor: "c2", node: { name: "email_customer" } },
        ],
        totalCount: 40,
        pageInfo: { hasNextPage: true, endCursor: "c2" },
      },
    });

    const page = await fetchToolPolicyPage(
      { ...ALL_TOOL_POLICIES, egress: "Money" },
      { first: 25, after: "c0", includeTotalCount: true },
      { signal },
    );

    const call = lastCall();
    expect(call.document).toBe(AgentToolPolicyConnectionDocument);
    expect(call.operationName).toBe("AgentToolPolicyConnection");
    expect(call.signal).toBe(signal);
    expect(call.variables).toEqual({
      input: { first: 25, after: "c0", egress: "Money" },
      includeTotalCount: true,
    });
    expect(page).toEqual({
      items: [{ name: "assign_move" }, { name: "email_customer" }],
      endCursor: "c2",
      hasNextPage: true,
      totalCount: 40,
    });
  });

  // A count that was not selected is absent from the response, not zero.
  it("reads an unselected count as unknown", async () => {
    requestGraphQLMock.mockResolvedValue({
      agentToolPolicyConnection: {
        edges: [],
        pageInfo: { hasNextPage: false, endCursor: null },
      },
    });

    const page = await fetchToolPolicyPage(ALL_TOOL_POLICIES, {
      first: 25,
      after: "c9",
      includeTotalCount: false,
    });

    expect(lastCall().variables).toEqual({
      input: { first: 25, after: "c9" },
      includeTotalCount: false,
    });
    expect(page).toEqual({ items: [], endCursor: null, hasNextPage: false, totalCount: null });
  });
});

describe("fetchAgentSafety", () => {
  // Null agentIds reads every agent on the server; this wrapper never sends it.
  it("reads nothing for no agents rather than every agent", async () => {
    await expect(fetchAgentSafety([])).resolves.toEqual([]);
    expect(requestGraphQLMock).not.toHaveBeenCalled();
  });

  it("names the agents it reads and keeps each tool's rule", async () => {
    requestGraphQLMock.mockResolvedValue({
      agentSafety: [
        {
          agentId: "agdef_1",
          tools: [
            {
              policyName: "assign_move",
              policy: { name: "assign_move", title: "Assign move" },
              clean: { answer: "RUNS_ON_ITS_OWN" },
              tainted: { answer: "NEEDS_APPROVAL" },
            },
          ],
        },
      ],
    });

    const agents = await fetchAgentSafety(["agdef_1"]);

    expect(lastCall().document).toBe(AgentSafetyDocument);
    expect(lastCall().variables).toEqual({ agentIds: ["agdef_1"] });
    expect(agents[0]?.tools[0]?.policy.title).toBe("Assign move");
    expect(agents[0]?.tools[0]?.tainted.answer).toBe("NEEDS_APPROVAL");
  });
});
