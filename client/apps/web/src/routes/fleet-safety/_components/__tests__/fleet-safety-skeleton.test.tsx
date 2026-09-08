import { cleanup, render, screen, within } from "@testing-library/react";
import { CSA_BASIC_ORDER, SAFETY_EVENT_KIND_LABELS } from "@trenova/shared/lib/csa";
import { afterEach, describe, expect, it } from "vitest";
import { FleetSafetySkeleton } from "../fleet-safety-skeleton";

afterEach(cleanup);

describe("FleetSafetySkeleton", () => {
  // The skeleton is the Suspense fallback for the route and the console's own
  // loading state. It has to be the loaded page's shape, section for section,
  // or the layout jumps twice: once when the chunk lands and once for the data.
  it("mirrors every section of the loaded console", () => {
    render(<FleetSafetySkeleton />);

    const root = screen.getByLabelText("Loading fleet safety");
    expect(root).toHaveAttribute("aria-busy", "true");

    // The shapes are aria-hidden so a screen reader hears one "loading", not
    // a tour of empty regions; the queries opt into hidden nodes to see them.
    const hidden = { hidden: true } as const;
    const basics = within(root).getByRole("region", { name: "CSA BASICs", ...hidden });
    expect(within(basics).getAllByRole("listitem", hidden)).toHaveLength(CSA_BASIC_ORDER.length);

    const trend = within(root).getByRole("region", { name: "Events by month", ...hidden });
    const kinds = within(trend).getByRole("list", { name: "Events by kind", ...hidden });
    expect(within(kinds).getAllByRole("listitem", hidden)).toHaveLength(
      Object.keys(SAFETY_EVENT_KIND_LABELS).length + 1,
    );

    for (const name of ["By terminal", "Needs attention", "Best records"]) {
      const panel = within(root).getByRole("region", { name, ...hidden });
      expect(within(panel).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);
    }
  });

  // The loaded cards are found by their accessible names. If the skeleton's
  // stand-ins answered to the same names, a test (or a screen reader) waiting
  // for the real card would be handed the empty one.
  it("keeps its stand-in regions out of the accessibility tree", () => {
    render(<FleetSafetySkeleton />);
    expect(screen.queryByRole("region", { name: "CSA BASICs" })).toBeNull();
    expect(screen.queryByRole("region", { name: "Needs attention" })).toBeNull();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
  });

  it("carries no text a reader could mistake for a figure", () => {
    render(<FleetSafetySkeleton />);
    expect(screen.getByLabelText("Loading fleet safety")).toHaveTextContent("");
  });
});
