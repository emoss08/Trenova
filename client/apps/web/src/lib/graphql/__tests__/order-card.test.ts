import { DeskOrderCardDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchDeskOrder } from "../order-card";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchDeskOrder", () => {
  it("reads one order by id, passing the signal on", async () => {
    const signal = new AbortController().signal;
    requestGraphQLMock.mockResolvedValue({ order: { id: "ord_1", orderNumber: "ORD-1" } } as never);

    expect(await fetchDeskOrder("ord_1", { signal })).toMatchObject({ orderNumber: "ORD-1" });
    expect(requestGraphQLMock.mock.calls.at(-1)?.[0]).toMatchObject({
      document: DeskOrderCardDocument,
      operationName: "DeskOrderCard",
      variables: { id: "ord_1" },
      signal,
    });
  });

  it("says an order the server does not return could not be found", async () => {
    requestGraphQLMock.mockResolvedValue({ order: null } as never);

    await expect(fetchDeskOrder("ord_gone")).rejects.toThrow("Order not found");
  });
});
