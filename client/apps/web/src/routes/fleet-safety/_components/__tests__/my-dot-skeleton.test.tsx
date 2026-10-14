import { cleanup, render, screen, within } from "@testing-library/react";
import { CSA_BASIC_ORDER } from "@trenova/shared/lib/csa";
import { afterEach, describe, expect, it } from "vitest";
import { MyDotSkeleton } from "../my-dot-skeleton";

afterEach(cleanup);

const hidden = { hidden: true } as const;

describe("MyDotSkeleton", () => {
  // The profile opens on safety, whose CSA BASICs table always has a row for
  // each of the seven BASICs; the outline draws the same rows.
  it("outlines the profile down to a row for each CSA BASIC", () => {
    render(<MyDotSkeleton />);

    const root = screen.getByLabelText("Loading your DOT profile");
    expect(root).toHaveAttribute("aria-busy", "true");
    const table = within(root).getByRole("table", hidden);
    expect(within(table).getAllByRole("row", hidden)).toHaveLength(CSA_BASIC_ORDER.length + 1);
  });

  it("keeps its stand-ins out of the accessibility tree and carries no text", () => {
    render(<MyDotSkeleton />);

    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    expect(screen.getByLabelText("Loading your DOT profile")).toHaveTextContent("");
  });
});
