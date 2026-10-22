import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { chargeLineBasis, storedChargesBill } from "../tariff-lines";

const t: TranslateFn = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

describe("chargeLineBasis", () => {
  it("writes a freight line's quantity in its unit times its rate", () => {
    expect(
      chargeLineBasis({ quantity: 812, quantityUnit: "mi", rate: 2.35, rateUnit: "$/mi" }, t),
    ).toBe("812 mi × $2.35/mi");
  });

  it("reads the quantity's unit from the rate when the line stores only the rate's", () => {
    expect(chargeLineBasis({ quantity: "12.4", rate: "18.5", rateUnit: "$/cwt" }, t)).toBe(
      "12.4 cwt × $18.50/cwt",
    );
  });

  it("counts stops, pallets and pieces, whose money folds the count in", () => {
    expect(chargeLineBasis({ quantity: 3, rate: 50, rateUnit: "$/stop" }, t)).toBe(
      "3 × $50.00/stop",
    );
  });

  it("writes a count times a per-item amount", () => {
    expect(chargeLineBasis({ quantity: 2, rate: 75, rateUnit: "$" }, t)).toBe("2 × $75.00");
  });

  it("shows the rating currency when given", () => {
    expect(
      chargeLineBasis({ quantity: 3, quantityUnit: "h", rate: 65, rateUnit: "$/h" }, t, {
        currency: "CAD",
      }),
    ).toBe("3 h × CA$65.00/h");
  });

  it("has no basis for a flat amount", () => {
    expect(chargeLineBasis({ quantity: null, rate: null }, t)).toBeNull();
    expect(chargeLineBasis({ quantity: 1, rate: null }, t)).toBeNull();
  });
});

describe("storedChargesBill", () => {
  it("draws freight lines then coded charges with the stored totals", () => {
    const bill = storedChargesBill(
      {
        freightLines: [
          { label: "Linehaul", quantity: 812, rate: 2.35, rateUnit: "$/mi", amount: 1908.2 },
        ],
        charges: [
          {
            label: "Extra drops",
            code: "XDROP",
            quantity: 2,
            rate: 50,
            rateUnit: "$/stop",
            amount: 100,
          },
          { label: "Tarp fee", code: "TARP", quantity: null, rate: null, amount: "25" },
        ],
        freightAmount: 1908.2,
        otherChargeAmount: 125,
        totalChargeAmount: "2033.2",
      },
      "CAD",
    );

    expect(bill).toEqual({
      currency: "CAD",
      lines: [
        {
          kind: "charge",
          label: "Linehaul",
          code: null,
          freight: true,
          amount: { $q: "1908.2", u: "$" },
          quantity: { $q: "812", u: "mi" },
          rate: { $q: "2.35", u: "$/mi" },
          range: null,
        },
        {
          kind: "charge",
          label: "Extra drops",
          code: "XDROP",
          freight: false,
          amount: { $q: "100", u: "$" },
          quantity: { $n: "2" },
          rate: { $q: "50", u: "$/stop" },
          range: null,
        },
        {
          kind: "charge",
          label: "Tarp fee",
          code: "TARP",
          freight: false,
          amount: { $q: "25", u: "$" },
          quantity: null,
          rate: null,
          range: null,
        },
      ],
      notes: [],
      freightTotal: { $q: "1908.2", u: "$" },
      accessorialTotal: { $q: "125", u: "$" },
      total: { $q: "2033.2", u: "$" },
      converted: null,
      exchange: null,
    });
  });

  it("draws a manual rate as its one flat freight line", () => {
    const bill = storedChargesBill(
      {
        freightLines: [{ label: "Freight", amount: "1500" }],
        charges: [],
        freightAmount: "1500",
        otherChargeAmount: "0",
        totalChargeAmount: "1500",
      },
      "USD",
    );

    expect(bill?.lines).toEqual([
      {
        kind: "charge",
        label: "Freight",
        code: null,
        freight: true,
        amount: { $q: "1500", u: "$" },
        quantity: null,
        rate: null,
        range: null,
      },
    ]);
  });
});
