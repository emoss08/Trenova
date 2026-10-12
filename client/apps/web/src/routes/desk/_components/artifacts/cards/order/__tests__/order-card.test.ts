import type { DeskOrder, DeskOrderCharge } from "@/lib/graphql/order-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  chargeLine,
  chargeStats,
  lifecycleSteps,
  orderFacts,
  orderStanding,
  orderWorth,
} from "../order-card";

const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu,
      (_, index, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

function charge(overrides: Partial<DeskOrderCharge> = {}): DeskOrderCharge {
  return {
    id: "och_1",
    description: "Lumper",
    amount: "85.10",
    invoiceId: null,
    invoicedAt: null,
    allocations: [],
    ...overrides,
  } as DeskOrderCharge;
}

describe("lifecycleSteps", () => {
  it("marks the steps behind the order done, its own now and the rest next", () => {
    expect(lifecycleSteps("InProgress").map((step) => [step.key, step.state])).toEqual([
      ["Draft", "done"],
      ["Confirmed", "done"],
      ["InProgress", "now"],
      ["Completed", "next"],
      ["Billed", "next"],
      ["Closed", "next"],
    ]);
    expect(lifecycleSteps("Draft")[0]?.state).toBe("now");
    expect(
      lifecycleSteps("Closed").every((step, index) => step.state === (index < 5 ? "done" : "now")),
    ).toBe(true);
  });

  it("takes a canceled order off the track", () => {
    expect(lifecycleSteps("Canceled").every((step) => step.state === "off")).toBe(true);
  });
});

describe("orderStanding", () => {
  it("reads each status with the tone its place on the way gives it", () => {
    expect(
      (
        ["Draft", "Confirmed", "InProgress", "Completed", "Billed", "Closed", "Canceled"] as const
      ).map((status) => orderStanding(status).tone),
    ).toEqual(["draft", "progress", "progress", "success", "success", "muted", "muted"]);
    expect(orderStanding("InProgress").label).toBe("In progress");
  });
});

describe("orderWorth", () => {
  it("reads the total, else the quote, else says the order is not priced", () => {
    expect(
      orderWorth({ totalAmount: "4200", quotedAmount: "4000", currencyCode: "USD" }, t),
    ).toEqual({
      amount: "$4,200.00",
      caption: "order total",
    });
    expect(orderWorth({ totalAmount: null, quotedAmount: "4000", currencyCode: "USD" }, t)).toEqual(
      {
        amount: "$4,000.00",
        caption: "quoted",
      },
    );
    expect(orderWorth({ totalAmount: null, quotedAmount: null, currencyCode: "USD" }, t)).toEqual({
      amount: null,
      caption: "Not priced yet",
    });
  });
});

describe("charges", () => {
  it("adds the charges in cents, splitting what is invoiced from what is still to invoice", () => {
    const stats = chargeStats(
      [
        charge({ amount: "0.10" }),
        charge({ id: "och_2", amount: "0.20", invoicedAt: 1_791_720_000, invoiceId: "inv_1" }),
        charge({ id: "och_3", amount: "1250" }),
      ],
      "USD",
      t,
    );
    expect(stats.map((stat) => [stat.label, stat.value])).toEqual([
      ["Charges", "$1,250.30"],
      ["Invoiced", "$0.20"],
      ["Still to invoice", "$1,250.10"],
    ]);
  });

  it("reads whether a charge was invoiced, partly or not, and how many ways it is split", () => {
    expect(chargeLine(charge(), t)).toBe("Not yet invoiced");
    expect(chargeLine(charge({ invoiceId: "inv_1" }), t)).toBe("Partly invoiced");
    expect(
      chargeLine(
        charge({
          invoiceId: "inv_1",
          invoicedAt: 1_791_720_000,
          allocations: [{ id: "a" }, { id: "b" }] as DeskOrderCharge["allocations"],
        }),
        t,
      ),
    ).toMatch(/^Invoiced .* · Split 2 ways$/u);
  });
});

describe("orderFacts", () => {
  it("lists the order's numbers and quote, saying what is not on file", () => {
    const facts = Object.fromEntries(
      orderFacts(
        {
          id: "ord_1",
          status: "Draft",
          orderNumber: "ORD-1",
          poNumber: " ",
          bol: null,
          currencyCode: "USD",
          quotedAmount: null,
          baseAmount: null,
          totalAmount: null,
          createdAt: 1_791_720_000,
          updatedAt: 1_791_720_000,
          customer: null,
          charges: [],
        } as DeskOrder,
        t,
      ).map((fact) => [fact.key, fact.value]),
    );
    expect(facts).toMatchObject({
      number: "ORD-1",
      po: "None",
      bol: "None",
      currency: "USD",
      quoted: "Not quoted",
    });
  });
});
