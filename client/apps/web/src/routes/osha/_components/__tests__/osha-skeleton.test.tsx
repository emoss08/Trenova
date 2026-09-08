import { YEARS_OFFERED } from "@/lib/osha-log";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { OshaLogSkeleton } from "../osha-skeleton";

afterEach(cleanup);

// The shapes are aria-hidden so a screen reader hears one "loading", not a
// tour of empty regions; the queries opt into hidden nodes to see them.
const hidden = { hidden: true } as const;

describe("OshaLogSkeleton", () => {
  // The skeleton is the Suspense fallback for the route and the console's own
  // loading state. It has to be the loaded page's shape, section for section,
  // or the layout jumps twice: once when the chunk lands and once for the log.
  it("mirrors every section of the loaded console", () => {
    render(<OshaLogSkeleton />);

    const root = screen.getByLabelText("Loading the log");
    expect(root).toHaveAttribute("aria-busy", "true");

    // One tab per year offered, each with its caption line.
    expect(root.firstElementChild?.firstElementChild?.childElementCount).toBe(YEARS_OFFERED);

    const summary = within(root).getByRole("region", { name: "Form 300A", ...hidden });
    for (const block of [
      "Number of cases",
      "Number of days",
      "Injury and illness types",
      "Establishment information",
    ]) {
      expect(within(summary).getByRole("region", { name: block, ...hidden })).toBeInTheDocument();
    }

    const track = within(root).getByRole("list", { name: "Where the year stands", ...hidden });
    expect(within(track).getAllByRole("listitem", hidden)).toHaveLength(4);

    const log = within(root).getByRole("region", { name: "Form 300", ...hidden });
    const table = within(log).getByRole("table", { name: "Cases", ...hidden });
    const [heading] = within(table).getAllByRole("row", hidden);
    expect(within(heading!).getAllByRole("columnheader", hidden)).toHaveLength(11);
    expect(within(table).getAllByRole("row", hidden).length).toBeGreaterThan(3);
  });

  // The loaded panels are found by their accessible names. If the skeleton's
  // stand-ins answered to the same names, a test (or a screen reader) waiting
  // for the real panel would be handed the empty one.
  it("keeps its stand-ins out of the accessibility tree", () => {
    render(<OshaLogSkeleton />);
    expect(screen.queryByRole("region", { name: "Form 300A" })).toBeNull();
    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.queryAllByRole("row")).toHaveLength(0);
    expect(screen.getByLabelText("Loading the log")).toHaveTextContent("");
  });
});
