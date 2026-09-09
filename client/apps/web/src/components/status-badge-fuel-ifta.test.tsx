import {
  FuelCardStatusBadge,
  FuelPurchaseImportStatusBadge,
  IftaReturnStatusBadge,
} from "@trenova/shared/components/status-badge";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

afterEach(cleanup);

// Each badge carries its explanation in the title so a hover says what the
// status means for the record, in the same words the workspace uses.

describe("IftaReturnStatusBadge", () => {
  it.each([
    ["Draft", "Worksheet can still change"],
    ["Finalized", "Locked; reopen with a reason to change"],
    ["Filed", "Submitted to the base jurisdiction"],
  ] as const)("renders %s with its meaning", (status, description) => {
    render(<IftaReturnStatusBadge status={status} />);
    const badge = screen.getByText(status);
    expect(badge.closest("[title]")).toHaveAttribute("title", description);
  });
});

describe("FuelPurchaseImportStatusBadge", () => {
  it.each([
    ["Pending", "Pending"],
    ["Parsed", "Parsed"],
    ["Committed", "Committed"],
    ["Discarded", "Discarded"],
    ["Failed", "Failed"],
  ] as const)("labels %s", (status, label) => {
    render(<FuelPurchaseImportStatusBadge status={status} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("explains that only a parsed batch can be imported", () => {
    render(<FuelPurchaseImportStatusBadge status="Parsed" />);
    expect(screen.getByText("Parsed").closest("[title]")?.getAttribute("title")).toMatch(/review/i);
  });
});

describe("FuelCardStatusBadge", () => {
  it.each([["Active"], ["Suspended"], ["Cancelled"]] as const)("labels %s", (status) => {
    render(<FuelCardStatusBadge status={status} />);
    expect(screen.getByText(status)).toBeInTheDocument();
  });

  it("says a cancelled card cannot come back", () => {
    render(<FuelCardStatusBadge status="Cancelled" />);
    expect(screen.getByText("Cancelled").closest("[title]")?.getAttribute("title")).toMatch(
      /cannot be reactivated/i,
    );
  });
});
