import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffTest } from "@trenova/shared/rules/tariff";
import {
  EMPTY_TARIFF_CONTEXT,
  tariffTestRunSchema,
  type TariffTestRun,
} from "@trenova/shared/types/tariff";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TestsPanel } from "../tests-panel";
import type { TariffTestsState } from "../use-tariff-tests";

afterEach(cleanup);

// The reference's weight-break inversion (formula-language.md, "Properties"): 999 lb bills at the
// light rate and 1,000 lb at the heavy one, so heavier bills less; a test reaches only the light
// branch, so the heavy branch is a coverage gap.
const SOURCE = `tariff "LTL"
input rate: Money/lb = $0.31/lb
let weight = shipment.weight ?? 0 lb
if weight >= 1_000 lb {
  charge "Freight" = weight * $0.25/lb
} else {
  charge "Freight" = weight * rate
}
if shipment.distance > 5_000 mi {
  charge "Long haul" = $100
}

property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  expect total nondecreasing in shipment.weight
}

property "light loads bill something" {
  vary shipment.weight in [5 lb, 50 lb, 500 lb]
  assume shipment.weight != 5 lb
  expect total > $0
}

test "a light load" {
  given shipment.weight = 500 lb
  expect charge("Freight") == $155
}
`;

function ran(run: TariffTestRun): TariffTestsState {
  const passed = run.tests.filter((test) => test.status === "passed").length;
  return {
    status: "ran",
    run,
    passed,
    failed: run.tests.length - passed,
    stale: false,
    cells: "ready",
    retryCells: () => {},
  };
}

const run = () => tariffTest(rulesSync(), { source: SOURCE, context: EMPTY_TARIFF_CONTEXT });

describe("TestsPanel with properties", () => {
  it("lists properties beside tests with their cases and what assume skipped", () => {
    render(<TestsPanel state={ran(run())} source={SOURCE} onReveal={vi.fn()} />);
    expect(screen.getByText("2 of 3 passing")).toBeInTheDocument();
    expect(screen.getAllByText("Property")).toHaveLength(2);
    const light = screen.getByText("light loads bill something").closest("li")!;
    expect(within(light).getByText("2 cases, 1 skipped")).toBeInTheDocument();
    const heavier = screen.getByText("heavier never bills less").closest("li")!;
    expect(within(heavier).getByText("24 cases")).toBeInTheDocument();
  });

  it("shows a monotone failure as the out-of-order pair with their results", () => {
    render(<TestsPanel state={ran(run())} source={SOURCE} onReveal={vi.fn()} />);
    const counterexample = screen.getByRole("region", { name: "Counterexample" });
    expect(
      within(within(counterexample).getByRole("list", { name: "Varied values" })).getByRole(
        "listitem",
      ),
    ).toHaveTextContent("shipment.weight = 999 lb");
    expect(
      within(counterexample).getByText(
        (_, node) =>
          node?.textContent === "total falls as shipment.weight grows:" && node.tagName === "SPAN",
      ),
    ).toBeInTheDocument();
    expect(
      within(counterexample).getByText("999 lb → $309.69, 1,000 lb → $250.00"),
    ).toBeInTheDocument();
  });

  it("saves the counterexample as a test with the module's block", async () => {
    const user = userEvent.setup();
    const save = vi.fn();
    const result = run();
    render(
      <TestsPanel state={ran(result)} source={SOURCE} onReveal={vi.fn()} onSaveAsTest={save} />,
    );
    await user.click(screen.getByRole("button", { name: "Save as test" }));
    expect(save).toHaveBeenCalledWith(result.tests[0]?.counterexample?.test);
    expect(save.mock.calls[0]?.[0]).toMatch(/^test "heavier never bills less: 999 lb" \{/);
  });

  it("offers no Save as test where the page cannot write the source", () => {
    render(<TestsPanel state={ran(run())} source={SOURCE} onReveal={vi.fn()} />);
    expect(screen.queryByRole("button", { name: "Save as test" })).not.toBeInTheDocument();
  });

  it("shows a refusing case's reason", () => {
    const result = run();
    const property = result.tests[0]!;
    const refused: TariffTestRun = {
      ...result,
      tests: [
        {
          ...property,
          counterexample: {
            ...property.counterexample!,
            monotone: null,
            refusal: { message: "No rate for this weight", range: null },
          },
        },
      ],
    };
    render(<TestsPanel state={ran(refused)} source={SOURCE} onReveal={vi.fn()} />);
    expect(
      screen.getByText("The case refused to rate: No rate for this weight"),
    ).toBeInTheDocument();
  });
});

describe("TestsPanel coverage", () => {
  it("says how many regions the tests reach and lists the unreached ones to reveal", async () => {
    const user = userEvent.setup();
    const reveal = vi.fn();
    const result = run();
    const coverage = result.coverage!;
    render(<TestsPanel state={ran(result)} source={SOURCE} onReveal={reveal} />);
    expect(screen.getByText("Tests reach 4 of 6 regions")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Show 1 unreached region" }));
    const list = screen.getByRole("list", { name: "Regions no test reaches" });
    const [only, ...rest] = within(list).getAllByRole("button");
    expect(rest).toEqual([]);
    expect(only).toHaveTextContent("If branch");
    expect(only).toHaveTextContent("if shipment.distance > 5_000 mi {");
    await user.click(only!);
    expect(reveal).toHaveBeenCalledWith(
      coverage.regions.find((region) => region.hits === 0)?.range,
    );
  });

  it("shows no coverage line when the run reports none", () => {
    render(
      <TestsPanel state={ran({ ...run(), coverage: null })} source={SOURCE} onReveal={vi.fn()} />,
    );
    expect(screen.queryByRole("region", { name: "Coverage" })).not.toBeInTheDocument();
  });
});

// A run in the shape `tariff.test` answers for testdata/tariff/eval/reviews.json's file: one test
// whose bill asks for two reviews, one with none, and the review region no test reached, beside
// a condition's two outcomes (eval/coverage.json), both on the condition's final expression.
const REVIEWS_SOURCE = `tariff "t"
charge "Linehaul" = $100
if shipment.hazmat {
  review "Hazmat"
}
review "Always check"
`;

const reviewBill = (reviews: { text: string; range: { from: number; to: number } }[]) => ({
  lines: [
    {
      kind: "charge",
      label: "Linehaul",
      code: null,
      freight: true,
      amount: { $q: "100", u: "$" },
      quantity: null,
      rate: null,
      limit: null,
      on: null,
      range: { from: 11, to: 35 },
    },
  ],
  notes: [],
  reviews,
  freightTotal: { $q: "100", u: "$" },
  accessorialTotal: { $q: "0", u: "$" },
  total: { $q: "100", u: "$" },
  converted: null,
});

const reviewsRun = tariffTestRunSchema.parse({
  tests: [
    {
      name: "hazmat waits for approval",
      range: { from: 80, to: 160 },
      kind: "test",
      status: "passed",
      failures: [],
      bill: reviewBill([
        { text: "Hazmat", range: { from: 59, to: 74 } },
        { text: "Always check", range: { from: 77, to: 98 } },
      ]),
      error: null,
      cases: 1,
      skipped: 0,
      counterexample: null,
    },
    {
      name: "a plain load",
      range: { from: 161, to: 200 },
      kind: "test",
      status: "passed",
      failures: [],
      bill: reviewBill([]),
      error: null,
      cases: 1,
      skipped: 0,
      counterexample: null,
    },
  ],
  needs: [],
  coverage: {
    regions: [
      { range: { from: 11, to: 35 }, kind: "charge", hits: 2 },
      { range: { from: 55, to: 76 }, kind: "if", hits: 1 },
      { range: { from: 59, to: 74 }, kind: "review", hits: 1 },
      { range: { from: 77, to: 98 }, kind: "review", hits: 0 },
    ],
    reached: 3,
    total: 4,
  },
});

// testdata/tariff/eval/coverage.json's condition, with only the "applies" test run.
const CONDITION_SOURCE = `condition "c"
shipment.hazmat and (if shipment.reefer { true } else { false })
test "a cold hazmat load applies" {
  given shipment.hazmat = true
  given shipment.reefer = true
  expect applies
}
`;

const conditionRun = tariffTestRunSchema.parse({
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
      { range: { from: 14, to: 78 }, kind: "notApplies", hits: 0 },
      { range: { from: 56, to: 60 }, kind: "if", hits: 1 },
      { range: { from: 70, to: 75 }, kind: "else", hits: 0 },
    ],
    reached: 2,
    total: 4,
  },
});

describe("TestsPanel with reviews and condition outcomes", () => {
  it("counts the reviews a test's bill asks for on its row", () => {
    render(<TestsPanel state={ran(reviewsRun)} source={REVIEWS_SOURCE} onReveal={vi.fn()} />);
    const held = screen.getByText("hazmat waits for approval").closest("li")!;
    expect(within(held).getByText("2 reviews")).toBeInTheDocument();
    const plain = screen.getByText("a plain load").closest("li")!;
    expect(within(plain).queryByText(/review/)).toBeNull();
  });

  it("names a review no test reaches", async () => {
    const user = userEvent.setup();
    render(<TestsPanel state={ran(reviewsRun)} source={REVIEWS_SOURCE} onReveal={vi.fn()} />);
    await user.click(screen.getByRole("button", { name: "Show 1 unreached region" }));
    const list = screen.getByRole("list", { name: "Regions no test reaches" });
    expect(within(list).getByRole("listitem")).toHaveTextContent('Reviewreview "Always check"');
  });

  it("names the condition outcome no test sees", async () => {
    const user = userEvent.setup();
    const reveal = vi.fn();
    render(<TestsPanel state={ran(conditionRun)} source={CONDITION_SOURCE} onReveal={reveal} />);
    expect(screen.getByText("Tests reach 2 of 4 regions")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Show 2 unreached regions" }));
    const list = screen.getByRole("list", { name: "Regions no test reaches" });
    const items = within(list).getAllByRole("listitem");
    expect(items.map((item) => item.querySelector("span")?.textContent)).toEqual([
      "Condition does not apply",
      "Else branch",
    ]);
    await user.click(within(items[0]!).getByRole("button"));
    expect(reveal).toHaveBeenCalledWith({ from: 14, to: 78 });
  });
});
