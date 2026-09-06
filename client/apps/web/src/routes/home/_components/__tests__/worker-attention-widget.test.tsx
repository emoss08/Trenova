import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { HomeWidget } from "@/lib/graphql/home-layout";
import { WorkerAttentionWidget } from "../widgets/worker-attention-widget";

const mocks = vi.hoisted(() => ({
  fetchWorkerRosterAttention: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-overview", () => ({
  WORKER_ROSTER_ATTENTION_KEY: "worker-roster-attention",
  fetchWorkerRosterAttention: mocks.fetchWorkerRosterAttention,
}));

type Attention = {
  activeWorkers: number;
  nonCompliant: number;
  trainingOverdue: number;
  atRisk: number;
  expiringSoon: number;
  reviewsAwaitingSignOff: number;
  ptoLiabilityDays: string;
  leaveCertificationsOutstanding: number;
};

function attention(overrides: Partial<Attention> = {}): Attention {
  return {
    activeWorkers: 40,
    nonCompliant: 0,
    trainingOverdue: 0,
    atRisk: 0,
    expiringSoon: 0,
    reviewsAwaitingSignOff: 0,
    ptoLiabilityDays: "0",
    leaveCertificationsOutstanding: 0,
    ...overrides,
  };
}

function widget(): HomeWidget {
  return {
    id: "wgt_01",
    key: "worker-attention",
    title: "Workforce Attention",
    w: 1,
    h: 1,
    config: {},
  } as unknown as HomeWidget;
}

async function renderWidget() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      </MemoryRouter>
    );
  }
  render(<WorkerAttentionWidget widget={widget()} data={{} as never} />, { wrapper: Wrapper });
  await screen.findByText("40 active workers");
}

describe("WorkerAttentionWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows the reviews waiting on a signature and the PTO the org is carrying", async () => {
    mocks.fetchWorkerRosterAttention.mockResolvedValue(
      attention({ reviewsAwaitingSignOff: 4, ptoLiabilityDays: "312.50" }),
    );

    await renderWidget();

    expect(screen.getByText("Reviews to sign")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
    expect(screen.getByText("PTO liability")).toBeInTheDocument();
    expect(screen.getByText("312.5 days")).toBeInTheDocument();
  });

  // A whole number of days is the common case; padding it to "312.00" makes a
  // headline figure read like an invoice line.
  it("drops an empty fraction from the liability figure", async () => {
    mocks.fetchWorkerRosterAttention.mockResolvedValue(attention({ ptoLiabilityDays: "312.00" }));

    await renderWidget();

    expect(screen.getByText("312 days")).toBeInTheDocument();
  });

  // Past the deadline the employer may deny the leave, so a certification the
  // employee still owes is a queue somebody has to work, not a statistic.
  it("shows the leave certifications the office is still waiting on", async () => {
    mocks.fetchWorkerRosterAttention.mockResolvedValue(
      attention({ leaveCertificationsOutstanding: 3 }),
    );

    await renderWidget();

    expect(screen.getByText("Certifications owed")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  // The all-clear copy speaks only for the four roster lines. Reviews and PTO
  // are standing figures, not exceptions, so they stay on screen regardless.
  it("keeps the standing figures when the roster is all clear", async () => {
    mocks.fetchWorkerRosterAttention.mockResolvedValue(
      attention({ reviewsAwaitingSignOff: 2, ptoLiabilityDays: "18" }),
    );

    await renderWidget();

    expect(
      screen.getByText("Every active worker is compliant, in date and off the watch list."),
    ).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("18 days")).toBeInTheDocument();
  });
});
