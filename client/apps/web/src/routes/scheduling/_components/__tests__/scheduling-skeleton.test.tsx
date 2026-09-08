import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { RotaBoardSkeleton, SchedulingSkeleton } from "../scheduling-skeleton";

afterEach(cleanup);

// The shapes are aria-hidden so a screen reader hears one "loading", not a
// tour of empty regions; the queries opt into hidden nodes to see them.
const hidden = { hidden: true } as const;

describe("SchedulingSkeleton", () => {
  // The skeleton is the Suspense fallback for the route. It has to be the
  // loaded page's shape, section for section, or the layout jumps twice: once
  // when the chunk lands and once for the week.
  it("mirrors every section of the loaded console", () => {
    render(<SchedulingSkeleton />);

    const root = screen.getByLabelText("Loading scheduling");
    expect(root).toHaveAttribute("aria-busy", "true");

    const attention = within(root).getByRole("region", { name: "Needs a look", ...hidden });
    expect(within(attention).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);

    const board = within(root).getByRole("table", { name: "Rota", ...hidden });
    const [heading] = within(board).getAllByRole("row", hidden);
    // A worker column, seven days and a week total.
    expect(within(heading!).getAllByRole("columnheader", hidden)).toHaveLength(9);
    expect(within(board).getAllByRole("row", hidden).length).toBeGreaterThan(3);
  });

  it("leaves out the swaps card when swaps are not readable", () => {
    render(<SchedulingSkeleton showSwaps={false} />);
    const root = screen.getByLabelText("Loading scheduling");
    expect(root.querySelector(".lg\\:grid-cols-6")).not.toBeNull();
    expect(root.querySelector(".lg\\:grid-cols-8")).toBeNull();
  });

  // The loaded board is found by role. If the skeleton's stand-ins answered
  // to the same roles, a test (or a screen reader) waiting for the real board
  // would be handed the empty one.
  it("keeps its stand-ins out of the accessibility tree", () => {
    render(<SchedulingSkeleton />);
    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.queryByRole("region", { name: "Needs a look" })).toBeNull();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    expect(screen.getByLabelText("Loading scheduling")).toHaveTextContent("");
  });
});

describe("RotaBoardSkeleton", () => {
  // The board outline is drawn in the width and density the reader chose, so
  // the loaded week lands in the same place its outline was.
  it("draws seven days for every week on the board", () => {
    render(<RotaBoardSkeleton weeks={2} />);
    const board = screen.getByRole("table", { name: "Rota", ...hidden });
    const [heading] = within(board).getAllByRole("row", hidden);
    expect(within(heading!).getAllByRole("columnheader", hidden)).toHaveLength(16);
    expect(board.closest("[data-cell-mode]")).toHaveAttribute("data-cell-mode", "block");
  });

  it("tightens its rows in the compact density", () => {
    render(<RotaBoardSkeleton density="compact" />);
    const board = screen.getByRole("table", { name: "Rota", ...hidden });
    expect(board.closest("[data-density]")).toHaveAttribute("data-density", "compact");
    expect(board.closest("[data-cell-mode]")).toHaveAttribute("data-cell-mode", "time");
  });
});
