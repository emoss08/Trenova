import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerOverviewTab from "../worker-overview-tab";

const { fetchWorkerOverview } = vi.hoisted(() => ({ fetchWorkerOverview: vi.fn() }));

vi.mock("@/lib/graphql/worker-overview", () => ({
  fetchWorkerOverview,
  WORKER_OVERVIEW_KEY: "worker-overview",
}));

const baseOverview = {
  asOf: 1_800_000_000,
  standing: "Good",
  concerns: [],
  worker: {
    id: "wrk_1",
    firstName: "Dana",
    lastName: "Reyes",
    status: "Active",
    canBeAssigned: true,
    type: "Employee",
    driverType: "Local",
    fleetCode: { id: "fc_1", code: "SOUTH", color: "#2563eb" },
    profile: {
      hireDate: 1_600_000_000,
      terminationDate: null,
      complianceStatus: "Compliant",
      isQualified: true,
    },
  },
  credentials: {
    complianceStatus: "Compliant",
    requiredCount: 4,
    validCount: 4,
    expiringCount: 0,
    expiredCount: 0,
    missingCount: 0,
  },
  training: {
    compliant: true,
    requiredCount: 3,
    currentCount: 3,
    dueCount: 0,
    overdueCount: 0,
    expiringCount: 0,
    expiredCount: 0,
    missingCount: 0,
  },
  safety: {
    score: 95,
    rating: "Excellent",
    activePoints: 0,
    pointsWatchThreshold: 6,
    pointsAtRiskThreshold: 10,
    accidents: 0,
    preventableAccidents: 0,
    citations: 0,
    outOfServiceOrders: 0,
    openEvents: 0,
    activeDiscipline: 0,
    highestDiscipline: null,
    recognitions: 2,
    daysSinceLastEvent: null,
  },
  checklist: null,
  pto: [
    {
      ptoType: "Vacation",
      tracked: true,
      balanceDays: "12.00",
      pendingDays: "0.00",
      availableDays: "12.00",
    },
  ],
  openReview: null,
  lastReview: null,
  nextReviewAt: null,
};

function renderTab(overview: unknown = baseOverview, onOpenTab = vi.fn()) {
  fetchWorkerOverview.mockResolvedValue(overview);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <WorkerOverviewTab workerId="wrk_1" onOpenTab={onOpenTab} />
    </QueryClientProvider>,
  );
  return onOpenTab;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerOverviewTab", () => {
  it("leads with the standing and what it means", async () => {
    renderTab();

    expect(await screen.findByTestId("worker-standing")).toHaveTextContent("Good standing");
    expect(screen.getByText("Nothing on the record needs attention.")).toBeInTheDocument();
  });

  it("says plainly when nothing needs attention", async () => {
    renderTab();

    expect(await screen.findByTestId("overview-concerns")).toHaveTextContent(
      "Nothing needs attention right now",
    );
  });

  // The whole point of the overview is that the worst thing is read first, so
  // the server's ranking has to survive into the markup.
  it("lists concerns worst first with the reason behind each", async () => {
    renderTab({
      ...baseOverview,
      standing: "AtRisk",
      concerns: [
        {
          severity: "Critical",
          code: "credentials_expired",
          headline: "2 credentials have expired",
          detail: "The worker is not qualified to drive until these are renewed.",
          tab: "credentials",
        },
        {
          severity: "Warning",
          code: "training_due",
          headline: "3 courses need attention soon",
          detail: "Due or expiring inside the reminder window.",
          tab: "training",
        },
        {
          severity: "Info",
          code: "open_safety_events",
          headline: "1 safety event is still open",
          detail: "They stay open until somebody records the outcome.",
          tab: "safety",
        },
      ],
    });

    const list = await screen.findByTestId("overview-concerns");
    const items = within(list).getAllByTestId(/^concern-/);
    expect(items).toHaveLength(3);
    expect(items[0]).toHaveTextContent("2 credentials have expired");
    expect(items[0]).toHaveTextContent("not qualified to drive");
    expect(items[1]).toHaveTextContent("3 courses need attention soon");
    expect(items[2]).toHaveTextContent("1 safety event is still open");
  });

  it("sends the reader to the tab that fixes the concern", async () => {
    const onOpenTab = renderTab({
      ...baseOverview,
      standing: "AtRisk",
      concerns: [
        {
          severity: "Critical",
          code: "training_missing",
          headline: "1 required course was never assigned",
          detail: "These are required for the worker's role and nothing is open.",
          tab: "training",
        },
      ],
    });

    fireEvent.click(await screen.findByTestId("concern-training_missing"));
    await waitFor(() => expect(onOpenTab).toHaveBeenCalledWith("training"));
  });

  it("summarises each part of the record with its headline numbers", async () => {
    renderTab({
      ...baseOverview,
      credentials: { ...baseOverview.credentials, validCount: 3, expiringCount: 1 },
      training: { ...baseOverview.training, currentCount: 2, overdueCount: 1 },
    });

    expect(await screen.findByTestId("overview-card-credentials")).toHaveTextContent(
      "3 of 4 valid",
    );
    expect(screen.getByTestId("overview-card-training")).toHaveTextContent("2 of 3 current");
    expect(screen.getByTestId("overview-card-safety")).toHaveTextContent("95");
    expect(screen.getByTestId("overview-card-pto")).toHaveTextContent("12");
  });

  it("opens the matching tab from a summary card", async () => {
    const onOpenTab = renderTab();

    fireEvent.click(await screen.findByTestId("overview-card-safety"));
    await waitFor(() => expect(onOpenTab).toHaveBeenCalledWith("safety"));
  });

  // A section the viewer cannot read comes back null. Drawing it as a clean
  // card would tell them the safety record is spotless when they simply are
  // not allowed to see it.
  it("omits a section the viewer cannot read rather than drawing it as clean", async () => {
    renderTab({ ...baseOverview, safety: null, pto: null });

    expect(await screen.findByTestId("overview-card-credentials")).toBeInTheDocument();
    expect(screen.queryByTestId("overview-card-safety")).not.toBeInTheDocument();
    expect(screen.queryByTestId("overview-card-pto")).not.toBeInTheDocument();
  });

  it("shows an open checklist with its progress", async () => {
    renderTab({
      ...baseOverview,
      checklist: {
        id: "wcl_1",
        name: "Driver onboarding",
        kind: "Onboarding",
        status: "Open",
        dueAt: null,
        progress: {
          total: 8,
          settled: 6,
          requiredTotal: 5,
          requiredDone: 4,
          overdue: 0,
          percent: 75,
          complete: false,
        },
      },
    });

    const card = await screen.findByTestId("overview-card-checklist");
    expect(card).toHaveTextContent("Driver onboarding");
    expect(card).toHaveTextContent("75%");
  });

  it("reports the last review and when the next one falls due", async () => {
    renderTab({
      ...baseOverview,
      lastReview: {
        id: "pr_1",
        title: "2025 annual",
        status: "Closed",
        periodEnd: 1_700_000_000,
        overallScore: "4.20",
      },
      nextReviewAt: 1_900_000_000,
    });

    const card = await screen.findByTestId("overview-card-reviews");
    expect(card).toHaveTextContent("4.2");
    expect(card).toHaveTextContent("2025 annual");
  });
});
