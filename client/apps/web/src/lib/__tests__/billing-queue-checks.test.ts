import type { BillingCheck, BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { describe, expect, it } from "vitest";
import {
  checkDetail,
  checkTitle,
  isSuggestion,
  openIssuesOf,
  paymentTermLabel,
} from "../billing-queue-checks";

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

  it("finds every open issue a failing check waits on, oldest first", () => {
    const fuel = { id: "bqis_1", checkKey: "charges", summary: "Fuel isn't on the rate con" };
    const stopOff = {
      id: "bqis_2",
      checkKey: "charges",
      summary: "Stop-off isn't on the rate con",
    };
    const kept = { id: "bqis_3", checkKey: "charges", summary: "Lumper", resolutionKey: "keep" };
    const pod = { id: "bqis_4", checkKey: "pod", summary: "POD isn't signed" };
    const flagged = item({
      review: { ...item().review!, issues: [fuel, kept, pod, stopOff] as never },
    });
    const check = {
      key: "charges",
      state: "warn",
      code: "issue",
      issueId: "bqis_1",
    } as BillingCheck;

    expect(openIssuesOf(flagged, check).map((issue) => issue.id)).toEqual(["bqis_1", "bqis_2"]);
  });

  it("leaves a charge the shipment does not bill to the charges, off the check", () => {
    const fuel = {
      id: "bqis_1",
      checkKey: "charges",
      code: "AccessorialNotOnRateCon",
      summary: "Fuel isn't on the rate con",
    };
    const lumper = {
      id: "bqis_2",
      checkKey: "charges",
      code: "UnbilledAccessorial",
      summary: "Lumper $185.00 isn't billed",
    };
    const flagged = item({ review: { ...item().review!, issues: [fuel, lumper] as never } });
    const check = {
      key: "charges",
      state: "warn",
      code: "issue",
      issueId: "bqis_1",
    } as BillingCheck;

    expect(openIssuesOf(flagged, check).map((issue) => issue.id)).toEqual(["bqis_1"]);
    expect(isSuggestion(lumper as never)).toBe(true);
    expect(isSuggestion(fuel as never)).toBe(false);
  });

  it("finds nothing to settle on a check that is clear, though it names what settled it", () => {
    const kept = { id: "bqis_3", checkKey: "charges", summary: "Lumper", resolutionKey: "keep" };
    const settled = item({ review: { ...item().review!, issues: [kept] as never } });
    const check = {
      key: "charges",
      state: "ok",
      code: "resolved",
      issueId: "bqis_3",
    } as BillingCheck;

    expect(openIssuesOf(settled, check)).toEqual([]);
  });

  it("writes payment terms the way people say them", () => {
    expect(paymentTermLabel("Net45", t)).toBe("Net 45");
    expect(paymentTermLabel("DueOnReceipt", t)).toBe("Due on receipt");
  });
});
