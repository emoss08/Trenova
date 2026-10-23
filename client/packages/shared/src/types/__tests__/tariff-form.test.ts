import { readStoredTariffBill, readStoredTariffChoices } from "@trenova/shared/types/tariff-form";
import { describe, expect, it } from "vitest";

// The two shapes the server sends a bill in: REST stores `tarifftypes.Bill` (Go's JSON tags in
// services/tms/pkg/tarifftypes/tarifftypes.go) and GraphQL reads `TariffBill` (shipment.graphqls).

const stored = {
  currency: "USD",
  lines: [
    {
      kind: "Charge",
      label: "Linehaul",
      freight: true,
      amount: "1908.2",
      quantity: "812",
      quantityUnit: "mi",
      rate: "2.35",
      rateUnit: "$/mi",
    },
    { kind: "Charge", label: "Fuel surcharge", code: "FSC", freight: false, amount: "190.82" },
    { kind: "Rounding", freight: true, amount: "0.01" },
  ],
  notes: [{ text: "Peg $1.20" }],
  freightTotal: "1908.21",
  accessorialTotal: "190.82",
  total: "2099.03",
};

const graphql = {
  currency: "USD",
  lines: [
    {
      kind: "Charge",
      label: "Linehaul",
      code: "",
      quantity: "812",
      quantityUnit: "mi",
      rate: "2.35",
      rateUnit: "$/mi",
      amount: "1908.2",
    },
    {
      kind: "Charge",
      label: "Fuel surcharge",
      code: "FSC",
      quantity: null,
      quantityUnit: "",
      rate: null,
      rateUnit: "",
      amount: "190.82",
    },
    {
      kind: "Rounding",
      label: "",
      code: "",
      quantity: null,
      quantityUnit: "",
      rate: null,
      rateUnit: "",
      amount: "0.01",
    },
  ],
  notes: ["Peg $1.20"],
  freightTotal: "1908.21",
  accessorialTotal: "190.82",
  total: "2099.03",
  exchange: null,
};

const expected = {
  currency: "USD",
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
      label: "Fuel surcharge",
      code: "FSC",
      freight: false,
      amount: { $q: "190.82", u: "$" },
      quantity: null,
      rate: null,
      range: null,
    },
    {
      kind: "rounding",
      label: null,
      code: null,
      freight: true,
      amount: { $q: "0.01", u: "$" },
      quantity: null,
      rate: null,
      range: null,
    },
  ],
  notes: [{ text: "Peg $1.20", range: null }],
  freightTotal: { $q: "1908.21", u: "$" },
  accessorialTotal: { $q: "190.82", u: "$" },
  total: { $q: "2099.03", u: "$" },
  converted: null,
  exchange: null,
};

describe("readStoredTariffBill", () => {
  it("reads a bill as REST stores it", () => {
    expect(readStoredTariffBill(stored)).toEqual(expected);
  });

  it("reads a bill as GraphQL returns it, a line without a code being freight", () => {
    expect(readStoredTariffBill(graphql)).toEqual(expected);
  });

  it("reads a bill it has already read unchanged", () => {
    expect(readStoredTariffBill(readStoredTariffBill(graphql))).toEqual(expected);
  });

  it("reads the exchange a bill was priced through", () => {
    const bill = readStoredTariffBill({ ...graphql, exchange: { currency: "CAD", rate: "1.35" } });
    expect(bill?.exchange).toEqual({ currency: "CAD", rate: "1.35" });
  });

  it("gives null for something that is not a bill", () => {
    expect(readStoredTariffBill({ total: "x" })).toBeNull();
    expect(readStoredTariffBill(null)).toBeNull();
  });
});

describe("readStoredTariffChoices", () => {
  // GraphQL `ShipmentRatingDetail.choices`: a subtotal is a Decimal, a refusal the server's text.
  const stored = [
    {
      label: "Linehaul",
      kind: "best",
      chosen: "Flat",
      options: [
        { label: "Per mile", status: "lost", subtotal: "1175.00", refusal: null },
        { label: "Flat", status: "chosen", subtotal: "900", refusal: null },
        {
          label: "Volume",
          status: "refused",
          subtotal: null,
          refusal: "Volume rates start at 6,000 lb",
        },
      ],
    },
    { label: "Floor", kind: "worst", chosen: null, options: [] },
  ];

  it("reads a stored choice into the module's wire form", () => {
    expect(readStoredTariffChoices(stored)).toEqual([
      {
        label: "Linehaul",
        kind: "best",
        range: null,
        chosen: "Flat",
        options: [
          {
            label: "Per mile",
            range: null,
            status: "lost",
            subtotal: { $q: "1175.00", u: "$" },
            refusal: null,
          },
          {
            label: "Flat",
            range: null,
            status: "chosen",
            subtotal: { $q: "900", u: "$" },
            refusal: null,
          },
          {
            label: "Volume",
            range: null,
            status: "refused",
            subtotal: null,
            refusal: { message: "Volume rates start at 6,000 lb", range: null },
          },
        ],
      },
      { label: "Floor", kind: "worst", range: null, chosen: null, options: [] },
    ]);
  });

  it("reads GraphQL's upper-case enum values", () => {
    const [choice] = readStoredTariffChoices([
      {
        label: "Floor",
        kind: "WORST",
        chosen: "Buy",
        options: [{ label: "Buy", status: "CHOSEN", subtotal: "10", refusal: null }],
      },
    ]);
    expect(choice?.kind).toBe("worst");
    expect(choice?.options[0]?.status).toBe("chosen");
  });

  it("parses a read choice again unchanged", () => {
    const once = readStoredTariffChoices(stored);
    expect(readStoredTariffChoices(once)).toEqual(once);
  });

  it("answers no choices for a rating without any or a value that is not one", () => {
    expect(readStoredTariffChoices(undefined)).toEqual([]);
    expect(readStoredTariffChoices(null)).toEqual([]);
    expect(readStoredTariffChoices("nope")).toEqual([]);
  });
});
