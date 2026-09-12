import { describe, expect, it } from "vitest";
import { describeBillingSchedule, type BillingSchedule } from "@/lib/billing-schedule";

function schedule(overrides: Partial<BillingSchedule> = {}): BillingSchedule {
  return {
    invoiceDelivery: "Consolidated",
    billingCycle: "Monthly",
    billingCycleAnchorDay: 1,
    splitBy: "Customer",
    sectionBy: "Shipment",
    invoiceDetail: "Detailed",
    maxShipmentsPerInvoice: 0,
    ...overrides,
  };
}

describe("describeBillingSchedule", () => {
  it("describes per-shipment billing without mentioning a cycle", () => {
    const sentence = describeBillingSchedule(
      schedule({ invoiceDelivery: "PerShipment", billingCycle: "Immediate" }),
    );

    expect(sentence).toBe(
      "Bills each shipment on its own invoice as soon as it is approved in the billing queue.",
    );
    expect(sentence).not.toContain("period");
  });

  it("describes per-order billing", () => {
    expect(describeBillingSchedule(schedule({ invoiceDelivery: "PerOrder" }))).toContain(
      "every billable leg",
    );
  });

  it("names the cadence and the split key for a statement customer", () => {
    const sentence = describeBillingSchedule(
      schedule({
        billingCycle: "Monthly",
        billingCycleAnchorDay: 1,
        splitBy: "CustomerAndPONumber",
      }),
    );

    expect(sentence).toContain("Bills every month on the 1st.");
    expect(sentence).toContain("one invoice per PO number");
  });

  it("says a single invoice when nothing splits the period", () => {
    expect(describeBillingSchedule(schedule({ splitBy: "Customer" }))).toContain(
      "combined into a single invoice",
    );
  });

  it("reports the section key and the detail level separately from the split", () => {
    const sentence = describeBillingSchedule(
      schedule({
        splitBy: "CustomerAndPONumber",
        sectionBy: "Destination",
        invoiceDetail: "Summary",
      }),
    );

    expect(sentence).toContain("one invoice per PO number");
    expect(sentence).toContain("grouped by delivery location");
    expect(sentence).toContain("one line per shipment");
  });

  it("names the weekday for weekly cycles", () => {
    expect(
      describeBillingSchedule(schedule({ billingCycle: "Weekly", billingCycleAnchorDay: 0 })),
    ).toContain("every week on Sunday");
    expect(
      describeBillingSchedule(schedule({ billingCycle: "BiWeekly", billingCycleAnchorDay: 5 })),
    ).toContain("every other week on Friday");
  });

  it("renders ordinals correctly for the awkward days", () => {
    for (const [day, expected] of [
      [1, "1st"],
      [2, "2nd"],
      [3, "3rd"],
      [4, "4th"],
      [11, "11th"],
      [12, "12th"],
      [13, "13th"],
      [21, "21st"],
      [22, "22nd"],
      [23, "23rd"],
    ] as const) {
      expect(
        describeBillingSchedule(schedule({ billingCycle: "Monthly", billingCycleAnchorDay: day })),
      ).toContain(`on the ${expected}`);
    }
  });

  it("mentions the shipment cap only when one is set", () => {
    expect(describeBillingSchedule(schedule({ maxShipmentsPerInvoice: 0 }))).not.toContain(
      "per invoice.",
    );
    expect(describeBillingSchedule(schedule({ maxShipmentsPerInvoice: 50 }))).toContain(
      "Up to 50 shipments per invoice.",
    );
  });

  it("covers every cadence without falling through to the immediate wording", () => {
    for (const cycle of [
      "Daily",
      "Weekly",
      "BiWeekly",
      "SemiMonthly",
      "Monthly",
      "Quarterly",
    ] as const) {
      expect(describeBillingSchedule(schedule({ billingCycle: cycle }))).not.toContain(
        "as soon as a shipment is approved",
      );
    }
  });
});
