import { describe, expect, it } from "vitest";
import { channelLabel, formatFieldLabel, isSensitiveOmittedValue } from "../audit-log-formatters";

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
