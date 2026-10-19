import { describe, expect, it } from "vitest";
import { toolNameInReadingOrder } from "../tool-names";

describe("toolNameInReadingOrder", () => {
  it("puts a namespaced tool's verb first", () => {
    expect(toolNameInReadingOrder("shipment_create")).toBe("create_shipment");
    expect(toolNameInReadingOrder("shipment_place_hold")).toBe("place_shipment_hold");
    expect(toolNameInReadingOrder("order_list")).toBe("list_orders");
    expect(toolNameInReadingOrder("shipment_list_permits")).toBe("list_shipment_permits");
  });

  it("reads a namespace's qualifier before it", () => {
    expect(toolNameInReadingOrder("settlement_get_driver")).toBe("get_driver_settlement");
    expect(toolNameInReadingOrder("settlement_list_carrier")).toBe("list_carrier_settlements");
    expect(toolNameInReadingOrder("settlement_add_driver_adjustment")).toBe(
      "add_driver_settlement_adjustment",
    );
    expect(toolNameInReadingOrder("settlement_list_pay_events")).toBe("list_settlement_pay_events");
  });

  it("reads a record the namespace names on its own without the namespace", () => {
    expect(toolNameInReadingOrder("receivable_get_bank_receipt")).toBe("get_bank_receipt");
    expect(toolNameInReadingOrder("receivable_list_customer_payments")).toBe(
      "list_customer_payments",
    );
    expect(toolNameInReadingOrder("receivable_get_aging")).toBe("get_receivable_aging");
    expect(toolNameInReadingOrder("receivable_apply_credit_memo")).toBe(
      "apply_receivable_credit_memo",
    );
  });

  it("leaves a name outside a namespace as it is", () => {
    expect(toolNameInReadingOrder("list_workers")).toBe("list_workers");
    expect(toolNameInReadingOrder("remember")).toBe("remember");
  });
});
