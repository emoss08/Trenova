import { describe, expect, it } from "vitest";
import { nextScope, paletteScopes } from "../palette-scope-tabs";
import { invoiceStatus } from "../palette-status";
import { record, t } from "./palette-test-fixtures";

describe("paletteScopes", () => {
  it("offers only the kinds of record the person may read, in the palette's order", () => {
    expect(paletteScopes(["worker", "invoice", "shipment"])).toEqual([
      "all",
      "shipment",
      "invoice",
      "worker",
      "pages",
      "commands",
    ]);
  });

  it("still offers everything, the pages and the commands to someone who may read no records", () => {
    expect(paletteScopes([])).toEqual(["all", "pages", "commands"]);
  });

  it("offers no kind of record before the person's permissions are known", () => {
    expect(paletteScopes(null)).toEqual(["all", "pages", "commands"]);
  });
});

describe("nextScope", () => {
  it("steps over a scope the person is not offered", () => {
    const scopes = paletteScopes(["shipment"]);

    expect(nextScope("shipment", 1, scopes)).toBe("pages");
    expect(nextScope("all", -1, scopes)).toBe("commands");
  });
});

describe("invoiceStatus", () => {
  const now = 1_800_000_000;
  const DAY = 86_400;
  const invoice = (metadata: Record<string, string>) =>
    record({ entityType: "invoice", id: "inv_1", title: "INV-1042", metadata });

  it("counts the days a posted invoice with money due is past its due date", () => {
    expect(
      invoiceStatus(
        invoice({ status: "Posted", balanceDue: "1250.00", dueDate: String(now - 12 * DAY) }),
        t,
        now,
      ),
    ).toEqual({
      label: "{0, plural, one {# day past due} other {# days past due}}|12",
      state: "warn",
    });
  });

  it("does not call a paid invoice late, however old its due date", () => {
    expect(
      invoiceStatus(
        invoice({
          status: "Posted",
          settlementStatus: "Paid",
          balanceDue: "0",
          dueDate: String(now - 90 * DAY),
        }),
        t,
        now,
      ),
    ).toEqual({ label: "Paid", state: "ok" });
  });

  it("says nothing of an invoice that is open and not yet due", () => {
    expect(
      invoiceStatus(
        invoice({ status: "Posted", balanceDue: "500.00", dueDate: String(now + 5 * DAY) }),
        t,
        now,
      ),
    ).toBeNull();
  });

  it("names a draft and a void", () => {
    expect(invoiceStatus(invoice({ status: "Draft" }), t, now)?.label).toBe("Draft");
    expect(invoiceStatus(invoice({ status: "Voided" }), t, now)?.label).toBe("Voided");
  });
});
