import type { FactoringSummary } from "@/lib/graphql/factoring";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringSummaryStrip } from "../factoring-summary";

const mocks = vi.hoisted(() => ({ fetchFactoringSummary: vi.fn() }));

vi.mock("@/lib/graphql/factoring", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring")>()),
  fetchFactoringSummary: mocks.fetchFactoringSummary,
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const summary: FactoringSummary = {
  ready: { count: 3, amountMinor: 300_000 },
  notReady: { count: 0, amountMinor: 0 },
  submitted: { count: 2, amountMinor: 180_000 },
  readyUnsubmitted: { count: 0, amountMinor: 0 },
  funded: { count: 4, amountMinor: 410_000 },
  reserveHeld: { count: 3, amountMinor: 28_750 },
  chargebacksDue: { count: 1, amountMinor: 97_250 },
};

function renderStrip(onSelect = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <FactoringSummaryStrip active="ready" onSelect={onSelect} />
    </QueryClientProvider>,
  );
  return onSelect;
}

describe("factoring summary strip", () => {
  it("shows what is funded, the reserve the factors hold and the chargebacks coming due", async () => {
    mocks.fetchFactoringSummary.mockResolvedValue(summary);
    const onSelect = renderStrip();

    const strip = await screen.findByRole("group", { name: "Factoring summary" });
    expect(within(strip).getByText("$4,100.00")).toBeInTheDocument();
    expect(within(strip).getByText("4 invoices")).toBeInTheDocument();
    expect(within(strip).getByText("$287.50")).toBeInTheDocument();
    expect(within(strip).getByText("3 invoices awaiting release")).toBeInTheDocument();
    expect(within(strip).getByText("$972.50")).toBeInTheDocument();
    expect(within(strip).getByText("1 invoice within a week")).toBeInTheDocument();

    await userEvent.click(within(strip).getByText("$4,100.00"));
    expect(onSelect).toHaveBeenCalledWith("funded");
  });
});
