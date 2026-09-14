import { describe, expect, it } from "vitest";
import type {
  BillingTransferFailureCode,
  BulkBillingTransferResult,
} from "@/lib/graphql/billing-transfer";
import {
  BILLING_TRANSFER_FAILURE_REASONS,
  buildBulkBillingTransferReportCsv,
  mergeBulkBillingTransferRetry,
  retryableShipmentIds,
  summarizeBulkBillingTransfer,
} from "../bulk-billing-transfer-report";

const identity = (message: string | null | undefined, ...args: unknown[]) =>
  args.reduce<string>((text, arg, index) => text.replace(`{${index}}`, String(arg)), message ?? "");

function result(overrides: Partial<BulkBillingTransferResult>): BulkBillingTransferResult {
  return {
    shipmentId: "shp_1",
    proNumber: "PRO-1",
    success: true,
    markedReadyToInvoice: false,
    failureCode: null,
    error: null,
    billingQueueItem: null,
    missingRequirements: [],
    validationFailures: [],
    ...overrides,
  };
}

function failed(shipmentId: string, failureCode: BillingTransferFailureCode) {
  return result({ shipmentId, proNumber: null, success: false, failureCode, error: "no" });
}

describe("bulk billing transfer report", () => {
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
      expect(BILLING_TRANSFER_FAILURE_REASONS[code].label).not.toHaveLength(0);
      expect(BILLING_TRANSFER_FAILURE_REASONS[code].description).not.toHaveLength(0);
    }
  });

  it("counts transfers, the shipments marked ready on the way, failures and unprocessed", () => {
    const summary = summarizeBulkBillingTransfer({
      results: [
        result({ shipmentId: "shp_1" }),
        result({ shipmentId: "shp_2", markedReadyToInvoice: true }),
        failed("shp_3", "RequirementsUnmet"),
        result({
          shipmentId: "shp_4",
          success: false,
          markedReadyToInvoice: true,
          failureCode: "Unexpected",
        }),
      ],
      notProcessedIds: ["shp_5", "shp_6"],
    });

    expect(summary).toEqual({
      transferred: 2,
      markedReadyToInvoice: 1,
      notTransferred: 2,
      notProcessed: 2,
      total: 6,
    });
  });

  it("retries what might pass now, never what is gone or already queued", () => {
    const ids = retryableShipmentIds({
      results: [
        result({ shipmentId: "shp_ok" }),
        failed("shp_docs", "RequirementsUnmet"),
        failed("shp_rate", "RateValidation"),
        failed("shp_ops", "ReturnToOperations"),
        failed("shp_status", "InvalidStatus"),
        failed("shp_boom", "Unexpected"),
        failed("shp_gone", "NotFound"),
        failed("shp_queued", "AlreadyTransferred"),
      ],
      notProcessedIds: ["shp_left"],
    });

    expect(ids).toEqual(["shp_docs", "shp_rate", "shp_ops", "shp_status", "shp_boom", "shp_left"]);
  });

  it("writes one CSV row per shipment with the reason and what blocked it", () => {
    const csv = buildBulkBillingTransferReportCsv(
      {
        results: [
          result({
            shipmentId: "shp_1",
            proNumber: "PRO-1",
            markedReadyToInvoice: true,
            billingQueueItem: { id: "bqi_1", number: "INV-1", status: "Approved" },
          }),
          result({
            shipmentId: "shp_2",
            proNumber: "PRO-2",
            success: false,
            failureCode: "RequirementsUnmet",
            error: "Shipment billing requirements must be resolved, before transfer",
            missingRequirements: [
              {
                documentTypeId: "dt_1",
                documentTypeCode: "POD",
                documentTypeName: "Proof of Delivery",
              },
              {
                documentTypeId: "dt_2",
                documentTypeCode: "BOL",
                documentTypeName: "Bill of Lading",
              },
            ],
            validationFailures: [{ field: "bol", code: "missing_bol", message: "BOL is required" }],
          }),
        ],
        notProcessedIds: ["shp_3"],
      },
      identity,
    );

    const lines = csv.split("\r\n");
    expect(lines).toHaveLength(4);
    expect(lines[0]).toBe(
      "PRO Number,Shipment ID,Outcome,Reason,Details,Missing Documents,Validation Failures,Billing Queue Number,Marked Ready to Invoice",
    );
    expect(lines[1]).toBe("PRO-1,shp_1,Transferred,,,,,INV-1,Yes");
    expect(lines[2]).toBe(
      'PRO-2,shp_2,Not transferred,Missing billing requirements,"Shipment billing requirements must be resolved, before transfer",Proof of Delivery; Bill of Lading,BOL is required,,No',
    );
    expect(lines[3]).toBe(",shp_3,Not processed,,,,,,No");
  });

  it("folds a retry into the report, replacing only the shipments it retried", () => {
    const merged = mergeBulkBillingTransferRetry(
      {
        results: [
          result({ shipmentId: "shp_1" }),
          failed("shp_2", "RequirementsUnmet"),
          failed("shp_3", "AlreadyTransferred"),
          failed("shp_4", "RateValidation"),
        ],
        notProcessedIds: ["shp_5", "shp_6"],
      },
      {
        results: [
          result({ shipmentId: "shp_2", markedReadyToInvoice: true }),
          failed("shp_5", "Unexpected"),
        ],
        notProcessedIds: ["shp_4", "shp_6"],
      },
    );

    expect(merged.results.map((r) => [r.shipmentId, r.success, r.failureCode])).toEqual([
      ["shp_1", true, null],
      ["shp_2", true, null],
      ["shp_3", false, "AlreadyTransferred"],
      ["shp_5", false, "Unexpected"],
    ]);
    expect(merged.notProcessedIds).toEqual(["shp_4", "shp_6"]);
  });

  it("keeps text that a spreadsheet would run as a formula inert", () => {
    const csv = buildBulkBillingTransferReportCsv(
      {
        results: [
          result({
            shipmentId: "shp_1",
            proNumber: "+PRO-1",
            success: false,
            failureCode: "RequirementsUnmet",
            error: "@SUM(1)",
            missingRequirements: [
              {
                documentTypeId: "dt_1",
                documentTypeCode: "X",
                documentTypeName: '=HYPERLINK("http://evil.example","POD")',
              },
            ],
            validationFailures: [{ field: "bol", code: "missing_bol", message: "-2+3" }],
          }),
        ],
        notProcessedIds: [],
      },
      identity,
    );

    const row = csv.split("\r\n")[1];
    expect(row.startsWith("'+PRO-1,")).toBe(true);
    expect(row).toContain(",'@SUM(1),");
    expect(row).toContain(`"'=HYPERLINK(""http://evil.example"",""POD"")"`);
    expect(row).toContain(",'-2+3,");
  });
});
