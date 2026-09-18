import { describe, expect, it } from "vitest";
import type {
  BillingTransferFailureCode,
  BillingTransferRunItem,
} from "@/lib/graphql/billing-transfer";
import {
  BILLING_TRANSFER_FAILURE_REASONS,
  buildBulkBillingTransferReportCsv,
} from "../bulk-billing-transfer-report";

const identity = (message: string | null | undefined, ...args: unknown[]) =>
  args.reduce<string>((text, arg, index) => text.replace(`{${index}}`, String(arg)), message ?? "");

function item(overrides: Partial<BillingTransferRunItem> = {}): BillingTransferRunItem {
  return {
    id: "btri_1",
    shipmentId: "shp_1",
    sequence: 0,
    proNumber: "PRO-1",
    status: "Transferred",
    failureCode: null,
    errorMessage: null,
    markedReadyToInvoice: false,
    billingQueueNumber: null,
    billingQueueStatus: null,
    missingRequirements: [],
    validationFailures: [],
    ...overrides,
  };
}

describe("BILLING_TRANSFER_FAILURE_REASONS", () => {
  it("has a reason for every failure code the server can send", () => {
    const codes: BillingTransferFailureCode[] = [
      "NotFound",
      "InvalidStatus",
      "AlreadyTransferred",
      "RequirementsUnmet",
      "RateValidation",
      "ReturnToOperations",
      "Unexpected",
    ];

    for (const code of codes) {
      expect(BILLING_TRANSFER_FAILURE_REASONS[code]).toBeDefined();
      expect(BILLING_TRANSFER_FAILURE_REASONS[code].label).not.toBe("");
      expect(BILLING_TRANSFER_FAILURE_REASONS[code].description).not.toBe("");
    }
  });
});

describe("buildBulkBillingTransferReportCsv", () => {
  it("writes one row per shipment with the reason and what blocked it", () => {
    const csv = buildBulkBillingTransferReportCsv(
      [
        item({
          shipmentId: "shp_ok",
          proNumber: "PRO-OK",
          status: "Transferred",
          markedReadyToInvoice: true,
          billingQueueNumber: "INV-1",
        }),
        item({
          id: "btri_2",
          shipmentId: "shp_bad",
          proNumber: "PRO-BAD",
          status: "NotTransferred",
          failureCode: "RequirementsUnmet",
          errorMessage: "Billing requirements must be resolved first",
          missingRequirements: [
            {
              documentTypeId: "dt_pod",
              documentTypeCode: "POD",
              documentTypeName: "Proof of Delivery",
            },
          ],
          validationFailures: [
            { field: "rate", code: "missing_basis", message: "The rate has no basis" },
          ],
        }),
        item({
          id: "btri_3",
          shipmentId: "shp_skipped",
          proNumber: "PRO-SKIP",
          status: "Skipped",
        }),
      ],
      identity,
    );

    const lines = csv.split("\r\n");
    expect(lines).toHaveLength(4);
    expect(lines[0]).toContain("PRO Number");

    expect(lines[1]).toContain("PRO-OK");
    expect(lines[1]).toContain("Transferred");
    expect(lines[1]).toContain("INV-1");
    expect(lines[1]).toContain("Yes");

    expect(lines[2]).toContain("PRO-BAD");
    expect(lines[2]).toContain("Not transferred");
    expect(lines[2]).toContain(BILLING_TRANSFER_FAILURE_REASONS.RequirementsUnmet.label);
    expect(lines[2]).toContain("Proof of Delivery");
    expect(lines[2]).toContain("The rate has no basis");

    // A shipment the run never reached is still in the report: that is the work
    // the biller has left, and inferring it from an absence would be worse.
    expect(lines[3]).toContain("PRO-SKIP");
    expect(lines[3]).toContain("Not processed");
  });

  it("keeps text that a spreadsheet would run as a formula inert", () => {
    const csv = buildBulkBillingTransferReportCsv(
      [
        item({
          proNumber: "=1+1",
          status: "NotTransferred",
          failureCode: "Unexpected",
          errorMessage: "@SUM(A1:A2)",
        }),
      ],
      identity,
    );

    const row = csv.split("\r\n")[1];
    expect(row).not.toMatch(/(^|,)"?=1\+1/);
    expect(row).not.toMatch(/(^|,)"?@SUM/);
  });

  it("writes only a header when the run answered for nothing", () => {
    const csv = buildBulkBillingTransferReportCsv([], identity);

    expect(csv.split("\r\n")).toHaveLength(1);
  });
});
