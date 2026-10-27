import { describe, expect, it } from "vitest";
import {
  tariffBillSchema,
  tariffCheckResultSchema,
  tariffTestRunSchema,
  tariffTraceEventSchema,
} from "../tariff";

// Fixtures follow the crate's serializers (src/tariff/eval/ops.rs `line_json` and `bill_json`,
// eval/trace.rs, eval/coverage.rs, check/inputs.rs) and the values its testdata records
// (testdata/tariff/eval/{limits,reviews,so-far,coverage}.json, check/time.json).

const money = (amount: string) => ({ $q: amount, u: "$" });

// `charge "Linehaul" (quantity: shipment.distance, rate: rate, minimum: $450)` on 100 mi at
// $2/mi, `charge "Fuel" as FSC = $20`, `minimum $500 on freight`, `minimum $550`, then two
// `review` statements.
const bill = {
  lines: [
    {
      kind: "charge",
      label: "Linehaul",
      code: null,
      freight: true,
      amount: money("450"),
      quantity: { $q: "100", u: "mi" },
      rate: { $q: "2", u: "$/mi" },
      limit: { kind: "minimum", computed: money("200") },
      on: null,
      range: { from: 40, to: 112 },
    },
    {
      kind: "charge",
      label: "Fuel",
      code: "FSC",
      freight: false,
      amount: money("20"),
      quantity: null,
      rate: null,
      limit: null,
      on: null,
      range: { from: 113, to: 140 },
    },
    {
      kind: "minimum",
      label: null,
      code: null,
      freight: true,
      amount: money("50"),
      quantity: null,
      rate: null,
      limit: null,
      on: "freight",
      range: null,
    },
    {
      kind: "minimum",
      label: null,
      code: null,
      freight: true,
      amount: money("30"),
      quantity: null,
      rate: null,
      limit: null,
      on: null,
      range: null,
    },
  ],
  notes: [],
  reviews: [
    { text: "Hazmat over 500 mi", range: { from: 59, to: 99 } },
    { text: "Always check", range: { from: 102, to: 123 } },
  ],
  freightTotal: money("530"),
  accessorialTotal: money("20"),
  total: money("550"),
  converted: null,
};

describe("tariffBillSchema", () => {
  it("reads each line's limit and scope and the bill's reviews", () => {
    const parsed = tariffBillSchema.parse(bill);
    expect(parsed.lines.map((line) => [line.kind, line.limit, line.on])).toEqual([
      ["charge", { kind: "minimum", computed: money("200") }, null],
      ["charge", null, null],
      ["minimum", null, "freight"],
      ["minimum", null, null],
    ]);
    expect(parsed.reviews).toEqual([
      { text: "Hazmat over 500 mi", range: { from: 59, to: 99 } },
      { text: "Always check", range: { from: 102, to: 123 } },
    ]);
  });

  it("reads a bill written before limits, scopes and reviews as having none", () => {
    const { reviews: _reviews, ...older } = bill;
    const parsed = tariffBillSchema.parse({
      ...older,
      lines: older.lines.map(({ limit: _limit, on: _on, ...line }) => line),
    });
    expect(parsed.reviews).toEqual([]);
    expect(parsed.lines.every((line) => line.limit === null && line.on === null)).toBe(true);
  });

  it("refuses a limit of another kind and a scope other than freight", () => {
    const [first, ...rest] = bill.lines;
    expect(
      tariffBillSchema.safeParse({
        ...bill,
        lines: [{ ...first, limit: { kind: "floor", computed: money("200") } }, ...rest],
      }).success,
    ).toBe(false);
    expect(
      tariffBillSchema.safeParse({ ...bill, lines: [{ ...first, on: "accessorials" }, ...rest] })
        .success,
    ).toBe(false);
  });
});

describe("tariffTraceEventSchema", () => {
  it("reads a review, a charge's limit and the subtotal a choice kept", () => {
    const events = [
      {
        kind: "charge",
        label: "Linehaul",
        code: null,
        value: money("450"),
        display: "$450.00",
        limit: { kind: "minimum", computed: money("200") },
        range: { from: 40, to: 112 },
      },
      { kind: "review", text: "Hazmat over 500 mi", range: { from: 59, to: 99 } },
      {
        kind: "choice",
        label: "Linehaul",
        chosen: "Flat",
        subtotal: money("900"),
        range: { from: 70, to: 220 },
      },
      { kind: "choice", label: "Stop", chosen: null, subtotal: null, range: { from: 5, to: 9 } },
    ].map((event) => tariffTraceEventSchema.parse(event));
    expect(events).toEqual([
      {
        kind: "charge",
        label: "Linehaul",
        code: null,
        value: money("450"),
        display: "$450.00",
        limit: { kind: "minimum", computed: money("200") },
        range: { from: 40, to: 112 },
      },
      { kind: "review", text: "Hazmat over 500 mi", range: { from: 59, to: 99 } },
      {
        kind: "choice",
        label: "Linehaul",
        chosen: "Flat",
        subtotal: money("900"),
        range: { from: 70, to: 220 },
      },
      { kind: "choice", label: "Stop", chosen: null, subtotal: null, range: { from: 5, to: 9 } },
    ]);
  });
});

describe("tariffTestRunSchema", () => {
  it("reads review regions and a condition's two outcomes on one range", () => {
    const run = tariffTestRunSchema.parse({
      tests: [
        {
          name: "a cold hazmat load applies",
          range: { from: 79, to: 177 },
          kind: "test",
          status: "passed",
          failures: [],
          bill: null,
          error: null,
          cases: 1,
          skipped: 0,
          counterexample: null,
        },
      ],
      needs: [],
      coverage: {
        regions: [
          { range: { from: 14, to: 78 }, kind: "applies", hits: 1 },
          { range: { from: 14, to: 78 }, kind: "notApplies", hits: 1 },
          { range: { from: 56, to: 60 }, kind: "if", hits: 1 },
          { range: { from: 70, to: 75 }, kind: "else", hits: 0 },
          { range: { from: 102, to: 123 }, kind: "review", hits: 0 },
        ],
        reached: 3,
        total: 5,
      },
    });
    expect(run.coverage?.regions.map((region) => region.kind)).toEqual([
      "applies",
      "notApplies",
      "if",
      "else",
      "review",
    ]);
  });

  it("reads a test's bill with its reviews", () => {
    const run = tariffTestRunSchema.parse({
      tests: [
        {
          name: "hazmat waits for approval",
          range: { from: 0, to: 10 },
          kind: "test",
          status: "passed",
          failures: [],
          bill,
          error: null,
          cases: 1,
          skipped: 0,
          counterexample: null,
        },
      ],
      needs: [],
      coverage: null,
    });
    expect(run.tests[0]?.bill?.reviews.map((review) => review.text)).toEqual([
      "Hazmat over 500 mi",
      "Always check",
    ]);
  });
});

describe("tariffCheckResultSchema", () => {
  it("reads a time input's descriptor, default and constraint", () => {
    const check = tariffCheckResultSchema.parse({
      ok: true,
      header: { kind: "tariff", name: "t" },
      diagnostics: [],
      inputs: [
        {
          constraint: {
            range: { max: { $time: "23:59:00" }, maxInclusive: true, min: { $time: "00:00:00" } },
          },
          default: { $time: "08:00:00" },
          descriptor: { kind: "time" },
          doc: "When the dock opens.",
          name: "open",
          range: { from: 42, to: 46 },
          required: false,
          type: "Time",
        },
      ],
      imports: [],
      outline: {
        items: [
          {
            kind: "total",
            name: "minimum on freight",
            code: null,
            depth: 0,
            range: { from: 141, to: 163 },
            selection: { from: 141, to: 148 },
          },
        ],
        folding: [],
      },
      charges: [],
    });
    expect(check.inputs[0]).toMatchObject({
      descriptor: { kind: "time" },
      default: { $time: "08:00:00" },
      constraint: {
        range: { min: { $time: "00:00:00" }, max: { $time: "23:59:00" }, maxInclusive: true },
      },
    });
  });
});
