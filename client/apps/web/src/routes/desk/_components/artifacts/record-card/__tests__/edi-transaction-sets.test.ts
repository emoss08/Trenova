import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { functionalGroupName, transactionName, transactionSetName } from "../edi-transaction-sets";

const t = translate as TranslateFn;

describe("transactionName", () => {
  it("names every transaction set the way a dispatcher says it", () => {
    expect(
      (["204", "210", "214", "990", "997", "999"] as const).map((set) => transactionName(set, t)),
    ).toEqual([
      "Load tender",
      "Freight invoice",
      "Shipment status",
      "Tender response",
      "Functional acknowledgment",
      "Implementation acknowledgment",
    ]);
  });
});

describe("transactionSetName", () => {
  it("names a set Trenova reads, and numbers one it does not", () => {
    expect([
      transactionSetName("204", t),
      transactionSetName("850", t),
      transactionSetName("", t),
      transactionSetName(null, t),
    ]).toEqual([
      "Load tender",
      "Transaction set 850",
      "Unnamed transaction set",
      "Unnamed transaction set",
    ]);
  });
});

describe("functionalGroupName", () => {
  it("names a group by what its functional identifier code carries", () => {
    expect(
      ["SM", "IM", "QM", "GF", "FA", "PO", "", undefined].map((code) =>
        functionalGroupName(code, t),
      ),
    ).toEqual([
      "Load tenders",
      "Freight invoices",
      "Shipment statuses",
      "Tender responses",
      "Acknowledgments",
      "Group PO",
      "Unnamed group",
      "Unnamed group",
    ]);
  });
});
