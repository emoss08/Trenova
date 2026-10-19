import { describe, expect, it } from "vitest";
import { toolNameInReadingOrder } from "../tool-names";

describe("toolNameInReadingOrder", () => {
  it("puts a namespaced tool's verb first", () => {
    expect(toolNameInReadingOrder("shipment_create")).toBe("create_shipment");
    expect(toolNameInReadingOrder("shipment_place_hold")).toBe("place_shipment_hold");
    expect(toolNameInReadingOrder("order_list")).toBe("list_orders");
    expect(toolNameInReadingOrder("shipment_list_permits")).toBe("list_shipment_permits");
  });

  it("leaves a name outside a namespace as it is", () => {
    expect(toolNameInReadingOrder("list_workers")).toBe("list_workers");
    expect(toolNameInReadingOrder("remember")).toBe("remember");
  });
});
