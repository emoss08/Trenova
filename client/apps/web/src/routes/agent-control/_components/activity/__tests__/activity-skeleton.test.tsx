import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { ActivityView } from "../../../ai-control-tabs";
import { ActivityTabSkeleton } from "../activity-skeleton";

afterEach(() => {
  cleanup();
});

describe("ActivityTabSkeleton", () => {
  it("heads the table with Nova's sentence and today's four figures", () => {
    const { container } = render(<ActivityTabSkeleton view="runs" />);

    const root = container.firstElementChild;
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root).toHaveAttribute("aria-label", "Loading activity");
    expect(root?.querySelectorAll(".hero")).toHaveLength(1);
    expect(root?.querySelectorAll(".figs .fg")).toHaveLength(4);
    expect(container.textContent).toBe("");
  });

  it.each<[ActivityView, number]>([
    ["runs", 6],
    ["proposals", 6],
    ["plans", 5],
    ["evaluations", 5],
    ["exceptions", 7],
  ])("draws the %s table with the %i columns it opens with", (view, columns) => {
    const { container } = render(<ActivityTabSkeleton view={view} />);

    expect(container.querySelectorAll("th")).toHaveLength(columns);
  });
});
