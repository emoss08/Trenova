import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SettlementHistoryEmpty } from "../settlement-history-empty";

afterEach(() => {
  cleanup();
});

describe("SettlementHistoryEmpty", () => {
  it("says where the record comes from when nothing has been settled", () => {
    render(<SettlementHistoryEmpty hasActiveFilters={false} onClearFilters={vi.fn()} />);

    expect(screen.getByRole("heading", { name: "No settlements yet" })).toBeInTheDocument();
    expect(screen.getByText(/generated in the workspace/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  it("offers to clear the filters when they are what emptied the record", async () => {
    const onClearFilters = vi.fn();
    const user = userEvent.setup();
    render(<SettlementHistoryEmpty hasActiveFilters onClearFilters={onClearFilters} />);

    expect(screen.getByRole("heading", { name: "Nothing matches" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(onClearFilters).toHaveBeenCalledTimes(1);
  });
});
