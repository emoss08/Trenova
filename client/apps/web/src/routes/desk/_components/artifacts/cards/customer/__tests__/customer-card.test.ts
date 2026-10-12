import type { DeskCustomer, DeskCustomerBilling } from "@/lib/graphql/customer-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  accountStanding,
  agingStretches,
  billingRules,
  billingTerms,
  collectionBars,
  creditUse,
  customerAddress,
  invoiceNeeds,
  paymentTermDays,
  payPace,
} from "../customer-card";

/** Fills a message's numbered places the way the catalog would, plurals read as their `other` arm. */
const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(/\{(\d+), plural, one \{[^}]*\} other \{([^}]*)\}\}/gu, (_, index, other: string) =>
      other.replace("#", String(args[Number(index)])),
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

/** Oct 11, 2026 12:00 UTC. */
const NOW = 1_791_720_000;

function customer(overrides: Partial<DeskCustomer> = {}): DeskCustomer {
  return {
    id: "cus_1",
    status: "Active",
    code: "ACME",
    name: "Acme Foods",
    addressLine1: "1200 Market St",
    addressLine2: null,
    city: "Atlanta",
    postalCode: "30303",
    dotNumber: null,
    mcNumber: null,
    createdAt: 1_710_000_000,
    updatedAt: 1_790_000_000,
    state: { abbreviation: "GA" },
    billingProfile: billing(),
    ...overrides,
  } as DeskCustomer;
}

function billing(overrides: Partial<DeskCustomerBilling> = {}): DeskCustomerBilling {
  return {
    paymentTerm: "Net30",
    billingCycle: "Immediate",
    invoiceDelivery: "PerShipment",
    emailInvoiceEnabled: true,
    ediInvoiceEnabled: false,
    autoBill: true,
    creditLimit: "50000",
    creditBalance: "12000",
    creditStatus: "Active",
    enforceCreditLimit: false,
    creditHoldReason: "",
    fuelSurchargeMode: "None",
    billingCurrency: "USD",
    requirePONumber: false,
    requireBOLNumber: false,
    requireDeliveryNumber: false,
    taxExempt: false,
    applyLateCharges: false,
    lateChargeRate: null,
    gracePeriodDays: 0,
    billingNotes: "",
    documentTypes: [],
    ...overrides,
  } as DeskCustomerBilling;
}

describe("accountStanding", () => {
  it("reads an inactive customer as inactive whatever its credit says", () => {
    expect(accountStanding("Inactive", "Hold", t)).toEqual({
      tone: "muted",
      mark: "pause",
      label: "Inactive",
    });
  });

  it("puts a hold or a suspension before a warning or a review", () => {
    expect(accountStanding("Active", "Hold", t)).toMatchObject({ tone: "warn", mark: "pause" });
    expect(accountStanding("Active", "Suspended", t)).toMatchObject({ tone: "muted" });
    expect(accountStanding("Active", "Warning", t)).toMatchObject({ tone: "warn", mark: "dot" });
    expect(accountStanding("Active", "Review", t)).toMatchObject({ tone: "progress" });
  });

  it("reads active credit, or no billing profile at all, as good standing", () => {
    expect(accountStanding("Active", "Active", t).label).toBe("In good standing");
    expect(accountStanding("Active", undefined, t).tone).toBe("success");
  });
});

describe("customerAddress", () => {
  it("writes the address on one line, leaving out what is blank", () => {
    expect(customerAddress(customer())).toBe("1200 Market St, Atlanta, GA 30303");
    expect(customerAddress(customer({ addressLine2: "Suite 400" }))).toBe(
      "1200 Market St, Suite 400, Atlanta, GA 30303",
    );
  });

  it("keeps the city when the state is not known", () => {
    expect(customerAddress(customer({ state: null, addressLine1: " " }))).toBe("Atlanta 30303");
  });
});

describe("paymentTermDays and payPace", () => {
  it("reads the days a term gives", () => {
    expect(paymentTermDays("DueOnReceipt")).toBe(0);
    expect(paymentTermDays("Net30")).toBe(30);
    expect(paymentTermDays("Net90")).toBe(90);
  });

  it("judges paying within terms, a slip, and a habit", () => {
    expect(payPace(28, "Net30")).toBe("ok");
    expect(payPace(30, "Net30")).toBe("ok");
    expect(payPace(45, "Net30")).toBe("warn");
    expect(payPace(46, "Net30")).toBe("bad");
  });

  it("judges nothing without a payment history or a term", () => {
    expect(payPace(0, "Net30")).toBeNull();
    expect(payPace(40, undefined)).toBeNull();
  });
});

describe("agingStretches", () => {
  it("ages what is open into the receivables buckets, each with its share", () => {
    const stretches = agingStretches(
      {
        currentMinor: 600_00,
        days1To30Minor: 200_00,
        days31To60Minor: 0,
        days61To90Minor: 0,
        daysOver90Minor: 200_00,
        totalOpenMinor: 1000_00,
      },
      t,
    );

    expect(stretches.map((stretch) => [stretch.label, stretch.share])).toEqual([
      ["Current", 0.6],
      ["1–30", 0.2],
      ["31–60", 0],
      ["61–90", 0],
      ["90+", 0.2],
    ]);
  });

  it("gives no share to anything when nothing is open, and never a negative one", () => {
    const stretches = agingStretches(
      {
        currentMinor: -50,
        days1To30Minor: 0,
        days31To60Minor: 0,
        days61To90Minor: 0,
        daysOver90Minor: 0,
        totalOpenMinor: 0,
      },
      t,
    );
    expect(stretches.every((stretch) => stretch.share === 0 && stretch.minor === 0)).toBe(true);
  });
});

describe("creditUse", () => {
  it("measures use against the limit, turning as the limit nears", () => {
    expect(creditUse(12_000, 50_000)).toEqual({ share: 0.24, state: "ok" });
    expect(creditUse(37_500, 50_000)?.state).toBe("warn");
    expect(creditUse(45_000, 50_000)?.state).toBe("bad");
    expect(creditUse(60_000, 50_000)).toEqual({ share: 1.2, state: "bad" });
  });

  it("has nothing to measure without a limit", () => {
    expect(creditUse(12_000, null)).toBeNull();
    expect(creditUse(12_000, 0)).toBeNull();
    expect(creditUse(12_000, Number.NaN)).toBeNull();
  });
});

describe("collectionBars", () => {
  const month = (year: number, index: number) => Date.UTC(year, index, 1) / 1000;

  it("draws the last months up to now, a month with no payment as an empty one", () => {
    const bars = collectionBars(
      [
        { monthStart: month(2026, 9), amountMinor: 400_00 },
        { monthStart: month(2026, 6), amountMinor: 800_00 },
      ],
      NOW,
      4,
    );

    expect(bars.map((bar) => bar.monthStart)).toEqual([
      month(2026, 6),
      month(2026, 7),
      month(2026, 8),
      month(2026, 9),
    ]);
    expect(bars.map((bar) => bar.minor)).toEqual([800_00, 0, 0, 400_00]);
    expect(bars.map((bar) => bar.height)).toEqual([1, 0, 0, 0.5]);
  });

  it("names each month in UTC, the zone its start is counted in", () => {
    const [bar] = collectionBars([], month(2026, 0) + 3600, 1);
    expect(bar.label).toBe(
      new Date(Date.UTC(2026, 0, 1)).toLocaleDateString(undefined, {
        month: "short",
        timeZone: "UTC",
      }),
    );
    expect(bar.height).toBe(0);
  });

  it("drops months older than the window and crosses a year", () => {
    const bars = collectionBars(
      [
        { monthStart: month(2025, 11), amountMinor: 100_00 },
        { monthStart: month(2025, 5), amountMinor: 900_00 },
      ],
      month(2026, 1),
      3,
    );
    expect(bars.map((bar) => bar.minor)).toEqual([100_00, 0, 0]);
  });
});

describe("billingTerms, invoiceNeeds and billingRules", () => {
  it("names the terms as the customer page names them, with how invoices go out", () => {
    expect(
      billingTerms(
        billing({
          invoiceDelivery: "Consolidated",
          ediInvoiceEnabled: true,
          billingCycle: "Monthly",
        }),
        t,
      ).map((term) => [term.label, term.value]),
    ).toEqual([
      ["Payment terms", "Net 30"],
      ["Billing cycle", "Monthly"],
      ["Invoices", "Statement (consolidated) · Email and EDI"],
      ["Fuel", "No fuel surcharge"],
    ]);
    expect(
      billingTerms(billing({ emailInvoiceEnabled: false }), t).find(
        (term) => term.key === "delivery",
      )?.value,
    ).toBe("Per shipment · Not sent");
  });

  it("lists the numbers and the paperwork every invoice needs", () => {
    expect(
      invoiceNeeds(
        billing({
          requirePONumber: true,
          requireDeliveryNumber: true,
          documentTypes: [
            { id: "dt_1", name: "Proof of delivery" },
            { id: "dt_2", name: "Lumper receipt" },
          ],
        }),
        t,
      ),
    ).toEqual(["PO number", "Delivery number", "Proof of delivery", "Lumper receipt"]);
    expect(invoiceNeeds(billing({ documentTypes: null }), t)).toEqual([]);
  });

  it("states the rules that change an invoice, and only those that hold", () => {
    expect(billingRules(billing(), t)).toEqual(["Bills automatically"]);
    expect(
      billingRules(
        billing({
          autoBill: false,
          taxExempt: true,
          applyLateCharges: true,
          lateChargeRate: "1.5",
          gracePeriodDays: 10,
          enforceCreditLimit: true,
          billingCurrency: "CAD",
        }),
        t,
      ),
    ).toEqual([
      "Billed by hand",
      "Tax exempt",
      "1.5% late charge after 10 days",
      "Credit limit enforced",
      "Billed in CAD",
    ]);
    expect(
      billingRules(billing({ applyLateCharges: true, lateChargeRate: "0" }), t).some((rule) =>
        rule.includes("late charge"),
      ),
    ).toBe(false);
  });
});
