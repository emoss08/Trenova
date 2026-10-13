import { describe, expect, it } from "vitest";
import { invoiceBulkEligibility } from "../invoice-bulk";

type Row = Parameters<typeof invoiceBulkEligibility>[0][number];

const row = (id: string, status: Row["status"], sendStatus: Row["sendStatus"]): Row => ({
  id,
  status,
  sendStatus,
});

describe("invoiceBulkEligibility", () => {
  const result = invoiceBulkEligibility([
    row("draft", "Draft", "NotSent"),
    row("posted-unsent", "Posted", "NotSent"),
    row("posted-failed", "Posted", "Failed"),
    row("posted-partial", "Posted", "PartiallySent"),
    row("posted-sent", "Posted", "Sent"),
    row("posted-sending", "Posted", "Sending"),
    row("voided", "Voided", "NotSent"),
  ]);

  it("posts only drafts", () => {
    expect(result.postable.map((r) => r.id)).toEqual(["draft"]);
  });

  it("sends posted invoices that have not gone out, and never a draft", () => {
    expect(result.sendable.map((r) => r.id)).toEqual([
      "posted-unsent",
      "posted-failed",
      "posted-partial",
    ]);
  });
});
