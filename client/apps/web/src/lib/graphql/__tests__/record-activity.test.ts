import { DeskRecordActivityDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchRecordActivity } from "../record-activity";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchRecordActivity", () => {
  it("reads what was done to the record newest first, leaving reads out, counting on the first page", async () => {
    requestGraphQLMock.mockResolvedValue({
      auditEntriesByResourceId: {
        edges: [{ node: { id: "ae_1" } }],
        pageInfo: { hasNextPage: true, endCursor: "c1" },
        totalCount: 14,
        totalCountExact: true,
      },
    } as never);

    const first = await fetchRecordActivity("ord_1", { first: 5, after: null });

    expect(first).toEqual({
      rows: [{ id: "ae_1" }],
      total: 14,
      totalExact: true,
      endCursor: "c1",
      hasNextPage: true,
    });
    expect(requestGraphQLMock.mock.calls.at(-1)?.[0]).toMatchObject({
      document: DeskRecordActivityDocument,
      variables: {
        resourceId: "ord_1",
        input: {
          first: 5,
          after: null,
          fieldFilters: [{ field: "operation", operator: "ne", value: "read" }],
          sort: [{ field: "timestamp", direction: "desc" }],
        },
        includeTotalCount: true,
      },
    });

    requestGraphQLMock.mockResolvedValue({
      auditEntriesByResourceId: { edges: [], pageInfo: { hasNextPage: false, endCursor: null } },
    } as never);
    const next = await fetchRecordActivity("ord_1", { first: 5, after: "c1" });
    expect(next.total).toBeNull();
  });
});
