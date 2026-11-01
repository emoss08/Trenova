import { beforeEach, describe, expect, it, vi } from "vitest";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { RoutingGuidePayload } from "@trenova/shared/types/routing-guide";
import { RoutingGuideService, toRoutingGuideInput } from "./routing-guide";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

function guide(): RoutingGuidePayload {
  return {
    version: 3,
    name: "ATL to CHI",
    description: null,
    status: "Active",
    originLocationId: null,
    destinationLocationId: null,
    originCity: "Atlanta",
    originState: "GA",
    destinationCity: "Chicago",
    destinationState: "IL",
    entries: [
      {
        id: "rge_1",
        version: 2,
        carrierId: "car_1",
        rank: 1,
        rateMethod: "Flat",
        rate: 1850.5,
        offerTtlSeconds: 3600,
        channel: "Email",
        useContractRate: true,
      },
    ],
  } as RoutingGuidePayload;
}

describe("toRoutingGuideInput", () => {
  it("sends each entry's contract rate choice, rate as a decimal string and the versions read", () => {
    expect(toRoutingGuideInput(guide())).toEqual({
      version: 3,
      name: "ATL to CHI",
      description: "",
      status: "Active",
      originLocationId: null,
      destinationLocationId: null,
      originCity: "Atlanta",
      originState: "GA",
      destinationCity: "Chicago",
      destinationState: "IL",
      entries: [
        {
          id: "rge_1",
          version: 2,
          carrierId: "car_1",
          rank: 1,
          rateMethod: "Flat",
          rate: "1850.5",
          useContractRate: true,
          offerTtlSeconds: 3600,
          channel: "Email",
        },
      ],
    });
  });
});

describe("RoutingGuideService", () => {
  beforeEach(() => {
    vi.mocked(requestGraphQL).mockReset();
  });

  it("updates through the mutation and takes the version the server saved", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ updateRoutingGuide: { id: "rg_1", version: 4 } });

    const saved = await new RoutingGuideService().update("rg_1", guide());

    expect(vi.mocked(requestGraphQL).mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({
        operationName: "UpdateRoutingGuide",
        variables: expect.objectContaining({ id: "rg_1" }),
      }),
    );
    expect(saved).toEqual(expect.objectContaining({ id: "rg_1", version: 4 }));
  });

  it("deletes through the mutation", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ deleteRoutingGuide: true });

    await new RoutingGuideService().delete("rg_1");

    expect(vi.mocked(requestGraphQL).mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({ operationName: "DeleteRoutingGuide", variables: { id: "rg_1" } }),
    );
  });
});
