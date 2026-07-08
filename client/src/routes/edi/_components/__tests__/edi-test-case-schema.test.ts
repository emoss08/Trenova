import { describe, expect, it } from "vitest";
import { ediTestCaseFormSchema, getTestCaseFormDefaults, toTestCaseRequest } from "../edi-schemas";
import { getTestCaseColumns } from "../edi-test-case-columns";

const validPayloadJson = JSON.stringify({
  transactionSet: "210",
  invoice: { invoiceNumber: "INV-1", totalAmount: "100.00" },
});

describe("EDI test case form helpers", () => {
  it("defaults new test cases to an empty scenario", () => {
    expect(getTestCaseFormDefaults()).toMatchObject({
      partnerDocumentProfileId: "",
      name: "",
      description: "",
      payloadJson: "{}",
      expectedWarnings: 0,
      expectedErrors: 0,
      version: 0,
    });
  });

  it("hydrates defaults from an existing test case", () => {
    const defaults = getTestCaseFormDefaults({
      id: "editc_1",
      partnerDocumentProfileId: "edidp_1",
      name: "204 happy path",
      description: "Baseline",
      payload: { transactionSet: "204" },
      expectedWarnings: 1,
      expectedErrors: 0,
      version: 4,
    });

    expect(defaults.partnerDocumentProfileId).toBe("edidp_1");
    expect(defaults.name).toBe("204 happy path");
    expect(defaults.version).toBe(4);
    expect(JSON.parse(defaults.payloadJson)).toMatchObject({ transactionSet: "204" });
  });

  it("rejects invalid payload JSON", () => {
    const result = ediTestCaseFormSchema.safeParse({
      partnerDocumentProfileId: "edidp_1",
      name: "Broken",
      description: "",
      payloadJson: "{not json",
      expectedWarnings: 0,
      expectedErrors: 0,
      version: 0,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.some((issue) => issue.path.includes("payloadJson"))).toBe(true);
    }
  });

  it("rejects payloads without a transaction branch", () => {
    const result = ediTestCaseFormSchema.safeParse({
      partnerDocumentProfileId: "edidp_1",
      name: "Empty payload",
      description: "",
      payloadJson: JSON.stringify({ transactionSet: "204" }),
      expectedWarnings: 0,
      expectedErrors: 0,
      expectedWarningCodes: "",
      expectedErrorCodes: "",
      version: 0,
    });

    expect(result.success).toBe(false);
  });

  it("accepts a payload with an invoice branch", () => {
    const result = ediTestCaseFormSchema.safeParse({
      partnerDocumentProfileId: "edidp_1",
      name: "210 invoice",
      description: "",
      payloadJson: validPayloadJson,
      expectedWarnings: 0,
      expectedErrors: 2,
      expectedWarningCodes: "",
      expectedErrorCodes: "",
      version: 0,
    });

    expect(result.success).toBe(true);
  });

  it("converts form values into a save request", () => {
    const request = toTestCaseRequest({
      partnerDocumentProfileId: "edidp_1",
      name: "210 invoice",
      description: "",
      payloadJson: validPayloadJson,
      expectedWarnings: 1,
      expectedErrors: 2,
      expectedWarningCodes: "value_truncated, missing_optional_element, value_truncated",
      expectedErrorCodes: "",
      version: 5,
    });

    expect(request.expectedWarningCodes).toEqual([
      "missing_optional_element",
      "value_truncated",
    ]);
    expect(request.expectedErrorCodes).toEqual([]);
    expect(request.partnerDocumentProfileId).toBe("edidp_1");
    expect(request.description).toBeUndefined();
    expect(request.expectedWarnings).toBe(1);
    expect(request.expectedErrors).toBe(2);
    expect(request.version).toBe(5);
    expect(request.payload.invoice?.invoiceNumber).toBe("INV-1");
  });
});

describe("EDI test case columns", () => {
  it("defines the certification table columns", () => {
    const columns = getTestCaseColumns();
    const labels = columns.map((column) => column.meta?.label);

    expect(labels).toEqual([
      "Name",
      "Partner",
      "Transaction",
      "Document Profile",
      "Expected Outcome",
      "Updated",
    ]);
    expect(columns[0]?.meta?.filterable).toBe(true);
  });
});
