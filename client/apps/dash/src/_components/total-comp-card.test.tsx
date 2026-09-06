import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TotalCompCard } from "./total-comp-card";

const { fetchMyTotalCompensation } = vi.hoisted(() => ({
  fetchMyTotalCompensation: vi.fn(),
}));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({ fetchMyTotalCompensation }));

const baseTotal = {
  asOf: 1_800_000_000,
  planYear: 2026,
  grossPayMinor: 9_000_000,
  employerBenefitMinor: 48_000,
  employeeBenefitMinor: 12_000,
  accruedTimeOffDays: "8.50",
  totalCompensationMinor: 9_048_000,
  enrollments: [
    {
      id: "wben_1",
      status: "Active",
      coverageTier: "Employee",
      employeeCostMinor: 12_000,
      employerCostMinor: 48_000,
      benefitPlan: { id: "bplan_1", name: "Medical PPO", planType: "Medical" },
    },
  ],
};

function renderCard(total: unknown = baseTotal) {
  fetchMyTotalCompensation.mockResolvedValue(total);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <TotalCompCard />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("TotalCompCard", () => {
  it("leads with what the job is worth, not just what was paid", async () => {
    renderCard();

    expect(await screen.findByText("$90,480.00")).toBeInTheDocument();
    expect(screen.getByText(/\$90,000\.00 paid/)).toBeInTheDocument();
  });

  // Money the driver paid is not money the job gave them, so it is reported
  // separately and never folded into the headline.
  it("keeps the driver's own contribution out of the total", async () => {
    renderCard();

    expect(
      await screen.findByText(/money you paid, not money the job gave you/),
    ).toBeInTheDocument();
    expect(screen.getByText(/You contribute \$120\.00 a period/)).toBeInTheDocument();
  });

  it("lists the cover behind the figure", async () => {
    renderCard();

    expect(await screen.findByText("Medical PPO")).toBeInTheDocument();
    expect(screen.getByText("Employee only")).toBeInTheDocument();
  });

  // Nothing paid and no cover means there is nothing to state yet — an empty
  // card reads as a package worth nothing.
  it("renders nothing when there is nothing to state", async () => {
    renderCard({
      ...baseTotal,
      grossPayMinor: 0,
      employerBenefitMinor: 0,
      employeeBenefitMinor: 0,
      totalCompensationMinor: 0,
      enrollments: [],
    });

    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(screen.queryByTestId("total-comp-card")).not.toBeInTheDocument();
  });
});
