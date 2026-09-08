import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { countDeskFilters, deskFloorStats, summarizeDesk } from "@trenova/shared/lib/detention";
import type { DeskEntry } from "@trenova/shared/types/detention";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DetentionDesk } from "../detention-desk";
import type { DetentionDeskState } from "../use-detention-desk";

vi.mock("../occurrence-detail-sheet", () => ({ OccurrenceDetailSheet: () => null }));

const NOW = 1_757_000_000;

function deskState(over: Partial<DetentionDeskState> = {}): DetentionDeskState {
  const entries: DeskEntry[] = [];
  return {
    nowSeconds: NOW,
    entries,
    visible: [],
    summary: summarizeDesk(entries),
    laneCounts: countDeskFilters(entries),
    floor: deskFloorStats(entries, NOW),
    noticeQueue: [],
    lane: "all",
    sort: "urgency",
    search: "",
    selectedId: null,
    isFiltered: false,
    isLoading: false,
    isError: false,
    isFetching: false,
    updatedSecondsAgo: 0,
    setLane: vi.fn(),
    setSort: vi.fn(),
    setSearch: vi.fn(),
    selectStop: vi.fn(),
    resetFilters: vi.fn(),
    refetch: vi.fn(),
    ...over,
  } as DetentionDeskState;
}

function renderDesk(desk: DetentionDeskState) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <DetentionDesk desk={desk} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
});

describe("detention desk empty states", () => {
  it("draws the desk it will become when nobody is on a dock", () => {
    renderDesk(deskState());

    expect(
      screen.getByRole("heading", { name: "No drivers are sitting on a dock" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/arrival is recorded/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  // The rail and the toolbar stay put above the sketch, so only the rows are
  // drawn; clearing the lane and search is the way back to the whole floor.
  it("offers to clear the lane and search when they hide every stop", async () => {
    const resetFilters = vi.fn();
    const user = userEvent.setup();
    renderDesk(
      deskState({
        entries: [{ occurrence: { id: "occ_1" } } as DeskEntry],
        visible: [],
        isFiltered: true,
        search: "zzz",
        resetFilters,
      }),
    );

    expect(screen.getByRole("heading", { name: "No stops match this view" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(resetFilters).toHaveBeenCalledTimes(1);
  });
});
