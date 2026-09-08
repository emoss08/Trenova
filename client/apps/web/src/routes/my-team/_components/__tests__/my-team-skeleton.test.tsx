import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { MyTeamSkeleton } from "../my-team-skeleton";

afterEach(cleanup);

// The shapes are aria-hidden so a screen reader hears one "loading", not a
// tour of empty regions; the queries opt into hidden nodes to see them.
const hidden = { hidden: true } as const;

describe("MyTeamSkeleton", () => {
  // The skeleton is the Suspense fallback for the route and the console's own
  // loading state. It has to be the loaded page's shape, section for section,
  // or the layout jumps twice: once when the chunk lands and once for the team.
  it("mirrors every section of the loaded console", () => {
    render(<MyTeamSkeleton />);

    const root = screen.getByLabelText("Loading your team");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.querySelectorAll(".lg\\:grid-cols-8 > .col-span-2")).toHaveLength(4);

    const attention = within(root).getByRole("region", { name: "Needs your attention", ...hidden });
    expect(within(attention).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);

    // The roster is grouped by how somebody reaches the manager.
    const direct = within(root).getByRole("region", { name: "Direct reports", ...hidden });
    const viaTerminal = within(root).getByRole("region", { name: "Through a terminal", ...hidden });
    expect(within(direct).getAllByRole("listitem", hidden).length).toBeGreaterThan(
      within(viaTerminal).getAllByRole("listitem", hidden).length,
    );

    for (const name of ["By terminal", "Coming up", "Approval cover"]) {
      const panel = within(root).getByRole("region", { name, ...hidden });
      expect(within(panel).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);
    }
  });

  // Somebody who may not read delegations gets no cover panel once loaded,
  // so the skeleton must draw the same.
  it("leaves out the cover panel when cover is not readable", () => {
    render(<MyTeamSkeleton showCover={false} />);
    const root = screen.getByLabelText("Loading your team");
    expect(within(root).queryByRole("region", { name: "Approval cover", ...hidden })).toBeNull();
    expect(within(root).getByRole("region", { name: "Coming up", ...hidden })).toBeInTheDocument();
  });

  // The loaded panels are found by their accessible names. If the skeleton's
  // stand-ins answered to the same names, a test (or a screen reader) waiting
  // for the real panel would be handed the empty one.
  it("keeps its stand-ins out of the accessibility tree", () => {
    render(<MyTeamSkeleton />);
    expect(screen.queryByRole("region", { name: "Direct reports" })).toBeNull();
    expect(screen.queryByRole("region", { name: "Needs your attention" })).toBeNull();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    expect(screen.getByLabelText("Loading your team")).toHaveTextContent("");
  });
});
