import { beforeEach, describe, expect, it, vi } from "vitest";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { ShipmentService, toPermitInput } from "./shipment";

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL: vi.fn(),
}));

describe("toPermitInput", () => {
  it("sends the cost as a decimal string and absent dates and notes as null", () => {
    expect(
      toPermitInput({
        stateId: "us_tx",
        permitNumber: "TX-778",
        status: "Pending",
        cost: "125.5",
      }),
    ).toEqual({
      stateId: "us_tx",
      permitNumber: "TX-778",
      status: "Pending",
      issuedAt: null,
      expiresAt: null,
      cost: "125.5",
      notes: null,
    });
  });
});

describe("ShipmentService permits and ETA", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
  });

  it("waives a requirement by its id through WaiveShipmentPermitRequirement", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      waiveShipmentPermitRequirement: {
        id: "pmr_1",
        shipmentId: "shp_1",
        stateId: "us_tx",
        status: "Waived",
        routeSequence: 1,
        exceedances: [],
        escorts: [],
        restrictions: [],
        leadTimeDays: 2,
        validityDays: 5,
        estimatedFee: null,
        isSuperload: false,
        satisfiedByPermitId: null,
        waivedById: "usr_1",
        waiverReason: "State escort booked by phone",
        provenance: null,
        derivedAt: 1_700_000_000,
      },
    });

    const requirement = await new ShipmentService().waivePermitRequirement(
      "pmr_1",
      "State escort booked by phone",
    );

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "WaiveShipmentPermitRequirement",
        variables: { requirementId: "pmr_1", reason: "State escort booked by phone" },
      }),
    );
    expect(requirement.status).toBe("Waived");
    expect(requirement.waivedById).toBe("usr_1");
  });

  it("reads a shipment with no projection as an unknown verdict", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ shipment: { id: "shp_1", eta: null } });

    const eta = await new ShipmentService().getEta("shp_1");

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({ operationName: "ShipmentEta", variables: { id: "shp_1" } }),
    );
    expect(eta.verdict).toBe("Unknown");
    expect(eta.stops).toEqual([]);
  });
});
