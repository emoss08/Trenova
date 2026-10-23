import { cleanup, render, screen, within } from "@testing-library/react";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffCheck } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { afterEach, describe, expect, it, vi } from "vitest";
import { OutlinePanel } from "../outline-panel";

afterEach(cleanup);

const SOURCE = `tariff "LTL"
best "Linehaul" {
  option "Per mile" {
    charge "Linehaul" (quantity: shipment.distance, rate: $2/mi)
  }
  option "Flat" {
    charge "Flat rate" = $900
  }
}

property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  expect total nondecreasing in shipment.weight
}

test "a long haul" {
  given shipment.distance = 900 mi
  expect total == $900
}
`;

const outline = () =>
  tariffCheck(rulesSync(), { source: SOURCE, kind: "tariff", context: EMPTY_TARIFF_CONTEXT })
    .outline;

describe("OutlinePanel", () => {
  it("nests each choice's options and their charges under Charges", () => {
    render(<OutlinePanel outline={outline()} onReveal={vi.fn()} />);
    const charges = screen.getByRole("heading", { name: "Charges" }).parentElement!;
    expect(
      within(charges)
        .getAllByRole("button")
        .map((row) => row.textContent),
    ).toEqual(["LinehaulChoice", "Per mile", "Linehaul", "Flat", "Flat rate"]);
  });

  it("lists properties with the tests, each with its verdict", () => {
    render(
      <OutlinePanel
        outline={outline()}
        onReveal={vi.fn()}
        testStatus={
          new Map([
            ["heavier never bills less", "failed"],
            ["a long haul", "passed"],
          ])
        }
      />,
    );
    const tests = screen.getByRole("heading", { name: "Tests" }).parentElement!;
    const rows = within(tests).getAllByRole("button");
    expect(rows.map((row) => row.textContent)).toEqual([
      "heavier never bills lessProperty",
      "a long haul",
    ]);
    expect(within(rows[0]!).getByLabelText("Failed")).toBeInTheDocument();
    expect(within(rows[1]!).getByLabelText("Passed")).toBeInTheDocument();
  });
});
