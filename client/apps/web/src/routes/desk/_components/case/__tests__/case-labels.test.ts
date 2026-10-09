import type { CaseChecklist, CaseRecord, CaseSummary } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  caseRailLabel,
  caseRecordPath,
  caseStateLabel,
  checklistItemDetail,
  stepLabel,
  stepPrompt,
} from "../case-labels";

const t = ((source: string, ...args: unknown[]) =>
  source.replace(/\{(\d+)[^}]*\}/gu, (_, index: string) =>
    String(args[Number(index)]),
  )) as TranslateFn;

const NOW = 1_800_000_000;

function record(overrides: Partial<CaseRecord> = {}): CaseRecord {
  return {
    type: "Shipment",
    id: "shp_1",
    label: "10293",
    status: "InTransit",
    closed: false,
    closedAs: "",
    invoiceId: "",
    customerId: "",
    carrierIds: [],
    ...overrides,
  };
}

function summary(overrides: Partial<CaseSummary> = {}): CaseSummary {
  return { state: "Working", openWaits: 0, record: record(), ...overrides };
}

describe("case state", () => {
  it("names who a waiting case waits on", () => {
    expect(
      caseStateLabel(
        summary({ state: "Waiting", waitingOn: "Carrier", openWaits: 1 }),
        NOW,
        "UTC",
        t,
      ),
    ).toBe("Waiting on the carrier");
    expect(caseRailLabel(summary({ state: "Waiting", waitingOn: "Customer" }), NOW, "UTC", t)).toBe(
      "Customer",
    );
  });

  it("says how a settled case closed", () => {
    const settled = summary({
      state: "Settled",
      record: record({ closed: true, closedAs: "Paid" }),
    });
    expect(caseStateLabel(settled, NOW, "UTC", t)).toBe("Settled · Paid");
  });

  it("reads a snooze that has run out as the state under it", () => {
    const lapsed = summary({ state: "Snoozed", snoozedUntil: NOW - 1, openWaits: 1 });
    expect(caseStateLabel(lapsed, NOW, "UTC", t)).toBe("Waiting");
    expect(caseRailLabel(summary({ state: "Snoozed", snoozedUntil: NOW - 1 }), NOW, "UTC", t)).toBe(
      "",
    );
  });
});

describe("caseRecordPath", () => {
  it("opens a dispute on its invoice's disputes tab", () => {
    expect(
      caseRecordPath(record({ type: "InvoiceDispute", id: "idsp_1", invoiceId: "inv_9" })),
    ).toContain("inv_9");
    expect(caseRecordPath(record({ type: "InvoiceDispute", id: "idsp_1" }))).toBeNull();
    expect(caseRecordPath(record({ status: "Missing" }))).toBeNull();
  });
});

describe("stepPrompt", () => {
  const checklist: CaseChecklist = {
    kind: "ReadyToBill",
    ready: false,
    next: "request_paperwork",
    items: [
      {
        key: "paperwork",
        state: "Blocked",
        codes: [],
        names: ["Bill of lading", "Lumper receipt"],
        count: 2,
        step: "request_paperwork",
        optional: false,
        label: "",
        stepLabel: "",
        prompt: "",
        manual: false,
        tickedBy: "",
      },
    ],
  };

  it("names the record and what the checklist knows", () => {
    expect(stepPrompt("request_paperwork", record(), checklist, t)).toBe(
      "Request the missing paperwork for shipment 10293: Bill of lading, Lumper receipt.",
    );
    expect(stepPrompt("send_invoice", record({ type: "Invoice", label: "INV-8" }), null, t)).toBe(
      "Send invoice INV-8 to the customer.",
    );
  });
});

describe("checklistItemDetail", () => {
  it("words a blocked item from its codes and names", () => {
    expect(
      checklistItemDetail(
        {
          key: "billingHolds",
          state: "Blocked",
          codes: ["credit_hold", "something_new"],
          names: [],
          count: 2,
          step: "clear_holds",
          optional: false,
          label: "",
          stepLabel: "",
          prompt: "",
          manual: false,
          tickedBy: "",
        },
        "UTC",
        t,
      ),
    ).toBe("Customer on credit hold · something_new");
  });
});

describe("a step the organization added", () => {
  const checklist: CaseChecklist = {
    kind: "ReadyToBill",
    ready: false,
    next: "custom:callshipper",
    items: [
      {
        key: "custom:callshipper",
        state: "Blocked",
        codes: [],
        names: [],
        count: 0,
        step: "custom:callshipper",
        optional: false,
        label: "Shipper called",
        stepLabel: "Call the shipper",
        prompt: "Draft what I should say to the shipper.",
        manual: true,
        tickedBy: "",
      },
    ],
  };

  it("names its button and asks what the organization wrote, with the record named", () => {
    expect(stepLabel("custom:callshipper", t, checklist)).toBe("Call the shipper");
    expect(stepPrompt("custom:callshipper", record(), checklist, t)).toBe(
      "For shipment 10293: Draft what I should say to the shipper.",
    );
  });

  it("falls back to its name when the organization wrote no button or request", () => {
    const bare: CaseChecklist = {
      ...checklist,
      items: [{ ...checklist.items[0], stepLabel: "", prompt: "" }],
    };
    expect(stepLabel("custom:callshipper", t, bare)).toBe("Shipper called");
    expect(stepPrompt("custom:callshipper", record(), bare, t)).toBe(
      "Take care of “Shipper called” for shipment 10293.",
    );
  });

  it("says who ticked it", () => {
    expect(
      checklistItemDetail(
        { ...checklist.items[0], state: "Done", tickedBy: "Dana", at: null },
        "UTC",
        t,
      ),
    ).toBe("Ticked by Dana");
  });
});
