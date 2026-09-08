import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { RandomTestingSkeleton } from "../random-testing-skeleton";

afterEach(cleanup);

// The shapes are aria-hidden so a screen reader hears one "loading", not a
// tour of empty regions; the queries opt into hidden nodes to see them.
const hidden = { hidden: true } as const;

describe("RandomTestingSkeleton", () => {
  // The skeleton is the Suspense fallback for the route and the console's own
  // loading state. It has to be the loaded page's shape, section for section,
  // or the layout jumps twice: once when the chunk lands and once for the data.
  it("mirrors every section of the loaded console", () => {
    render(<RandomTestingSkeleton />);

    const root = screen.getByLabelText("Loading random testing");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.querySelectorAll(".lg\\:grid-cols-8 > .col-span-2")).toHaveLength(4);

    const pools = within(root).getByRole("list", { name: "Pools", ...hidden });
    const rows = within(pools).getAllByRole("listitem", hidden);
    expect(rows.length).toBeGreaterThan(1);
    // A pool drawn quarterly and one drawn monthly: the slot strips differ.
    const slots = rows.map((row) => row.querySelectorAll(".h-7.w-9").length);
    expect(new Set(slots).size).toBeGreaterThan(1);

    const rounds = within(root).getByRole("table", { name: "Rounds", ...hidden });
    const [heading] = within(rounds).getAllByRole("row", hidden);
    expect(within(heading!).getAllByRole("columnheader", hidden)).toHaveLength(8);
    expect(within(rounds).getAllByRole("row", hidden).length).toBeGreaterThan(3);
  });

  // The loaded panels are found by their accessible names. If the skeleton's
  // stand-ins answered to the same names, a test (or a screen reader) waiting
  // for the real panel would be handed the empty one.
  it("keeps its stand-ins out of the accessibility tree", () => {
    render(<RandomTestingSkeleton />);
    expect(screen.queryByRole("table", { name: "Rounds" })).toBeNull();
    expect(screen.queryByRole("list", { name: "Pools" })).toBeNull();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    expect(screen.getByLabelText("Loading random testing")).toHaveTextContent("");
  });
});
