import {
  BillingTransferCandidateIdsDocument,
  BillingTransferCandidatesDocument,
  BulkTransferShipmentsToBillingDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  bulkTransferShipmentsToBillingGraphQL,
  listBillingTransferCandidateIdsGraphQL,
  listBillingTransferCandidatesGraphQL,
} from "../billing-transfer";

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

describe("billing transfer GraphQL", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("asks only for shipments that can still transfer, with their customer", async () => {
    const connection = {
      edges: [],
      totalCount: 0,
      pageInfo: { hasNextPage: false, endCursor: null },
    };
    requestGraphQLMock.mockResolvedValue({ shipments: connection });
    const controller = new AbortController();

    const result = await listBillingTransferCandidatesGraphQL(
      { first: 50, after: "cursor-1", query: "PRO-9", status: "Completed" },
      { signal: controller.signal },
    );

    expect(result).toBe(connection);
    const call = lastCall();
    expect(call.document).toBe(BillingTransferCandidatesDocument);
    expect(call.operationName).toBe("BillingTransferCandidates");
    expect(call.signal).toBe(controller.signal);
    expect(call.variables.input).toEqual({
      first: 50,
      after: "cursor-1",
      query: "PRO-9",
      status: "Completed",
      billingTransferEligible: true,
      includeCustomer: true,
      expandShipmentDetails: false,
    });
    expect(call.variables.includeTotalCount).toBe(false);
  });

  it("sends no status or search when the list is unfiltered", async () => {
    requestGraphQLMock.mockResolvedValue({
      shipments: { edges: [], totalCount: 0, pageInfo: { hasNextPage: false, endCursor: null } },
    });

    await listBillingTransferCandidatesGraphQL({
      first: 50,
      after: null,
      query: "  ",
      status: null,
    });

    expect(lastCall().variables.includeTotalCount).toBe(true);
    const input = lastCall().variables.input;
    expect(input.after).toBeUndefined();
    expect(input.query).toBeUndefined();
    expect(input.status).toBeUndefined();
  });

  it("resolves every matching candidate id with the same filters as the list", async () => {
    const payload = { ids: ["shp_1"], totalCount: 1, truncated: false };
    requestGraphQLMock.mockResolvedValue({ shipmentBillingTransferCandidateIds: payload });
    const controller = new AbortController();

    const result = await listBillingTransferCandidateIdsGraphQL(
      { query: " BOL-1 ", status: "ReadyToInvoice" },
      { signal: controller.signal },
    );

    expect(result).toBe(payload);
    const call = lastCall();
    expect(call.document).toBe(BillingTransferCandidateIdsDocument);
    expect(call.operationName).toBe("BillingTransferCandidateIds");
    expect(call.signal).toBe(controller.signal);
    expect(call.variables.input).toEqual({ query: "BOL-1", status: "ReadyToInvoice" });
  });

  it("transfers as invoices and lets completed shipments be marked ready first", async () => {
    const response = { results: [], totalCount: 0, successCount: 0, errorCount: 0 };
    requestGraphQLMock.mockResolvedValue({ bulkTransferShipmentsToBilling: response });

    const result = await bulkTransferShipmentsToBillingGraphQL(["shp_1", "shp_2"]);

    expect(result).toBe(response);
    const call = lastCall();
    expect(call.document).toBe(BulkTransferShipmentsToBillingDocument);
    expect(call.operationName).toBe("BulkTransferShipmentsToBilling");
    expect(call.variables.input).toEqual({
      shipmentIds: ["shp_1", "shp_2"],
      billType: "Invoice",
      markCompletedReadyToInvoice: true,
    });
  });
});
