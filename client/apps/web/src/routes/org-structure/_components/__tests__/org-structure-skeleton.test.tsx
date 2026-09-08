import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { OrgStructureSkeleton } from "../org-structure-skeleton";

afterEach(cleanup);

// The shapes are aria-hidden so a screen reader hears one "loading", not a
// tour of empty regions; the queries opt into hidden nodes to see them.
const hidden = { hidden: true } as const;

describe("OrgStructureSkeleton", () => {
  // The skeleton is the Suspense fallback for the route and the console's own
  // loading state. It has to be the loaded page's shape, section for section,
  // or the layout jumps twice: once when the chunk lands and once for the data.
  it("mirrors every section of the loaded console", () => {
    render(<OrgStructureSkeleton />);

    const root = screen.getByLabelText("Loading the organisation");
    expect(root).toHaveAttribute("aria-busy", "true");

    const chart = within(root).getByRole("region", { name: "Org chart", ...hidden });
    const positions = within(chart).getByRole("list", { name: "Positions", ...hidden });
    const rows = within(positions).getAllByRole("listitem", hidden);
    expect(rows.length).toBeGreaterThan(1);
    // An indented tree, not a flat list: at least one row sits under another.
    expect(rows.some((row) => row.style.paddingLeft !== rows[0]?.style.paddingLeft)).toBe(true);

    const cover = within(root).getByRole("region", { name: "Approval cover", ...hidden });
    expect(within(cover).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);

    for (const name of ["By terminal", "By department", "Titles nobody holds yet"]) {
      const panel = within(root).getByRole("region", { name, ...hidden });
      expect(within(panel).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);
    }
  });

  // Somebody who may not read delegations gets a three-card strip and no
  // cover panel once loaded, so the skeleton must draw the same.
  it("leaves out the cover card and panel when cover is not readable", () => {
    render(<OrgStructureSkeleton showCover={false} />);

    const root = screen.getByLabelText("Loading the organisation");
    expect(within(root).queryByRole("region", { name: "Approval cover", ...hidden })).toBeNull();
    expect(root.querySelector(".lg\\:grid-cols-6")).not.toBeNull();
    expect(root.querySelector(".lg\\:grid-cols-8")).toBeNull();
  });

  // The loaded panels are found by their accessible names. If the skeleton's
  // stand-ins answered to the same names, a test (or a screen reader) waiting
  // for the real panel would be handed the empty one.
  it("keeps its stand-in regions out of the accessibility tree", () => {
    render(<OrgStructureSkeleton />);
    expect(screen.queryByRole("region", { name: "By terminal" })).toBeNull();
    expect(screen.queryByRole("list", { name: "Positions" })).toBeNull();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    expect(screen.getByLabelText("Loading the organisation")).toHaveTextContent("");
  });
});
