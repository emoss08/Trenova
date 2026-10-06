import { describe, expect, it } from "vitest";
import type { Shipment } from "@trenova/shared/types/shipment";
import { resolveCoverage } from "../coverage";

function shipment(partial: Record<string, unknown>): Shipment {
  return { id: "shp_1", status: "New", ...partial } as unknown as Shipment;
}

describe("resolveCoverage", () => {
  it("names the driver and unit of an assigned move", () => {
    const coverage = resolveCoverage(
      shipment({
        moves: [
          {
            coverageType: "driver",
            assignment: {
              primaryWorker: { id: "wrk_1", firstName: "Jaime", lastName: "Park" },
              tractor: { code: "T-211" },
            },
          },
        ],
      }),
    );
    expect(coverage).toEqual({
      kind: "driver",
      id: "wrk_1",
      name: "Jaime Park",
      initials: "JP",
      detail: "T-211",
    });
  });

  it("prefers an active carrier assignment and shows its driver or SCAC", () => {
    const coverage = resolveCoverage(
      shipment({
        moves: [
          {
            coverageType: "carrier",
            carrierAssignment: {
              id: "ca_1",
              status: "Confirmed",
              carrier: { id: "car_1", name: "Knight-Swift", scac: "KNIG" },
              externalDriverName: "Sam Ortiz",
            },
          },
        ],
      }),
    );
    expect(coverage).toEqual({
      kind: "carrier",
      id: "car_1",
      name: "Knight-Swift",
      initials: "KS",
      detail: "Sam Ortiz",
    });
  });

  it("reports a live tender as awaiting rather than uncovered", () => {
    expect(resolveCoverage(shipment({ tenderStatus: "Tendered", moves: [{}] }))).toEqual({
      kind: "tendered",
    });
  });

  it("reports an empty move, a canceled carrier and a missing move as uncovered", () => {
    expect(resolveCoverage(shipment({ moves: [{}] }))).toEqual({ kind: "uncovered" });
    expect(
      resolveCoverage(
        shipment({
          moves: [
            {
              coverageType: "carrier",
              carrierAssignment: { id: "ca_2", status: "Canceled", carrier: { id: "c", name: "X" } },
            },
          ],
        }),
      ),
    ).toEqual({ kind: "uncovered" });
    expect(resolveCoverage(shipment({}))).toEqual({ kind: "uncovered" });
  });
});
