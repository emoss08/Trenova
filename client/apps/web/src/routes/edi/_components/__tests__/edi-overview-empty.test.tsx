import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EDIOverview } from "../overview/edi-overview";
import * as summaryHook from "../overview/use-edi-summary";

const QUIET = {
  ediSummary: {
    deliveryStatusCounts: [],
    ackStatusCounts: [],
    inboundFileStatusCounts: [],
    inboundTransferStatusCounts: [],
    overdueAckCount: 0,
    attentionItems: [],
  },
};

const BUSY = {
  ediSummary: {
    ...QUIET.ediSummary,
    deliveryStatusCounts: [{ status: "Sent", count: 12 }],
  },
};

// One helper stands in for three hooks whose return types differ in the shape of their
// `data`. Generic over the hook's own result so each spy is handed what it expects;
// pinning it to useEDISummary's result made the scorecards and volume-series spies
// reject it.
function loaded<T>(data: unknown): T {
  return { data, isLoading: false, isError: false } as unknown as T;
}

function renderOverview() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <EDIOverview />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("EDIOverview empty state", () => {
  // A quiet window is not eight zeros: the overview draws what it will be
  // and offers the widest window, since the traffic may simply be older.
  it("draws the overview it will become when nothing moved in the window", async () => {
    const summary = vi.spyOn(summaryHook, "useEDISummary").mockReturnValue(loaded(QUIET));
    vi.spyOn(summaryHook, "useEDIPartnerScorecards").mockReturnValue(
      loaded({ ediPartnerScorecards: [] }),
    );
    vi.spyOn(summaryHook, "useEDIVolumeSeries").mockReturnValue(loaded({ ediVolumeSeries: [] }));
    const user = userEvent.setup();
    renderOverview();

    expect(screen.getByRole("heading", { name: "Nothing in this window" })).toBeInTheDocument();
    expect(screen.getByText(/last 24 hours/)).toBeInTheDocument();
    expect(screen.queryByText("Dead-lettered messages")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Look at everything" }));
    expect(summary).toHaveBeenLastCalledWith(undefined);
  });

  it("points at trading partners once the widest window is empty too", async () => {
    vi.spyOn(summaryHook, "useEDISummary").mockReturnValue(loaded(QUIET));
    vi.spyOn(summaryHook, "useEDIPartnerScorecards").mockReturnValue(
      loaded({ ediPartnerScorecards: [] }),
    );
    vi.spyOn(summaryHook, "useEDIVolumeSeries").mockReturnValue(loaded({ ediVolumeSeries: [] }));
    const user = userEvent.setup();
    renderOverview();

    await user.click(screen.getByRole("button", { name: "All" }));
    expect(screen.getByRole("heading", { name: "Nothing yet" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Look at everything" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Set up a trading partner" })).toHaveAttribute(
      "href",
      "/edi/partners",
    );
  });

  it("keeps the tiles, zeros included, once anything has moved", () => {
    vi.spyOn(summaryHook, "useEDISummary").mockReturnValue(loaded(BUSY));
    vi.spyOn(summaryHook, "useEDIPartnerScorecards").mockReturnValue(
      loaded({ ediPartnerScorecards: [] }),
    );
    vi.spyOn(summaryHook, "useEDIVolumeSeries").mockReturnValue(loaded({ ediVolumeSeries: [] }));
    renderOverview();

    expect(screen.getByText("Dead-lettered messages")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /Nothing/ })).not.toBeInTheDocument();
  });
});
