import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import { billingQueueLaneQuery, laneItems, laneTotal } from "../../billing-queue-queries";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: mocks.requestGraphQL }));

afterEach(() => vi.clearAllMocks());

const filters = {
  status: null,
  billers: ["usr_1", "usr_2"],
  billType: "Invoice",
  payer: "cus_acme",
  search: "  PRO-24118 ",
  includeClosed: false,
  minAmount: "5000",
  maxAmount: null,
  createdBefore: 1_800_000_000,
};

function connection(ids: string[], next: string | null, totalCount?: number) {
  return {
    billingQueueItems: {
      edges: ids.map((id) => ({ cursor: `c-${id}`, node: { id } })),
      totalCount,
      totalCountExact: true,
      pageInfo: { hasNextPage: next !== null, endCursor: next },
    },
  };
}

describe("billingQueueLaneQuery", () => {
  it("narrows a lane to one customer's shipments and to the dates asked for", async () => {
    mocks.requestGraphQL.mockResolvedValueOnce(connection([], null, 0));
    const client = new QueryClient();

    await client.fetchInfiniteQuery(
      billingQueueLaneQuery("Ready", {
        status: null,
        billers: [],
        billType: null,
        search: "",
        includeClosed: false,
        shipmentCustomer: "cus_shipper",
        delivered: { kind: "last", days: 7 },
        pickedUp: { kind: "range", from: 1_793_000_000, to: 1_793_500_000 },
        queued: { kind: "last", days: 30 },
      }),
    );

    expect(mocks.requestGraphQL.mock.calls[0]![0].variables.input.fieldFilters).toEqual([
      { field: "shipment.customerId", operator: "eq", value: "cus_shipper" },
      { field: "shipment.actualDeliveryDate", operator: "lastndays", value: 7 },
      {
        field: "shipment.actualShipDate",
        operator: "daterange",
        value: { from: 1_793_000_000, to: 1_793_500_000 },
      },
      { field: "createdAt", operator: "lastndays", value: 30 },
    ]);
  });

  it("asks GraphQL for one lane with the filters, counting only the first page", async () => {
    mocks.requestGraphQL
      .mockResolvedValueOnce(connection(["bqi_1", "bqi_2"], "c-bqi_2", 3))
      .mockResolvedValueOnce(connection(["bqi_3"], null));
    const client = new QueryClient();
    const options = billingQueueLaneQuery("Ready", filters);

    const data = await client.fetchInfiniteQuery({ ...options, pages: 2 });
    expect(mocks.requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "BillingQueueItems",
        variables: {
          input: {
            first: 25,
            after: undefined,
            query: "PRO-24118",
            fieldFilters: [
              { field: "assignedBillerId", operator: "in", value: ["usr_1", "usr_2"] },
              { field: "billType", operator: "eq", value: "Invoice" },
              { field: "billToCustomerId", operator: "eq", value: "cus_acme" },
            ],
          },
          filter: {
            lanes: ["Ready"],
            includeClosed: false,
            minAmount: "5000",
            maxAmount: undefined,
            createdBefore: 1_800_000_000,
          },
          includeTotalCount: true,
        },
      }),
    );
    expect(mocks.requestGraphQL).toHaveBeenCalledTimes(2);
    const secondCall = mocks.requestGraphQL.mock.calls[1][0] as {
      variables: { input: { after?: string }; includeTotalCount: boolean };
    };
    expect(secondCall.variables.input.after).toBe("c-bqi_2");
    expect(secondCall.variables.includeTotalCount).toBe(false);
    expect(laneItems(data).map((row) => row.id)).toEqual(["bqi_1", "bqi_2", "bqi_3"]);
    expect(laneTotal(data)).toBe(3);
  });

  it("keys a lane by its trimmed filters so a trailing space reuses the cached page", () => {
    expect(billingQueueLaneQuery("Ready", filters).queryKey).toEqual(
      billingQueueLaneQuery("Ready", { ...filters, search: "PRO-24118" }).queryKey,
    );
  });
});
