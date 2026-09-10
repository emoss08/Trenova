import type { FuelCardRow } from "@/lib/graphql/fuel-card";
import { cancelFuelCardSchema, fuelCardFormSchema } from "@trenova/shared/types/fuel-card";
import { describe, expect, it } from "vitest";
import { buildFuelCardDefaults, toFuelCardInput } from "../fuel-card-panel";

const row: FuelCardRow = {
  id: "fcrd_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  provider: "EFS",
  lastFour: "4821",
  label: "Unit 118 card",
  externalCardId: "EFS-TOKEN-9",
  assignedWorkerId: "wrk_1",
  assignedTractorId: "trk_118",
  status: "Suspended",
  expiresAt: 1_800_000_000,
  cancelledAt: null,
  cancelReason: null,
  notes: "Reissued after loss",
  discoveredAt: null,
  version: 3,
  createdAt: 1,
  updatedAt: 2,
  assignedWorker: { id: "wrk_1", wholeName: "Dana Ruiz", firstName: "Dana", lastName: "Ruiz" },
  assignedTractor: { id: "trk_118", code: "118" },
};

describe("fuel card form mapping", () => {
  it("starts a new card active on the most common provider with nothing assigned", () => {
    const defaults = buildFuelCardDefaults(null);
    expect(defaults).toEqual({
      provider: "Comdata",
      lastFour: "",
      label: "",
      externalCardId: null,
      assignedWorkerId: null,
      assignedTractorId: null,
      status: "Active",
      expiresAt: null,
      notes: null,
    });
  });

  it("round-trips a server row through the form and back into a mutation input", () => {
    const defaults = buildFuelCardDefaults(row);
    const parsed = fuelCardFormSchema.safeParse(defaults);
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);

    expect(toFuelCardInput(defaults)).toEqual({
      provider: "EFS",
      lastFour: "4821",
      label: "Unit 118 card",
      externalCardId: "EFS-TOKEN-9",
      assignedWorkerId: "wrk_1",
      assignedTractorId: "trk_118",
      status: "Suspended",
      expiresAt: 1_800_000_000,
      notes: "Reissued after loss",
    });
  });

  it("sends null for cleared optional fields rather than empty strings", () => {
    const input = toFuelCardInput({
      ...buildFuelCardDefaults(row),
      externalCardId: "",
      assignedWorkerId: "",
      assignedTractorId: "",
      expiresAt: null,
      notes: "   ",
    });
    expect(input.externalCardId).toBeNull();
    expect(input.assignedWorkerId).toBeNull();
    expect(input.assignedTractorId).toBeNull();
    expect(input.expiresAt).toBeNull();
    expect(input.notes).toBeNull();
  });

  it("insists on exactly four digits for the last four", () => {
    for (const lastFour of ["482", "48211", "48a1", ""]) {
      const result = fuelCardFormSchema.safeParse({ ...buildFuelCardDefaults(row), lastFour });
      expect(result.success, lastFour).toBe(false);
      expect(result.error?.issues[0]?.path.join(".")).toBe("lastFour");
    }
    expect(
      fuelCardFormSchema.safeParse({ ...buildFuelCardDefaults(row), lastFour: " 4821 " }).success,
    ).toBe(true);
  });

  it("only lets the form set Active or Suspended; cancelling is its own action", () => {
    const result = fuelCardFormSchema.safeParse({
      ...buildFuelCardDefaults(row),
      status: "Cancelled",
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path.join(".")).toBe("status");
  });

  it("requires a label", () => {
    const result = fuelCardFormSchema.safeParse({ ...buildFuelCardDefaults(row), label: "  " });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path.join(".")).toBe("label");
  });
});

describe("cancel fuel card", () => {
  it("needs a reason of at least ten characters", () => {
    expect(cancelFuelCardSchema.safeParse({ reason: "lost" }).success).toBe(false);
    expect(cancelFuelCardSchema.safeParse({ reason: "Card reported lost by driver" }).success).toBe(
      true,
    );
  });
});
