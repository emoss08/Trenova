import type { Permit, PermitRequirement } from "@trenova/shared/types/permit";
import { describe, expect, it } from "vitest";
import { permitChecklist } from "../permit-checklist";

// Requirements as GET /shipments/:id/permit-requirements returns them, and permits as
// GET /shipments/:id/permits does (services/tms permit domain).
function requirement(
  id: string | null,
  stateId: string,
  routeSequence: number,
  extra: Partial<PermitRequirement> = {},
): PermitRequirement {
  return {
    id,
    shipmentId: "shp_1",
    stateId,
    status: "Open",
    routeSequence,
    leadTimeDays: 3,
    validityDays: 10,
    estimatedFee: 0,
    isSuperload: false,
    satisfiedByPermitId: null,
    waivedById: null,
    derivedAt: 1,
    ...extra,
  } as PermitRequirement;
}

function permit(id: string, stateId: string): Permit {
  return {
    id,
    shipmentId: "shp_1",
    stateId,
    permitNumber: `${stateId}-1`,
    status: "Active",
  } as Permit;
}

describe("permitChecklist", () => {
  it("lists what is still to file first, in route order, then what is filed or waived", () => {
    const result = permitChecklist(
      [
        requirement("req_nj", "st_nj", 0, { status: "Satisfied", satisfiedByPermitId: "pmt_nj" }),
        requirement("req_oh", "st_oh", 2),
        requirement("req_pa", "st_pa", 1),
        requirement("req_in", "st_in", 3, { status: "Waived", waiverReason: "Escorted move" }),
      ],
      [permit("pmt_nj", "st_nj")],
    );

    expect(result.entries.map((entry) => [entry.key, entry.phase])).toEqual([
      ["req_pa", "todo"],
      ["req_oh", "todo"],
      ["req_nj", "filed"],
      ["req_in", "waived"],
    ]);
    expect(result.open).toBe(2);
    expect(result.resolved).toBe(2);
  });

  it("pairs a satisfied requirement with the permit that satisfied it", () => {
    const result = permitChecklist(
      [requirement("req_nj", "st_nj", 0, { status: "Satisfied", satisfiedByPermitId: "pmt_b" })],
      [permit("pmt_a", "st_nj"), permit("pmt_b", "st_nj")],
    );

    expect(result.entries[0]?.permit?.id).toBe("pmt_b");
    expect(result.entries[1]).toMatchObject({ key: "pmt_a", phase: "filed", requirement: null });
  });

  it("falls back to the state's permit for a requirement saved before the link was kept", () => {
    const result = permitChecklist(
      [requirement("req_nj", "st_nj", 0, { status: "Satisfied" })],
      [permit("pmt_nj", "st_nj")],
    );

    expect(result.entries).toHaveLength(1);
    expect(result.entries[0]?.permit?.id).toBe("pmt_nj");
  });

  it("leaves out requirements a newer derivation superseded", () => {
    const result = permitChecklist(
      [
        requirement("req_old", "st_pa", 0, { status: "Superseded" }),
        requirement("req_pa", "st_pa", 0),
      ],
      [],
    );

    expect(result.entries.map((entry) => entry.key)).toEqual(["req_pa"]);
    expect(result.open + result.resolved).toBe(1);
  });

  it("keys a derived requirement, which has no id yet, by its state and place on the route", () => {
    const result = permitChecklist([requirement(null, "st_pa", 1)], []);

    expect(result.entries[0]?.key).toBe("st_pa-1");
  });

  it("is empty for a load no state needs a permit for", () => {
    expect(permitChecklist([], [])).toEqual({ entries: [], open: 0, resolved: 0 });
  });
});
