import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { ShipmentBoardCapabilities } from "@/lib/graphql/shipment-board";
import { BoardCapabilitiesProvider } from "@/lib/shipment-board/capabilities";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ShipmentBriefing } from "./briefing/shipment-briefing";
import { ShipmentBriefingSkeleton } from "./briefing/shipment-briefing-skeleton";
import { CapacityStrip } from "./capacity/capacity-strip";
import { ShipmentSidePanelSkeleton } from "./panel/side-panel-skeleton";

// Every board read stays unanswered, so each part stays on its first-load state.
vi.mock("@/lib/queries/shipment-board", () => {
  const pending = () => new Promise<never>(() => {});
  return {
    shipmentBoard: {
      briefing: (timezone: string) => ({ queryKey: ["briefing", timezone], queryFn: pending }),
      capacity: (kind: string) => ({ queryKey: ["capacity", kind], queryFn: pending }),
    },
  };
});

const CAPABILITIES: ShipmentBoardCapabilities = {
  ai: true,
  operationType: "both",
  hos: false,
  maps: false,
};

function renderOnBoard(ui: ReactElement, { capabilities = CAPABILITIES, searchParams = "" } = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams}>
          <BoardCapabilitiesProvider value={capabilities}>{ui}</BoardCapabilitiesProvider>
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(cleanup);

describe("shipment board loading states", () => {
  it("draws the brief's outline while the day's brief is read", () => {
    renderOnBoard(<ShipmentBriefing />);

    expect(screen.getByLabelText("Loading the brief")).toHaveAttribute("aria-busy", "true");
  });

  // Without an AI provider there is no brief to wait for, so its outline would
  // be a promise the board never keeps.
  it("draws no brief outline for a workspace without an AI provider", () => {
    const { container } = renderOnBoard(<ShipmentBriefingSkeleton />, {
      capabilities: { ...CAPABILITIES, ai: false },
    });

    expect(container).toBeEmptyDOMElement();
  });

  it("marks the capacity strip busy and keeps its provider switch while capacity is read", () => {
    renderOnBoard(<CapacityStrip />);

    const strip = screen.getByRole("region", { name: "Capacity" });
    expect(strip).toHaveAttribute("aria-busy", "true");
    expect(screen.getByRole("tab", { name: /Drivers/ })).toBeInTheDocument();
  });

  it("opens the panel's outline on the tab the address names", () => {
    renderOnBoard(<ShipmentSidePanelSkeleton />, { searchParams: "?tab=activity" });
    expect(screen.getByLabelText("Loading activity")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading suggested actions")).not.toBeInTheDocument();
    cleanup();

    renderOnBoard(<ShipmentSidePanelSkeleton />);
    expect(screen.getByLabelText("Loading suggested actions")).toBeInTheDocument();
    expect(screen.getByLabelText("Loading the watchlist")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading activity")).not.toBeInTheDocument();
  });
});
