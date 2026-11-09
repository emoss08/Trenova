import { describe, expect, it } from "vitest";
import {
  channelLabel,
  formatAuditFieldValue,
  formatFieldLabel,
  isSensitiveOmittedValue,
  normalizeAuditChanges,
  recordCurrency,
} from "../audit-log-formatters";

describe("channelLabel", () => {
  it("names each channel the server records", () => {
    expect(channelLabel("agent_conversation")).toBe("Agent, in a conversation");
    expect(channelLabel("scim")).toBe("SCIM directory");
    expect(channelLabel("job")).toBe("Background job");
  });

  it("shows a channel it does not know as the server sent it", () => {
    expect(channelLabel("fax")).toBe("fax");
  });
});

describe("isSensitiveOmittedValue", () => {
  it("marks a value the server left out or fingerprinted", () => {
    expect(isSensitiveOmittedValue("[REDACTED]")).toBe(true);
    expect(isSensitiveOmittedValue("[REDACTED] …6789 #3f9a1c2e")).toBe(true);
    expect(isSensitiveOmittedValue("[REDACTED] …6789")).toBe(true);
    expect(isSensitiveOmittedValue("****")).toBe(true);
  });

  it("leaves a stored value alone, even one that mentions redaction", () => {
    expect(isSensitiveOmittedValue("Assigned")).toBe(false);
    expect(isSensitiveOmittedValue("see [REDACTED] above")).toBe(false);
    expect(isSensitiveOmittedValue("")).toBe(false);
    expect(isSensitiveOmittedValue(4412)).toBe(false);
    expect(isSensitiveOmittedValue(null)).toBe(false);
  });
});

describe("formatFieldLabel", () => {
  it("writes a field path in sentence case", () => {
    expect(formatFieldLabel("creditLimit")).toBe("Credit limit");
    expect(formatFieldLabel("taxId")).toBe("Tax id");
    expect(formatFieldLabel("status")).toBe("Status");
    expect(formatFieldLabel("balanceDueMinor")).toBe("Balance due");
    expect(formatFieldLabel("shipmentMoves.0.weight")).toBe("Shipment moves 0 weight");
    expect(formatFieldLabel("audit_financial_retention_days")).toBe(
      "Audit financial retention days",
    );
  });

  it("keeps an acronym whole", () => {
    expect(formatFieldLabel("SCACCode")).toBe("SCAC code");
    expect(formatFieldLabel("customerEIN")).toBe("Customer EIN");
  });
});

describe("normalizeAuditChanges", () => {
  const typeOf = (change: Record<string, unknown>) =>
    normalizeAuditChanges({ [String(change.path)]: change })[0].type;

  it("reads the server's created and deleted fields as added and removed", () => {
    expect(typeOf({ path: "notes", to: "Call first", type: "created", fieldType: "string" })).toBe(
      "added",
    );
    expect(
      typeOf({ path: "notes", from: "Call first", type: "deleted", fieldType: "string" }),
    ).toBe("removed");
  });

  it("reads an update from null as filled in and to null as cleared", () => {
    expect(typeOf({ path: "postedAt", from: null, to: 1794088846, type: "updated" })).toBe("added");
    expect(typeOf({ path: "postedAt", from: 1794088846, to: null, type: "updated" })).toBe(
      "removed",
    );
  });

  it("keeps an update between two values, falsy ones included, as a change", () => {
    expect(typeOf({ path: "status", from: "Draft", to: "Posted", type: "updated" })).toBe(
      "changed",
    );
    expect(typeOf({ path: "balanceDueMinor", from: 0, to: 258222, type: "updated" })).toBe(
      "changed",
    );
    expect(typeOf({ path: "hazmat", from: false, to: true, type: "updated" })).toBe("changed");
  });
});

describe("formatAuditFieldValue", () => {
  it("writes money in minor units in the record's currency, keeping the stored value", () => {
    expect(formatAuditFieldValue(258_222, "balanceDueMinor", "USD")).toEqual({
      value: "$2,582.22",
      transformed: true,
      hint: "Minor units",
    });
    expect(formatAuditFieldValue(258_222, "balanceDueMinor", "EUR").value).toBe("€2,582.22");
  });

  it("writes an amount without a currency when the record names none", () => {
    expect(formatAuditFieldValue(258_222, "balanceDueMinor").value).toBe("2,582.22");
    expect(formatAuditFieldValue(258_222, "balanceDueMinor", null).value).toBe("2,582.22");
  });

  it("leaves a number in any other field as stored", () => {
    expect(formatAuditFieldValue(258_222, "weight", "USD")).toEqual({
      value: "258222",
      transformed: false,
    });
  });
});

describe("recordCurrency", () => {
  it("reads the currency after the change, then before it", () => {
    expect(recordCurrency({ currencyCode: "CAD" }, { currencyCode: "USD" })).toBe("CAD");
    expect(recordCurrency(null, { currencyCode: "USD" })).toBe("USD");
  });

  it("names none when neither state holds a currency code", () => {
    expect(recordCurrency({ status: "Posted" }, undefined)).toBeNull();
    expect(recordCurrency({ currencyCode: "dollars" }, null)).toBeNull();
    expect(recordCurrency("{}", [])).toBeNull();
  });
});
