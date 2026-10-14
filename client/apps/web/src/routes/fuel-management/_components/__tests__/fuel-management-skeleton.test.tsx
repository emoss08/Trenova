import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import type { ComponentType } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import FuelDashboard from "../fuel-dashboard";
import {
  FuelDashboardSkeleton,
  IndexSectionSkeleton,
  ProgramSectionSkeleton,
} from "../fuel-management-skeleton";
import IndexSection from "../index-section";
import ProgramSection from "../program-section";

const mocks = vi.hoisted(() => ({
  fetchFuelDashboard: vi.fn(),
  fetchFuelProgramCurrentRates: vi.fn(),
}));

vi.mock("@/lib/graphql/fuel-surcharge", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/fuel-surcharge")>()),
  fetchFuelDashboard: mocks.fetchFuelDashboard,
  fetchFuelProgramCurrentRates: mocks.fetchFuelProgramCurrentRates,
}));

vi.mock("../program-panel", () => ({ ProgramPanel: () => null }));
vi.mock("../index-panel", () => ({ IndexPanel: () => null }));
vi.mock("../price-history-drawer", () => ({ PriceHistoryDrawer: () => null }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const hidden = { hidden: true } as const;
const DOE_SERIES = 11;

function renderPending(Component: ComponentType) {
  const pending = new Promise<never>(() => {});
  mocks.fetchFuelDashboard.mockReturnValue(pending);
  mocks.fetchFuelProgramCurrentRates.mockReturnValue(pending);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <Component />
    </QueryClientProvider>,
  );
}

describe("fuel management skeletons", () => {
  it.each([
    ["Loading fuel prices", FuelDashboardSkeleton],
    ["Loading surcharge programs", ProgramSectionSkeleton],
    ["Loading fuel indices", IndexSectionSkeleton],
  ] as const)("%s announces itself once and carries no text", (label, Skeleton) => {
    render(<Skeleton />);

    const root = screen.getByLabelText(label);
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root).toHaveTextContent("");
    expect(screen.queryAllByRole("row")).toHaveLength(0);
  });

  // The index table lists the eleven DOE diesel series the EIA integration
  // provisions; the outline has a row for each under the table's own head.
  it("outlines the index table row for row", () => {
    render(<IndexSectionSkeleton />);

    const table = within(screen.getByLabelText("Loading fuel indices")).getByRole("table", hidden);
    expect(within(table).getAllByRole("row", hidden)).toHaveLength(DOE_SERIES + 1);
  });

  // Each tab draws the same outline while its list is read as it does while
  // its code arrives, so the tab does not change shape between the two.
  it.each([
    ["Loading fuel prices", FuelDashboard],
    ["Loading indices", IndexSection],
    ["Loading programs", ProgramSection],
  ] as const)("the tab reads as %s while its list loads", (label, Component) => {
    renderPending(Component as ComponentType);

    expect(screen.getByLabelText(label)).toBeInTheDocument();
  });

  it("outlines the program cards under the real note while the rates load", () => {
    renderPending(ProgramSection);

    expect(screen.getByRole("button", { name: /New program/ })).toBeInTheDocument();
    expect(screen.getByLabelText("Loading programs")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByText("No programs yet")).toBeNull();
  });
});
