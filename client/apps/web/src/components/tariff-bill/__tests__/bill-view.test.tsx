import { tariffBillSchema, type TariffBill } from "@trenova/shared/types/tariff";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BillView } from "../bill-view";

// The bill tariff.evaluate writes for the reference's "A first look" tariff on an 812-mile,
// two-drop load, with a minimum and a rounding remainder, in the module's wire shape.
const bill: TariffBill = tariffBillSchema.parse({
  lines: [
    {
      kind: "charge",
      label: "Linehaul",
      code: null,
      freight: true,
      amount: { $q: "1908.2", u: "$" },
      quantity: { $q: "812", u: "mi" },
      rate: { $q: "2.35", u: "$/mi" },
      range: { from: 410, to: 456 },
    },
    {
      kind: "charge",
      label: "Fuel surcharge",
      code: "FSC",
      freight: false,
      amount: { $q: "343.48", u: "$" },
      quantity: null,
      rate: null,
      range: { from: 520, to: 640 },
    },
    {
      kind: "minimum",
      label: null,
      code: null,
      freight: true,
      amount: { $q: "48.32", u: "$" },
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
  notes: [
    { text: "No detention: every stop left within its free time", range: { from: 700, to: 760 } },
  ],
  freightTotal: { $q: "1956.53", u: "$" },
  accessorialTotal: { $q: "343.48", u: "$" },
  total: { $q: "2300.01", u: "$" },
});

describe("BillView", () => {
  it("shows each charge as written, with its code and quantity × rate, and labels adjustments", () => {
    render(<BillView bill={bill} />);
    const rows = screen.getAllByRole("listitem").filter((row) => row.dataset.kind);
    expect(rows.map((row) => row.dataset.kind)).toEqual([
      "charge",
      "charge",
      "minimum",
      "rounding",
    ]);
    expect(within(rows[0]!).getByText("Linehaul")).toBeInTheDocument();
    expect(within(rows[0]!).getByText("812 mi × $2.35/mi")).toBeInTheDocument();
    expect(within(rows[0]!).getByText("$1,908.20")).toBeInTheDocument();
    expect(within(rows[1]!).getByText("FSC")).toBeInTheDocument();
    expect(within(rows[2]!).getByText("Minimum charge adjustment")).toBeInTheDocument();
    expect(within(rows[3]!).getByText("Rounding")).toBeInTheDocument();
    expect(
      screen.getByText("No detention: every stop left within its free time"),
    ).toBeInTheDocument();
  });

  it("totals freight, accessorials and the bill", () => {
    render(<BillView bill={bill} />);
    expect(screen.getByText("Freight").nextSibling).toHaveTextContent("$1,956.53");
    expect(screen.getByText("Accessorials").nextSibling).toHaveTextContent("$343.48");
    expect(screen.getByText("Total").nextSibling).toHaveTextContent("$2,300.01");
  });

  it("leaves out the subtotals when asked for the total only", () => {
    render(<BillView bill={bill} totalOnly />);
    expect(screen.queryByText("Freight")).toBeNull();
    expect(screen.getByText("Total")).toBeInTheDocument();
  });

  it("shows the converted totals in the billing currency", () => {
    const converted = tariffBillSchema.parse({
      ...bill,
      converted: {
        currency: "CAD",
        rate: "1.3502",
        lines: [
          { $q: "2576.45", u: "$" },
          { $q: "463.77", u: "$" },
          { $q: "65.24", u: "$" },
          { $q: "0.01", u: "$" },
        ],
        total: { $q: "3105.47", u: "$" },
      },
    });
    render(<BillView bill={converted} />);
    expect(screen.getByText("CA$2,576.45")).toBeInTheDocument();
    expect(screen.getByText("Total in CAD").nextSibling).toHaveTextContent("CA$3,105.47");
    expect(screen.getByText("Exchange rate").nextSibling).toHaveTextContent("1.3502");
  });

  it("reveals a line's source when a line with a range is picked", () => {
    const onSelectRange = vi.fn();
    render(<BillView bill={bill} onSelectRange={onSelectRange} />);
    fireEvent.click(screen.getByRole("button", { name: /Linehaul/ }));
    expect(onSelectRange).toHaveBeenCalledWith({ from: 410, to: 456 });
    expect(screen.queryByRole("button", { name: /Rounding/ })).toBeNull();
  });

  it("lists the reviews the tariff asks for apart from its notes", () => {
    const reviewed = tariffBillSchema.parse({
      ...bill,
      reviews: [
        { text: "Holiday pickup", range: { from: 800, to: 840 } },
        { text: "Liftgate billed on a dock delivery", range: null },
      ],
    });
    const onSelectRange = vi.fn();
    render(<BillView bill={reviewed} onSelectRange={onSelectRange} />);

    const reviews = screen.getByRole("list", { name: "Reviews" });
    expect(
      within(reviews)
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual(["Holiday pickup", "Liftgate billed on a dock delivery"]);
    const notes = screen.getByRole("list", { name: "Notes" });
    expect(notes).not.toHaveTextContent("Holiday pickup");

    fireEvent.click(within(reviews).getByRole("button", { name: "Holiday pickup" }));
    expect(onSelectRange).toHaveBeenCalledWith({ from: 800, to: 840 });
  });

  it("shows no reviews list when the tariff asks for none", () => {
    render(<BillView bill={bill} />);
    expect(screen.queryByRole("list", { name: "Reviews" })).toBeNull();
    expect(bill.reviews).toEqual([]);
  });

  it("says when a rating billed nothing", () => {
    render(<BillView bill={{ ...bill, lines: [], notes: [] }} />);
    expect(screen.getByText("This rating billed no charges.")).toBeInTheDocument();
  });

  it("puts the reviews under a Needs review heading, apart from the notes", () => {
    const reviewed = tariffBillSchema.parse({
      ...bill,
      reviews: [{ text: "Holiday pickup", range: null }],
    });
    render(<BillView bill={reviewed} />);
    const section = screen.getByRole("region", { name: "Needs review" });
    expect(within(section).getByText("Needs review")).toBeInTheDocument();
    expect(within(section).getByRole("list", { name: "Reviews" })).toHaveTextContent(
      "Holiday pickup",
    );
    expect(screen.getByRole("list", { name: "Notes" })).not.toHaveTextContent("Holiday pickup");
  });

  it("draws no Needs review heading when the tariff asks for no review", () => {
    render(<BillView bill={bill} />);
    expect(screen.queryByText("Needs review")).toBeNull();
  });
});

// The lines tariff.evaluate writes for testdata/tariff/eval/limits.json's cases: a charge its
// own minimum raised (100 mi at $2/mi came to $200, billed $450), one its maximum lowered, one
// no limit touched, and the adjustments of `minimum $400 on freight` and of a plain `minimum`.
const limited: TariffBill = tariffBillSchema.parse({
  lines: [
    {
      kind: "charge",
      label: "Linehaul",
      code: null,
      freight: true,
      amount: { $q: "450", u: "$" },
      quantity: { $q: "100", u: "mi" },
      rate: { $q: "2", u: "$/mi" },
      limit: { kind: "minimum", computed: { $q: "200", u: "$" } },
      on: null,
      range: { from: 40, to: 112 },
    },
    {
      kind: "charge",
      label: "Stops",
      code: "STOP",
      freight: false,
      amount: { $q: "1500", u: "$" },
      quantity: { $q: "10", u: "stop" },
      rate: { $q: "200", u: "$/stop" },
      limit: { kind: "maximum", computed: { $q: "2000", u: "$" } },
      on: null,
      range: { from: 113, to: 190 },
    },
    {
      kind: "charge",
      label: "Fuel",
      code: "FSC",
      freight: false,
      amount: { $q: "45", u: "$" },
      quantity: null,
      rate: null,
      limit: null,
      on: null,
      range: { from: 191, to: 230 },
    },
    {
      kind: "minimum",
      label: null,
      code: null,
      freight: true,
      amount: { $q: "100", u: "$" },
      quantity: null,
      rate: null,
      limit: null,
      on: "freight",
      range: null,
    },
    {
      kind: "maximum",
      label: null,
      code: null,
      freight: true,
      amount: { $q: "-95", u: "$" },
      quantity: null,
      rate: null,
      limit: null,
      on: null,
      range: null,
    },
  ],
  notes: [],
  reviews: [],
  freightTotal: { $q: "455", u: "$" },
  accessorialTotal: { $q: "1545", u: "$" },
  total: { $q: "2000", u: "$" },
});

describe("BillView with line limits", () => {
  it("says which limit set a line's amount and what the line came to before it", () => {
    render(<BillView bill={limited} />);
    const rows = screen.getAllByRole("listitem").filter((row) => row.dataset.kind);
    expect(within(rows[0]!).getByText("100 mi × $2.00/mi")).toBeInTheDocument();
    expect(within(rows[0]!).getByText("$450.00")).toBeInTheDocument();
    expect(within(rows[0]!).getByText("Minimum applied · computed $200.00")).toBeInTheDocument();
    expect(within(rows[1]!).getByText("Maximum applied · computed $2,000.00")).toBeInTheDocument();
    expect(within(rows[2]!).queryByText(/applied/)).toBeNull();
  });

  it("labels a freight-only adjustment apart from the whole bill's", () => {
    render(<BillView bill={limited} />);
    const rows = screen.getAllByRole("listitem").filter((row) => row.dataset.kind);
    expect(within(rows[3]!).getByText("Minimum charge adjustment (freight)")).toBeInTheDocument();
    expect(within(rows[4]!).getByText("Maximum charge adjustment")).toBeInTheDocument();
  });
});
