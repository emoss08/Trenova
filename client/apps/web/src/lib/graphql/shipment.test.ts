import { beforeEach, describe, expect, it, vi } from "vitest";
import { UpdateShipmentDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { ShipmentUpdateInput } from "@trenova/shared/types/shipment";
import { updateShipmentGraphQL } from "./shipment";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

// The form holds the rating detail because the billing panel renders it. That
// makes it easy to echo back on save, and echoing it back is exactly what the
// server must never accept: the shipment carries far more of it than the form
// does — the contract, the breakdown, the guardrail — and a locked shipment is
// never re-rated, so whatever is sent is what the invoice would be explained by.
function shipmentPayload(): ShipmentUpdateInput {
  return {
    id: "shp_1",
    version: 4,
    serviceTypeId: "st_1",
    shipmentTypeId: "sht_1",
    customerId: "cus_1",
    formulaTemplateId: "ft_1",
    status: "New",
    entryMethod: "Manual",
    ratingUnit: 1,
    fuelSurchargeLocked: false,
    ratingDetail: {
      formulaTemplateId: "ft_1",
      formulaTemplateName: "Mileage Base",
      expression: "distance * ratePerMile",
      resolvedVariables: { distance: 1240 },
      result: 1400,
      ratedAt: 1_800_000_000,
    },
    moves: [],
  } as unknown as ShipmentUpdateInput;
}

describe("updateShipmentGraphQL", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    requestGraphQLMock.mockResolvedValue({ updateShipment: { id: "shp_1" } });
  });

  it("never sends the rating detail back to the server", async () => {
    await updateShipmentGraphQL("shp_1", shipmentPayload());

    expect(requestGraphQLMock).toHaveBeenCalledTimes(1);
    const call = requestGraphQLMock.mock.calls[0]?.[0] as {
      document: unknown;
      operationName: string;
      variables: { id: string; input: Record<string, unknown> };
    };

    expect(call.document).toBe(UpdateShipmentDocument);
    expect(call.operationName).toBe("UpdateShipment");
    expect(call.variables.id).toBe("shp_1");
    expect(call.variables.input).not.toHaveProperty("ratingDetail");
  });

  it("still sends the fields the shipment owns", async () => {
    await updateShipmentGraphQL("shp_1", shipmentPayload());

    const call = requestGraphQLMock.mock.calls[0]?.[0] as {
      variables: { input: Record<string, unknown> };
    };

    expect(call.variables.input).toMatchObject({
      serviceTypeId: "st_1",
      shipmentTypeId: "sht_1",
      customerId: "cus_1",
      formulaTemplateId: "ft_1",
      version: 4,
    });
  });
});
