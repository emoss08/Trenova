import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CostControlSkeleton } from "../skeleton";

function cards(root: HTMLElement) {
  return Array.from(root.querySelectorAll<HTMLElement>('[data-slot="card"]'));
}

function fieldKinds(card: HTMLElement) {
  return Array.from(card.querySelectorAll<HTMLElement>('[data-slot="field-skeleton"]')).map(
    (field) => field.dataset.kind,
  );
}

describe("CostControlSkeleton", () => {
  // A screen reader hears one busy region, not a tour of grey shapes.
  it("is announced once as loading, with every placeholder hidden", () => {
    render(<CostControlSkeleton />);

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.children).toHaveLength(1);
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });

  // The fallback is the form's own layout: cost basis, variable costs, fixed costs, GL actuals.
  it("draws the form's cards in order with the fields each one always shows", () => {
    render(<CostControlSkeleton />);

    const [costBasis, variable, fixed, glActuals, ...rest] = cards(
      screen.getByLabelText("Loading settings"),
    );
    expect(rest).toHaveLength(0);
    expect(fieldKinds(costBasis!)).toEqual(["switch", "input", "switch", "input"]);
    expect(fieldKinds(glActuals!)).toEqual(["switch"]);

    const category = ["switch", "switch", "input"];
    expect(fieldKinds(variable!)).toEqual(Array.from({ length: 6 }, () => category).flat());
    expect(fieldKinds(fixed!)).toEqual(Array.from({ length: 4 }, () => category).flat());
  });

  // Six variable and four fixed categories are what every organization starts with
  // (costingcontrol/defaults.go), each drawn as a headed row ruled from the next.
  it("draws one ruled row per default cost category", () => {
    render(<CostControlSkeleton />);

    const [, variable, fixed] = cards(screen.getByLabelText("Loading settings"));
    for (const [card, count] of [
      [variable!, 6],
      [fixed!, 4],
    ] as const) {
      const rows = Array.from(card.querySelectorAll('[data-slot="record-row-skeleton"]'));
      expect(rows).toHaveLength(count);
      expect(rows.filter((row) => row.classList.contains("border-b"))).toHaveLength(count - 1);
      expect(rows.at(-1)).not.toHaveClass("border-b");
    }
  });
});
