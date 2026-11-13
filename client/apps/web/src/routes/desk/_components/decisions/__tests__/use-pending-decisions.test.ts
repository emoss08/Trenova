import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { pendingDecisionsOptions } from "../use-pending-decisions";

const requestGraphQL = vi.hoisted(() => vi.fn());

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL }));

const EMPTY_PAGE = {
  pendingDecisions: {
    edges: [],
    totalCount: 0,
    totalCountExact: true,
    pageInfo: { hasNextPage: false, endCursor: null },
  },
};

function inputOf(call: number) {
  return (requestGraphQL.mock.calls[call][0] as { variables: { input: unknown } }).variables.input;
}

describe("the decision queue's read", () => {
  beforeEach(() => {
    requestGraphQL.mockReset();
    requestGraphQL.mockResolvedValue(EMPTY_PAGE);
  });

  it("asks the server for the words searched, the agent and the kind of change", async () => {
    const client = new QueryClient();

    await client.fetchInfiniteQuery(
      pendingDecisionsOptions({
        agentDefinitionId: "agd_billing",
        toolName: "reassign_move",
        search: "  acme  ",
      }),
    );

    expect(inputOf(0)).toEqual({
      first: 25,
      after: null,
      agentDefinitionId: "agd_billing",
      toolName: "reassign_move",
      search: "acme",
    });
  });

  it("asks for the whole queue when nothing narrows it", async () => {
    const client = new QueryClient();

    await client.fetchInfiniteQuery(pendingDecisionsOptions({}));

    expect(inputOf(0)).toEqual({
      first: 25,
      after: null,
      agentDefinitionId: null,
      toolName: null,
      search: null,
    });
  });

  it("keeps a search apart from the queue it narrows, and from the same words spaced out", () => {
    const whole = pendingDecisionsOptions({}).queryKey;
    const searched = pendingDecisionsOptions({ search: "acme" }).queryKey;

    expect(searched).not.toEqual(whole);
    expect(pendingDecisionsOptions({ search: " acme " }).queryKey).toEqual(searched);
  });
});
