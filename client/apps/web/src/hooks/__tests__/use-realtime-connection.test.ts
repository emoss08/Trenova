import { patchEntityInListRows, shouldPatchEvent } from "@/hooks/realtime-patching";
import { describe, expect, it } from "vitest";

describe("realtime shipment patching", () => {
  it("replaces cached shipment moves with incoming moves", () => {
    const current = {
      results: [
        {
          id: "shp_1",
          status: "Assigned",
          moves: [
            {
              id: "sm_old",
              status: "Assigned",
              stops: [{ id: "stp_old", status: "New" }],
            },
          ],
        },
      ],
    };
    const nextMoves = [
      {
        id: "sm_new",
        status: "InTransit",
        stops: [{ id: "stp_new", status: "InTransit" }],
      },
    ];
    const event = {
      organizationId: "org_1",
      businessUnitId: "bu_1",
      resource: "shipments",
      action: "updated",
      recordId: "shp_1",
      fields: ["moves", "updatedAt"],
      entity: {
        id: "shp_1",
        moves: nextMoves,
        updatedAt: 123,
      },
    };

    expect(shouldPatchEvent(event)).toBe(true);

    const result = patchEntityInListRows(current, event);

    expect(result.patched).toBe(true);
    expect((result.data as typeof current).results[0].moves).toBe(nextMoves);
    expect((result.data as typeof current).results[0].moves).not.toEqual(current.results[0].moves);
  });
});
