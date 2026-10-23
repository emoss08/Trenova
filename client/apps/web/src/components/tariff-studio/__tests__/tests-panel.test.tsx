import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffTest } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT, type TariffTestRun } from "@trenova/shared/types/tariff";
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
