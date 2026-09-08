import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { BenefitsSkeleton } from "../benefits-skeleton";

afterEach(cleanup);

// The shapes are aria-hidden so a screen reader hears one "loading", not a
// tour of empty regions; the queries opt into hidden nodes to see them.
const hidden = { hidden: true } as const;

describe("BenefitsSkeleton", () => {
  // The skeleton is the Suspense fallback for the route and the console's own
  // loading state. It has to be the loaded page's shape, section for section,
  // or the layout jumps twice: once when the chunk lands and once for the data.
  it("mirrors every section of the loaded console", () => {
    render(<BenefitsSkeleton />);

    const root = screen.getByLabelText("Loading benefits");
    expect(root).toHaveAttribute("aria-busy", "true");

    // Plans are grouped by kind, each group a heading over its own card.
    const medical = within(root).getByRole("region", { name: "Medical", ...hidden });
    expect(within(medical).getAllByRole("listitem", hidden).length).toBeGreaterThan(1);
    const dental = within(root).getByRole("region", { name: "Dental", ...hidden });
    expect(within(dental).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);

    for (const name of ["Starting soon", "Ending soon", "Recently declined"]) {
      const panel = within(root).getByRole("region", { name, ...hidden });
      expect(within(panel).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);
    }
  });

  // The loaded panels are found by their accessible names. If the skeleton's
  // stand-ins answered to the same names, a test (or a screen reader) waiting
  // for the real panel would be handed the empty one.
  it("keeps its stand-in regions out of the accessibility tree", () => {
    render(<BenefitsSkeleton />);
    expect(screen.queryByRole("region", { name: "Medical" })).toBeNull();
    expect(screen.queryByRole("region", { name: "Starting soon" })).toBeNull();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    expect(screen.getByLabelText("Loading benefits")).toHaveTextContent("");
  });
});
