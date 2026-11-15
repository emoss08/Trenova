import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { HomeWidget } from "@/lib/graphql/home-layout";
import { WorkerAttentionWidget } from "../widgets/worker-attention-widget";

const mocks = vi.hoisted(() => ({
  fetchWorkerRosterAttention: vi.fn(),
  planOff: new Set<string>(),
}));

vi.mock("@trenova/shared/hooks/use-plan-feature", () => ({
  usePlanFeature: (feature: string) =>
    mocks.planOff.has(feature)
      ? { available: false, featureName: "Time off", addOnName: "HR Suite" }
      : { available: true, featureName: "", addOnName: "" },
}));

vi.mock("@/lib/graphql/worker-overview", () => ({
  WORKER_ROSTER_ATTENTION_KEY: "worker-roster-attention",
  fetchWorkerRosterAttention: mocks.fetchWorkerRosterAttention,
}));

type Attention = {
  activeWorkers: number;
  nonCompliant: number;
  trainingOverdue: number | null;
  atRisk: number;
  expiringSoon: number;
  reviewsAwaitingSignOff: number | null;
  ptoLiabilityDays: string | null;
  leaveCertificationsOutstanding: number | null;
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
    label: "Workforce attention",
    title: "Workforce Attention",
    w: 1,
    h: 1,
    planLock: null,
    lockedMetrics: [],
    config: {},
  } as unknown as HomeWidget;
}

async function renderWidget() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <TestRouter>
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      </TestRouter>
    );
  }
  render(<WorkerAttentionWidget widget={widget()} data={{} as never} />, { wrapper: Wrapper });
  await screen.findByText("40 active workers");
}

describe("WorkerAttentionWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.planOff.clear();
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

  // Reviews and leave are granted on their own. The server sends null to a
  // reader who may not read them, and a hidden count is not a zero.
  it("leaves out the figures the reader may not see", async () => {
    mocks.fetchWorkerRosterAttention.mockResolvedValue(
      attention({ reviewsAwaitingSignOff: null, leaveCertificationsOutstanding: null }),
    );

    await renderWidget();

    expect(screen.queryByText("Reviews to sign")).not.toBeInTheDocument();
    expect(screen.queryByText("Certifications owed")).not.toBeInTheDocument();
    expect(screen.getByText("PTO liability")).toBeInTheDocument();
  });

  // PTO and training are each their own resource. Without the read the server sends null, and
  // a hidden figure is neither a zero nor a gap that asks for an explanation.
  it("leaves out the PTO liability and the overdue training of a reader without their reads", async () => {
    mocks.fetchWorkerRosterAttention.mockResolvedValue(
      attention({ trainingOverdue: null, ptoLiabilityDays: null }),
    );

    await renderWidget();

    expect(screen.queryByText("PTO liability")).not.toBeInTheDocument();
    expect(screen.queryByText("Training overdue")).not.toBeInTheDocument();
    expect(screen.queryByText(/not on your plan/)).not.toBeInTheDocument();
  });

  it("draws nothing for a worker figure the plan leaves out, not even its add-on's name", async () => {
    mocks.planOff.add("workforce.time_off");
    mocks.planOff.add("workforce.talent");
    mocks.fetchWorkerRosterAttention.mockResolvedValue(
      attention({ trainingOverdue: null, ptoLiabilityDays: null }),
    );

    await renderWidget();

    expect(screen.queryByText("PTO liability")).not.toBeInTheDocument();
    expect(screen.queryByText("Training overdue")).not.toBeInTheDocument();
    expect(screen.queryByText(/not on your plan/)).not.toBeInTheDocument();
    expect(document.querySelector('[data-slot="plan-locked"]')).toBeNull();
  });

  it("leaves out the compliance and safety lines on a plan without Safety & Compliance", async () => {
    mocks.planOff.add("workforce.compliance");
    mocks.planOff.add("workforce.safety");
    mocks.fetchWorkerRosterAttention.mockResolvedValue(attention({ trainingOverdue: 2 }));

    await renderWidget();

    expect(screen.queryByText("Non-compliant")).not.toBeInTheDocument();
    expect(screen.queryByText("Safety at risk")).not.toBeInTheDocument();
    expect(screen.queryByText("Expiring in 30 days")).not.toBeInTheDocument();
    expect(screen.getByText("Training overdue")).toBeInTheDocument();
  });

  it("draws no roster lines and no all-clear sentence when the plan carries none of them", async () => {
    for (const feature of [
      "workforce.compliance",
      "workforce.safety",
      "workforce.talent",
      "workforce.time_off",
    ]) {
      mocks.planOff.add(feature);
    }
    mocks.fetchWorkerRosterAttention.mockResolvedValue(
      attention({
        trainingOverdue: null,
        ptoLiabilityDays: null,
        reviewsAwaitingSignOff: null,
        leaveCertificationsOutstanding: null,
      }),
    );

    await renderWidget();

    expect(
      screen.queryByText("Every active worker is compliant, in date and off the watch list."),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Non-compliant|Safety at risk/ })).toBeNull();
    expect(document.querySelector("dl")).toBeNull();
    expect(screen.getByText("40 active workers")).toBeInTheDocument();
  });
});
