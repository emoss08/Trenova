import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { ShipmentBillingReadiness } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import { approveBlocker, billingChecks } from "../billing-item-checks";

const t = ((text: string, ...args: unknown[]) =>
  text
    .replace(/\{(\d)\}/gu, (_m, at) => String(args[Number(at)]))
    .replace(/\{0, plural[^}]*\{[^}]*\}[^}]*\{[^}]*\}\}/u, String(args[0]))) as never;

function item(extra: Partial<BillingQueueItem> = {}): BillingQueueItem {
  return {
    id: "bqi_1",
    shipmentId: "shp_1",
    status: "InReview",
    billType: "Invoice",
    billToCustomerId: "cus_1",
    assignedBillerId: "usr_1",
    assignedBiller: { name: "Avery Lane" },
    billToCustomer: { id: "cus_1", name: "Acme Manufacturing" },
    detentionHolds: [],
    payerShare: null,
    createdAt: 0,
    ...extra,
  } as unknown as BillingQueueItem;
}

function readiness(extra: Partial<ShipmentBillingReadiness> = {}): ShipmentBillingReadiness {
  return {
    requirements: [{ documentTypeName: "POD", satisfied: true }],
    missingRequirements: [],
    validationFailures: [],
    warnings: [],
    serviceFailureContext: { hasUnresolved: false, unresolvedCount: 0, serviceFailureIds: [] },
    payers: [],
    canMarkReadyToInvoice: true,
    ...extra,
  } as unknown as ShipmentBillingReadiness;
}

describe("billing checks", () => {
  it("asks for a biller when nobody is assigned, and blocks approval on it", () => {
    const unassigned = item({ assignedBillerId: null, assignedBiller: null });
    const checks = billingChecks(unassigned, readiness(), t);
    expect(checks[0]).toMatchObject({ key: "biller", state: "fail", action: "assign" });
    expect(approveBlocker(unassigned, checks, readiness(), t)).toBe("Assign a biller first");
  });

  it("names missing documents and detention waiting on approval", () => {
    const held = item({
      detentionHolds: [
        { occurrenceId: "o1", locationName: "Acme DC 2", billableAmount: "130", currency: "USD" },
      ] as never,
    });
    const checks = billingChecks(
      held,
      readiness({ missingRequirements: [{ documentTypeName: "POD" }] as never }),
      t,
    );
    expect(checks.find((check) => check.key === "documents")).toMatchObject({
      state: "fail",
      detail: "Missing POD",
    });
    expect(checks.find((check) => check.key === "hold-o1")?.state).toBe("fail");
  });

  it("is clear to approve when every check passes and the review has started", () => {
    const ready = item();
    const checks = billingChecks(ready, readiness(), t);
    expect(checks.every((check) => check.state === "ok")).toBe(true);
    expect(approveBlocker(ready, checks, readiness(), t)).toBe("");
    expect(approveBlocker(item({ status: "ReadyForReview" }), checks, readiness(), t)).toBe(
      "Start the review first",
    );
  });
});
