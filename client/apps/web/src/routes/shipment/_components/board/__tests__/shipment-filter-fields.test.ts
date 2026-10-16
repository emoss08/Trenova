import { translate } from "@trenova/shared/i18n/runtime";
import { getShipmentFilterFields } from "../shipment-columns";
import { describe, expect, it } from "vitest";

// The server resolves `shipperStop` and `consigneeStop` (Shipment.GetPostgresSearchConfig)
// to the shipment's first pickup and last delivery stop, one row each, so a location
// picked here is matched against the stop's `locationId`.
describe("shipment board filter fields", () => {
  const fields = getShipmentFilterFields(translate);
  const byLabel = (label: string) => fields.find((field) => field.label === label);

  it("offers the pickup location, matched on the shipper stop's location", () => {
    expect(byLabel("Pickup location")).toEqual({
      apiField: "shipperStop.locationId",
      label: "Pickup location",
      filterType: "record",
      filterRecord: "LOCATION",
    });
  });

  it("offers the delivery location, matched on the consignee stop's location", () => {
    expect(byLabel("Delivery location")).toEqual({
      apiField: "consigneeStop.locationId",
      label: "Delivery location",
      filterType: "record",
      filterRecord: "LOCATION",
    });
  });

  it("lists the locations right after the bill-to customer", () => {
    expect(fields.slice(0, 3).map((field) => field.apiField)).toEqual([
      "billToCustomerId",
      "shipperStop.locationId",
      "consigneeStop.locationId",
    ]);
  });

  it("names every filter field once", () => {
    const apiFields = fields.map((field) => field.apiField);

    expect(new Set(apiFields).size).toBe(apiFields.length);
  });
});
