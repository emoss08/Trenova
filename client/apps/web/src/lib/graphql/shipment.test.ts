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
// does — the contract, the bill, where each input came from — and a locked shipment is
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
    fuelSurchargeLocked: false,
    tariffInputs: { rate: { $q: "2.35", u: "$/mi" }, breaks: [] },
    freightLines: [{ label: "Linehaul", quantity: 812, rate: 2.35, amount: 1908.2 }],
    ratingDetail: {
      formulaTemplateId: "ft_1",
      formulaTemplateName: "Mileage Base",
      ratedAt: 1_800_000_000,
      bill: null,
      inputs: { rate: { $q: "2.35", u: "$/mi" } },
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

  it("never sends the freight lines back, since the rating writes them", async () => {
    await updateShipmentGraphQL("shp_1", shipmentPayload());

    const call = requestGraphQLMock.mock.calls[0]?.[0] as {
      variables: { input: Record<string, unknown> };
    };

    expect(call.variables.input).not.toHaveProperty("freightLines");
  });

  it("sends the tariff's input values in their wire form, and no base rate", async () => {
    await updateShipmentGraphQL("shp_1", shipmentPayload());

    const call = requestGraphQLMock.mock.calls[0]?.[0] as {
      variables: { input: Record<string, unknown> };
    };

    expect(call.variables.input.tariffInputs).toEqual({
      rate: { $q: "2.35", u: "$/mi" },
      breaks: [],
    });
    expect(call.variables.input).not.toHaveProperty("baseRate");
    expect(call.variables.input).not.toHaveProperty("ratingUnit");
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
