import { DeskRecordShipmentsDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchRecordShipments } from "../record-shipments";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables: { input: Record<string, unknown>; includeTotalCount: boolean };
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

function node(id: string) {
  return { id, proNumber: id.toUpperCase(), status: "New", totalChargeAmount: null, moves: [] };
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchRecordShipments", () => {
  it("reads a customer's freight by its customer, newest first, counting on the first page", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: {
        edges: [{ node: node("shp_1") }, { node: node("shp_2") }],
        pageInfo: { hasNextPage: true, endCursor: "c2" },
        totalCount: 12,
      },
    } as never);
    const controller = new AbortController();

    const page = await fetchRecordShipments(
      { kind: "customer", id: "cus_1" },
      { first: 2, after: null },
      { signal: controller.signal },
    );

    const call = lastCall();
    expect(call.document).toBe(DeskRecordShipmentsDocument);
    expect(call.operationName).toBe("DeskRecordShipments");
    expect(call.signal).toBe(controller.signal);
    expect(call.variables.includeTotalCount).toBe(true);
    expect(call.variables.input).toEqual({
      first: 2,
      after: null,
      expandShipmentDetails: true,
      sort: [{ field: "createdAt", direction: "desc" }],
      fieldFilters: [{ field: "customerId", operator: "eq", value: "cus_1" }],
    });
    expect(page).toEqual({
      rows: [node("shp_1"), node("shp_2")],
      total: 12,
      endCursor: "c2",
      hasNextPage: true,
    });
  });

  it("reads a tractor's loads by its assignments, and leaves the count off later pages", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: {
        edges: [{ node: node("shp_3") }],
        pageInfo: { hasNextPage: false, endCursor: null },
      },
    } as never);

    const page = await fetchRecordShipments(
      { kind: "tractor", id: "tr_1" },
      { first: 5, after: "c2" },
    );

    const call = lastCall();
    expect(call.variables.includeTotalCount).toBe(false);
    expect(call.variables.input).toMatchObject({ first: 5, after: "c2", tractorIds: ["tr_1"] });
    expect(call.variables.input).not.toHaveProperty("fieldFilters");
    expect(page).toEqual({
      rows: [node("shp_3")],
      total: null,
      endCursor: null,
      hasNextPage: false,
    });
  });

  it("counts the rows it got when the server leaves the total out of a first page", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: {
        edges: [{ node: node("shp_1") }],
        pageInfo: { hasNextPage: false, endCursor: "c1" },
        totalCount: null,
      },
    } as never);

    const page = await fetchRecordShipments(
      { kind: "tractor", id: "tr_1" },
      { first: 5, after: null },
    );

    expect(page.total).toBe(1);
  });
});

describe("fetchRecordShipments for a trailer", () => {
  it("reads the loads a trailer carried by its assignments", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: { edges: [], pageInfo: { hasNextPage: false, endCursor: null }, totalCount: 0 },
    } as never);

    await fetchRecordShipments({ kind: "trailer", id: "trl_1" }, { first: 5, after: null });

    expect(lastCall().variables.input).toMatchObject({ trailerIds: ["trl_1"] });
    expect(lastCall().variables.input).not.toHaveProperty("tractorIds");
    expect(lastCall().variables.input).not.toHaveProperty("fieldFilters");
  });
});

describe("fetchRecordShipments for a worker or a location", () => {
  it("reads the loads a worker drove by their assignments", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: { edges: [], pageInfo: { hasNextPage: false, endCursor: null }, totalCount: 0 },
    } as never);

    await fetchRecordShipments({ kind: "worker", id: "wrk_1" }, { first: 5, after: null });

    expect(lastCall().variables.input).toMatchObject({ workerIds: ["wrk_1"] });
    expect(lastCall().variables.input).not.toHaveProperty("stopLocationIds");
    expect(lastCall().variables.input).not.toHaveProperty("fieldFilters");
  });

  it("reads the loads with a stop at a location", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: { edges: [], pageInfo: { hasNextPage: false, endCursor: null }, totalCount: 0 },
    } as never);

    await fetchRecordShipments({ kind: "location", id: "loc_1" }, { first: 5, after: null });

    expect(lastCall().variables.input).toMatchObject({ stopLocationIds: ["loc_1"] });
    expect(lastCall().variables.input).not.toHaveProperty("workerIds");
    expect(lastCall().variables.input).not.toHaveProperty("fieldFilters");
  });
});

describe("fetchRecordShipments for a carrier", () => {
  it("reads the loads tendered to a carrier", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: { edges: [], pageInfo: { hasNextPage: false, endCursor: null }, totalCount: 0 },
    } as never);

    await fetchRecordShipments({ kind: "carrier", id: "car_1" }, { first: 5, after: null });

    expect(lastCall().variables.input).toMatchObject({ carrierIds: ["car_1"] });
    expect(lastCall().variables.input).not.toHaveProperty("stopLocationIds");
  });
});

describe("fetchRecordShipments for an order", () => {
  it("reads an order's legs by the order they belong to", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: { edges: [], pageInfo: { hasNextPage: false, endCursor: null }, totalCount: 0 },
    } as never);

    await fetchRecordShipments({ kind: "order", id: "ord_1" }, { first: 5, after: null });

    expect(lastCall().variables.input).toMatchObject({
      fieldFilters: [{ field: "orderId", operator: "eq", value: "ord_1" }],
    });
    expect(lastCall().variables.input).not.toHaveProperty("carrierIds");
  });
});

describe("fetchRecordShipments for an invoice", () => {
  it("reads the shipments an invoice bills by the invoice", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: { edges: [], pageInfo: { hasNextPage: false, endCursor: null }, totalCount: 0 },
    } as never);

    await fetchRecordShipments({ kind: "invoice", id: "inv_1" }, { first: 5, after: null });

    expect(lastCall().variables.input).toMatchObject({ invoiceIds: ["inv_1"] });
    expect(lastCall().variables.input).not.toHaveProperty("fieldFilters");
  });
});

describe("fetchRecordShipments with openOnly", () => {
  beforeEach(() => {
    requestGraphQLMock.mockResolvedValue({
      shipments: { edges: [], pageInfo: { hasNextPage: false, endCursor: null }, totalCount: 0 },
    } as never);
  });

  it("keeps a customer's own filter and adds the open statuses after it", async () => {
    await fetchRecordShipments(
      { kind: "customer", id: "cus_1" },
      { first: 3, after: null, openOnly: true },
    );

    expect(lastCall().variables.input.fieldFilters).toEqual([
      { field: "customerId", operator: "eq", value: "cus_1" },
      {
        field: "status",
        operator: "notin",
        value: ["Completed", "ReadyToInvoice", "Invoiced", "Canceled"],
      },
    ]);
  });

  it("filters a worker's loads to the open ones alongside the assignment scope", async () => {
    await fetchRecordShipments(
      { kind: "worker", id: "wrk_1" },
      { first: 1, after: null, openOnly: true },
    );

    expect(lastCall().variables.input).toMatchObject({
      workerIds: ["wrk_1"],
      fieldFilters: [
        {
          field: "status",
          operator: "notin",
          value: ["Completed", "ReadyToInvoice", "Invoiced", "Canceled"],
        },
      ],
    });
  });
});
