import { describe, expect, it } from "vitest";
import { shipmentPanelPath } from "@/lib/shipment-utils";

describe("shipmentPanelPath", () => {
  it("expands the shipment's row and opens its edit panel", () => {
    expect(shipmentPanelPath("shp_01M29TH78AH2BQ7NP65RE51PQN")).toBe(
      "/shipment-management/shipments?expanded=shp_01M29TH78AH2BQ7NP65RE51PQN&panelType=edit&panelEntityId=shp_01M29TH78AH2BQ7NP65RE51PQN",
    );
  });

  it("encodes an id that is not URL-safe rather than splicing it into the query", () => {
    expect(shipmentPanelPath("a&b=c")).toBe(
      "/shipment-management/shipments?expanded=a%26b%3Dc&panelType=edit&panelEntityId=a%26b%3Dc",
    );
  });
});
