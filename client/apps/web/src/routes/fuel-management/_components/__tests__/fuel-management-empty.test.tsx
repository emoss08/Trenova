import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import FuelDashboard from "../fuel-dashboard";
import IndexSection from "../index-section";
import ProgramSection from "../program-section";

const mocks = vi.hoisted(() => ({
  fetchFuelDashboard: vi.fn(),
  fetchFuelProgramCurrentRates: vi.fn(),
}));

vi.mock("@/lib/graphql/fuel-surcharge", () => ({
  fetchFuelDashboard: mocks.fetchFuelDashboard,
  fetchFuelProgramCurrentRates: mocks.fetchFuelProgramCurrentRates,
  fetchFuelPriceHistory: vi.fn(),
  fetchFuelSurchargeProgramDetail: vi.fn(),
  fetchEIASeriesOptions: vi.fn(),
  generateFuelSurchargeTable: vi.fn(),
  deleteFuelSurchargeProgram: vi.fn(),
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));
vi.mock("../program-panel", () => ({
  ProgramPanel: ({ open, programId }: { open: boolean; programId: string | null }) =>
    open ? <div data-testid="program-panel" data-program={programId ?? ""} /> : null,
}));
vi.mock("../index-panel", () => ({
  IndexPanel: ({ open }: { open: boolean }) => (open ? <div data-testid="index-panel" /> : null),
}));
vi.mock("../price-history-drawer", () => ({ PriceHistoryDrawer: () => null }));

function renderWithin(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("fuel management empty states", () => {
  // The way in is the Fuel Indices tab on the same page; the button takes
  // the reader there rather than telling them to find it.
  it("says how to get an index on the dashboard and opens the indices tab", async () => {
    mocks.fetchFuelDashboard.mockResolvedValue([]);
    const onOpenIndices = vi.fn();
    const user = userEvent.setup();
    renderWithin(<FuelDashboard onOpenIndices={onOpenIndices} />);

    expect(await screen.findByText("No fuel indices yet")).toBeInTheDocument();
    expect(screen.getByText(/EIA Fuel Prices/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Open fuel indices" }));
    expect(onOpenIndices).toHaveBeenCalledTimes(1);
  });

  it("offers the first program from the programs tab", async () => {
    mocks.fetchFuelProgramCurrentRates.mockResolvedValue([]);
    const user = userEvent.setup();
    renderWithin(<ProgramSection />);

    expect(await screen.findByText("No programs yet")).toBeInTheDocument();
    expect(screen.getByText(/billing profile/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Create a program" }));

    expect(screen.getByTestId("program-panel")).toHaveAttribute("data-program", "");
  });

  it("offers a custom index from the indices tab", async () => {
    mocks.fetchFuelDashboard.mockResolvedValue([]);
    const user = userEvent.setup();
    renderWithin(<IndexSection />);

    expect(await screen.findByText("No indices yet")).toBeInTheDocument();
    expect(screen.getByText(/DOE diesel series/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add a custom index" }));

    expect(screen.getByTestId("index-panel")).toBeInTheDocument();
  });
});
