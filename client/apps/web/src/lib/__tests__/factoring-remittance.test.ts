import { describe, expect, it } from "vitest";
import {
  canApplyRemittance,
  canDiscardRemittance,
  canEditRemittanceLines,
  isRemittanceException,
  mappingFromColumns,
  mappingProblems,
  mappingToColumns,
  matchScoreLabel,
  remittanceStep,
} from "../factoring-remittance";

describe("remittance mapping", () => {
  it("round-trips the server's columns and drops fields the file does not have", () => {
    const mapping = mappingFromColumns([
      { field: "Cash", column: 4 },
      { field: "InvoiceNumber", column: 0 },
    ]);
    expect(mapping).toEqual({ Cash: 4, InvoiceNumber: 0 });
    expect(mappingToColumns({ ...mapping, Date: null, Fee: undefined })).toEqual([
      { field: "InvoiceNumber", column: 0 },
      { field: "Cash", column: 4 },
    ]);
  });

  it("refuses what the server refuses: no reference, no amount, a column read twice", () => {
    expect(mappingProblems({})).toEqual([{ kind: "reference" }, { kind: "amount" }]);
    expect(mappingProblems({ FactorReference: 1, Reserve: 2 })).toEqual([]);
    expect(mappingProblems({ InvoiceNumber: 0, Debtor: 0, Advance: 3 })).toEqual([
      { kind: "duplicate", fields: ["InvoiceNumber", "Debtor"] },
    ]);
    expect(mappingProblems({ InvoiceNumber: 0, Fee: 2 })).toEqual([{ kind: "amount" }]);
  });
});

describe("remittance state", () => {
  it("shows the step a remittance's status calls for", () => {
    expect(remittanceStep({ status: "Pending", documentId: null })).toBe("upload");
    expect(remittanceStep({ status: "Pending", documentId: "doc_1" })).toBe("reading");
    expect(remittanceStep({ status: "NeedsMapping" })).toBe("mapping");
    expect(remittanceStep({ status: "Parsed" })).toBe("review");
    expect(remittanceStep({ status: "PartiallyApplied" })).toBe("review");
    expect(remittanceStep({ status: "Applied" })).toBe("done");
    expect(remittanceStep({ status: "Discarded" })).toBe("done");
    expect(remittanceStep({ status: "Failed" })).toBe("failed");
  });

  it("applies only a read remittance with a matched line, and discards none partly booked", () => {
    expect(canApplyRemittance({ status: "Parsed", matchedCount: 1 })).toBe(true);
    expect(canApplyRemittance({ status: "Parsed", matchedCount: 0 })).toBe(false);
    expect(canApplyRemittance({ status: "Applied", matchedCount: 3 })).toBe(false);
    expect(canEditRemittanceLines("PartiallyApplied")).toBe(true);
    expect(canEditRemittanceLines("NeedsMapping")).toBe(false);
    expect(canDiscardRemittance("PartiallyApplied")).toBe(false);
    expect(canDiscardRemittance("Failed")).toBe(true);
  });

  it("counts suggested, unmatched and unreadable lines as exceptions", () => {
    expect(isRemittanceException("Suggested")).toBe(true);
    expect(isRemittanceException("Unmatched")).toBe(true);
    expect(isRemittanceException("Error")).toBe(true);
    expect(isRemittanceException("Matched")).toBe(false);
    expect(isRemittanceException("Ignored")).toBe(false);
    expect(isRemittanceException("Applied")).toBe(false);
  });

  it("reads a score as a percentage within bounds", () => {
    expect(matchScoreLabel(95)).toBe("95%");
    expect(matchScoreLabel(140)).toBe("100%");
    expect(matchScoreLabel(-3)).toBe("0%");
  });
});
