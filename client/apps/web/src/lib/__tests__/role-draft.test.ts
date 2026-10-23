import { describe, expect, it } from "vitest";
import { toRoleDraft } from "../role-draft";

describe("toRoleDraft", () => {
  it("sends the same draft for the same grants in any order", () => {
    const one = toRoleDraft(
      "internal",
      [
        { resource: "shipment", operations: ["update", "read"] },
        { resource: "customer", operations: ["read"] },
      ],
      ["rol_b", "rol_a"],
    );
    const other = toRoleDraft(
      "internal",
      [
        { resource: "customer", operations: ["read"] },
        { resource: "shipment", operations: ["read", "update"] },
      ],
      ["rol_a", "rol_b"],
    );

    expect(one).toEqual(other);
    expect(one).toEqual({
      maxSensitivity: "internal",
      parentRoleIds: ["rol_a", "rol_b"],
      permissions: [
        { resource: "customer", operations: ["read"] },
        { resource: "shipment", operations: ["read", "update"] },
      ],
    });
  });

  it("inherits no role unless told to", () => {
    expect(toRoleDraft("public", []).parentRoleIds).toEqual([]);
  });
});
