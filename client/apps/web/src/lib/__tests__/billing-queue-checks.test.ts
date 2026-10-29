import type { BillingCheck, BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { describe, expect, it } from "vitest";
import { checkDetail, checkTitle, issueOf, paymentTermLabel } from "../billing-queue-checks";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d)\}/gu, (_m, at) => String(args[Number(at)]))) as never;

function item(extra: Partial<BillingQueueItem> = {}): BillingQueueItem {
  return {
    id: "bqi_1",
    shipmentId: "shp_1",
    status: "InReview",
    review: { checks: [], needsCount: 0, ready: true, blocker: "", issues: [], documents: [] },
    ...extra,
  } as unknown as BillingQueueItem;
}

describe("the five checks", () => {
  it("titles each check as the design does", () => {
    expect(checkTitle("charges", t)).toBe("Charges match the rate con");
    expect(checkTitle("duplicate", t)).toBe("Not a duplicate");
  });

  it("words a passing check from its code and facts", () => {
    const terms = {
      key: "terms",
      state: "ok",
      code: "terms",
      detail: "ap@acme.com · Net30",
      facts: { contact: "ap@acme.com", paymentTerm: "Net30" },
    } as BillingCheck;
    expect(checkDetail(terms, t)).toBe("ap@acme.com · Net 30");

    const unique = {
      key: "duplicate",
      state: "ok",
      code: "unique",
      detail: "",
      facts: { shipment: "S-1001" },
    } as BillingCheck;
    expect(checkDetail(unique, t)).toBe("No other invoice for S-1001");
  });

  it("falls back to the server's own line for a code it does not know", () => {
    const check = { key: "charges", state: "ok", code: "resolved", detail: "Lumper fee kept" };
    expect(checkDetail(check as BillingCheck, t)).toBe("Lumper fee kept");
  });

  it("finds the issue a failing check waits on", () => {
    const issue = { id: "bqis_1", summary: "POD isn't signed", options: [] };
    const withIssue = item({
      review: { ...item().review!, issues: [issue as never] },
    });
    const check = { key: "pod", state: "warn", code: "issue", issueId: "bqis_1" } as BillingCheck;
    expect(issueOf(withIssue, check)?.summary).toBe("POD isn't signed");
    expect(issueOf(withIssue, { ...check, issueId: null })).toBeNull();
  });

  it("writes payment terms the way people say them", () => {
    expect(paymentTermLabel("Net45", t)).toBe("Net 45");
    expect(paymentTermLabel("DueOnReceipt", t)).toBe("Due on receipt");
  });
});
