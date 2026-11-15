import { describe, expect, it } from "vitest";
import { shipmentTypeInlinePatch } from "../shipment-type-inline-edit";

describe("shipmentTypeInlinePatch", () => {
  it("saves a colour picked in the table", () => {
    expect(shipmentTypeInlinePatch("color", "#2e8b57")).toEqual({ color: "#2e8b57" });
  });

  it("saves a colour taken away as none rather than as null, which would keep the old one", () => {
    expect(shipmentTypeInlinePatch("color", null)).toEqual({ color: "" });
  });

  it("saves the code and description edited in place", () => {
    expect(shipmentTypeInlinePatch("code", "LTL")).toEqual({ code: "LTL" });
    expect(shipmentTypeInlinePatch("description", "Less than truckload")).toEqual({
      description: "Less than truckload",
    });
  });

  it("refuses a code left empty", () => {
    expect(() => shipmentTypeInlinePatch("code", "")).toThrow("Code is required");
    expect(() => shipmentTypeInlinePatch("code", null)).toThrow("Code is required");
  });

  it("saves nothing for a column the table does not edit", () => {
    expect(shipmentTypeInlinePatch("status", "Inactive")).toBeNull();
  });
});
