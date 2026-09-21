import { describe, expect, it } from "vitest";
import { derivePageContext, stripAppTitle } from "../page-context";

/**
 * The server accepts only the record kinds it lists (domain/agent/pagecontext.go)
 * and rejects anything else with a field error, so the client has to derive a
 * kind it knows the server will take, or send none at all.
 */
describe("derivePageContext", () => {
  it("names the record kind and the open panel record on a list page", () => {
    expect(
      derivePageContext({
        pathname: "/shipment-management/shipments",
        search: "?panelType=edit&panelEntityId=shp_01J",
        title: "Shipments | Acme Freight",
      }),
    ).toEqual({
      path: "/shipment-management/shipments?panelType=edit&panelEntityId=shp_01J",
      entityType: "shipment",
      entityId: "shp_01J",
      title: "Shipments",
    });
  });

  it("reads the legacy entityId parameter too", () => {
    expect(
      derivePageContext({ pathname: "/hr/workers", search: "?entityId=wrk_1", title: "Workers" }),
    ).toMatchObject({ entityType: "worker", entityId: "wrk_1" });
  });

  it("sends the kind without an id when no record is open", () => {
    expect(
      derivePageContext({ pathname: "/billing/queue", search: "", title: "Billing queue" }),
    ).toEqual({
      path: "/billing/queue",
      entityType: "billing_queue_item",
      entityId: "",
      title: "Billing queue",
    });
  });

  it("sends only the path for a page with no record kind the server knows", () => {
    expect(
      derivePageContext({ pathname: "/reports/ifta", search: "?year=2026", title: "IFTA" }),
    ).toEqual({ path: "/reports/ifta?year=2026", entityType: "", entityId: "", title: "IFTA" });
  });

  it("never sends an id it cannot attach to a known kind", () => {
    expect(
      derivePageContext({ pathname: "/reports/ifta", search: "?entityId=x", title: "IFTA" }),
    ).toMatchObject({ entityType: "", entityId: "" });
  });

  it("keeps the path within the server's length limit", () => {
    const search = "?q=" + "a".repeat(600);
    const context = derivePageContext({ pathname: "/hr/workers", search, title: "Workers" });

    expect(context?.path.length).toBeLessThanOrEqual(500);
    expect(context?.path.startsWith("/hr/workers")).toBe(true);
  });

  it("returns null outside the application shell", () => {
    expect(derivePageContext({ pathname: "", search: "", title: "" })).toBeNull();
  });
});

describe("stripAppTitle", () => {
  it("drops the organization suffix the metadata component appends", () => {
    expect(stripAppTitle("Shipments | Acme Freight")).toBe("Shipments");
    expect(stripAppTitle("Acme Freight")).toBe("Acme Freight");
    expect(stripAppTitle("  Dispatch Console | A | B ")).toBe("Dispatch Console | A");
  });
});
