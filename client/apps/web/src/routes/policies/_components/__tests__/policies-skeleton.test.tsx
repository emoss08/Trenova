import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { PoliciesSkeleton, PolicyCardsSkeleton } from "../policies-skeleton";

afterEach(cleanup);

// The shapes are aria-hidden so a screen reader hears one "loading", not a
// tour of empty cards; the queries opt into hidden nodes to see them.
const hidden = { hidden: true } as const;

describe("PoliciesSkeleton", () => {
  // The skeleton is the Suspense fallback for the route. It has to be the
  // loaded page's shape, section for section, or the layout jumps twice: once
  // when the chunk lands and once for the policies.
  it("mirrors every section of the loaded console", () => {
    render(<PoliciesSkeleton />);

    const [root] = screen.getAllByLabelText("Loading policies");
    expect(root).toHaveAttribute("aria-busy", "true");
    // Three stats across a six-column strip, each two wide.
    expect(root!.querySelectorAll(".grid-cols-6 > .col-span-2")).toHaveLength(3);
    const cards = within(root!).getByRole("list", { name: "Loading policies", ...hidden });
    expect(within(cards).getAllByRole("listitem", hidden).length).toBeGreaterThan(2);
  });

  // The loaded cards carry buttons and text a test waits on. If the skeleton
  // answered to the same queries, the wait would be handed the empty cards.
  it("keeps its stand-ins out of the accessibility tree", () => {
    render(<PoliciesSkeleton />);
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    expect(screen.queryByRole("list")).toBeNull();
    expect(screen.getAllByLabelText("Loading policies")[0]).toHaveTextContent("");
  });
});

describe("PolicyCardsSkeleton", () => {
  it("draws the card grid alone, with every card hidden from assistive technology", () => {
    render(<PolicyCardsSkeleton />);
    const grid = screen.getByLabelText("Loading policies");
    expect(grid).toHaveAttribute("aria-busy", "true");
    expect(within(grid).getAllByRole("listitem", hidden).length).toBeGreaterThan(2);
    expect(within(grid).queryAllByRole("listitem")).toHaveLength(0);
  });
});
