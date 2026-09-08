import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub } from "@/test/accounting-page-mocks";
import { DetentionIntelligence } from "../detention-intelligence";
import { DetentionIntelligencePage } from "../../page";

const mocks = vi.hoisted(() => ({
  facilities: vi.fn(),
  customers: vi.fn(),
  waivers: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    detentionAnalyticsService: {
      facilities: mocks.facilities,
      customers: mocks.customers,
      waivers: mocks.waivers,
    },
  },
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));

const DAY = 86_400;

function renderWithin(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.facilities.mockResolvedValue([]);
  mocks.customers.mockResolvedValue([]);
  mocks.waivers.mockResolvedValue([]);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("detention intelligence empty state", () => {
  it("draws the page it will become when no stop settled detention in the window", async () => {
    const onWiden = vi.fn();
    const user = userEvent.setup();
    renderWithin(<DetentionIntelligence windowValue="90" onWiden={onWiden} />);

    expect(await screen.findByText("No detention in this window")).toBeInTheDocument();
    expect(screen.getByText(/last 90 days/)).toBeInTheDocument();
    expect(screen.getByText(/detention engine/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Look back 180 days" }));
    expect(onWiden).toHaveBeenCalledTimes(1);
  });

  it("has nothing further to offer once the window is already at its widest", async () => {
    renderWithin(<DetentionIntelligence windowValue="180" />);

    expect(await screen.findByText("No detention in this window")).toBeInTheDocument();
    expect(screen.getByText(/last 180 days/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Look back/ })).not.toBeInTheDocument();
  });

  // The window lives in the page header; the sketch's button must move that
  // control, not a copy of it, so the header and the figures agree.
  it("widens the page's own window from the sketch", async () => {
    const user = userEvent.setup();
    renderWithin(<DetentionIntelligencePage />);

    await user.click(await screen.findByRole("button", { name: "Look back 180 days" }));

    await waitFor(() => {
      const spans = mocks.facilities.mock.calls.map(
        ([args]: [{ from: number; to: number }]) => (args.to - args.from) / DAY,
      );
      expect(spans).toContain(180);
    });
  });
});
