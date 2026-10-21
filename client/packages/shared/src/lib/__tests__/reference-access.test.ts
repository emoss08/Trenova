import { describe, expect, it } from "vitest";
import {
  referenceKindAccessLabel,
  rootField,
  rootFields,
} from "@trenova/shared/lib/reference-access";

describe("rootField", () => {
  it("returns a root record's own value, falsy ones included", () => {
    expect(rootField(0, "PayAdvance.amountMinor")).toBe(0);
    expect(rootField("", "AgentDefinition.systemKey")).toBe("");
    expect(rootField(false, "AgentDefinition.enabled")).toBe(false);
  });

  it("fails loudly on a withheld value, which a root read never answers", () => {
    expect(() => rootField(null, "EscrowAccount.balanceMinor")).toThrow(
      "EscrowAccount.balanceMinor was withheld on a record read as its operation's root",
    );
    expect(() => rootField(undefined, "User.emailAddress")).toThrow(
      "User.emailAddress was withheld",
    );
  });
});

describe("rootFields", () => {
  it("checks only the named fields and keeps the rest of the record as it was", () => {
    const record = { id: "dset_1", netPayMinor: 125_000, currencyCode: "USD", batchId: null };

    expect(rootFields(record, "DriverSettlement", ["netPayMinor", "currencyCode"])).toEqual(
      record,
    );
  });

  it("names the first withheld field", () => {
    expect(() =>
      rootFields({ netPayMinor: 1, currencyCode: null }, "DriverSettlement", [
        "netPayMinor",
        "currencyCode",
      ]),
    ).toThrow("DriverSettlement.currencyCode was withheld");
  });
});

describe("referenceKindAccessLabel", () => {
  it("reads a kind as a plural noun mid-sentence", () => {
    expect(referenceKindAccessLabel("customer")).toBe("customers");
    expect(referenceKindAccessLabel("worker_credential_type")).toBe("credential types");
  });

  it("reads a kind without a label in words", () => {
    expect(referenceKindAccessLabel("agent_memory")).toBe("agent memory");
  });
});
