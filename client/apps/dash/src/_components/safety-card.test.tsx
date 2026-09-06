import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SafetyCard } from "./safety-card";

const {
  fetchMySafetyScorecard,
  fetchMyRecognitions,
  fetchMyDisciplinaryActions,
  acknowledgeMyDisciplinaryAction,
  toast,
} = vi.hoisted(() => ({
  fetchMySafetyScorecard: vi.fn(),
  fetchMyRecognitions: vi.fn(),
  fetchMyDisciplinaryActions: vi.fn(),
  acknowledgeMyDisciplinaryAction: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchMySafetyScorecard,
  fetchMyRecognitions,
  fetchMyDisciplinaryActions,
  acknowledgeMyDisciplinaryAction,
}));

vi.mock("sonner", () => ({ toast }));

const scorecard = {
  workerId: "wrk_1",
  asOf: 1_800_000_000,
  score: 82,
  rating: "Good",
  activePoints: 3,
  pointsWatchThreshold: 6,
  pointsAtRiskThreshold: 10,
  accidents: 0,
  preventableAccidents: 0,
  incidents: 0,
  nearMisses: 1,
  citations: 1,
  inspections: 3,
  inspectionsPassed: 3,
  inspectionsFailed: 0,
  outOfServiceOrders: 0,
  cleanInspectionRate: 1,
  openEvents: 0,
  activeDiscipline: 1,
  highestDiscipline: "VerbalWarning",
  daysSinceLastEvent: 45,
  lastEventAt: 1_796_000_000,
  recognitions: 1,
};

const recognitions = [
  {
    id: "wrec_1",
    kind: "SafetyMilestone",
    title: "One year accident-free",
    message: "Thank you for the care you take.",
    occurredAt: 1_799_000_000,
    awardedBy: { id: "usr_1", name: "Dana" },
  },
];

const unacknowledged = {
  id: "wdac_1",
  level: "VerbalWarning",
  status: "Active",
  reason: "Late departure",
  details: null,
  issuedAt: 1_798_000_000,
  expiresAt: 1_830_000_000,
  suspensionDays: null,
  acknowledgedAt: null,
  workerComment: null,
  active: true,
  issuedBy: { id: "usr_1", name: "Dana" },
};

function renderCard() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <SafetyCard />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SafetyCard", () => {
  it("shows the driver the same score the office sees, with their clean record", async () => {
    fetchMySafetyScorecard.mockResolvedValue(scorecard);
    fetchMyRecognitions.mockResolvedValue(recognitions);
    fetchMyDisciplinaryActions.mockResolvedValue([]);
    renderCard();

    expect(await screen.findByText("82")).toBeInTheDocument();
    expect(screen.getByText("Good")).toBeInTheDocument();
    expect(screen.getByText("3 of 3 clean (100%)")).toBeInTheDocument();
    expect(screen.getByText(/45 days/)).toBeInTheDocument();
    expect(screen.getByText("One year accident-free")).toBeInTheDocument();
  });

  it("asks the driver to acknowledge an action and sends their reply", async () => {
    fetchMySafetyScorecard.mockResolvedValue(scorecard);
    fetchMyRecognitions.mockResolvedValue([]);
    fetchMyDisciplinaryActions.mockResolvedValue([unacknowledged]);
    acknowledgeMyDisciplinaryAction.mockResolvedValue({
      ...unacknowledged,
      acknowledgedAt: 1_800_000_000,
      workerComment: "Understood",
    });
    renderCard();

    const row = within(await screen.findByTestId("dash-discipline-wdac_1"));
    expect(row.getByText("Verbal warning")).toBeInTheDocument();
    expect(row.getByText("Late departure")).toBeInTheDocument();

    fireEvent.change(row.getByLabelText("Your response"), { target: { value: "Understood" } });
    fireEvent.click(row.getByRole("button", { name: "I've read this" }));
    await waitFor(() =>
      expect(acknowledgeMyDisciplinaryAction).toHaveBeenCalledWith("wdac_1", "Understood"),
    );
    expect(toast.success).toHaveBeenCalled();
  });

  it("does not ask again once an action is acknowledged", async () => {
    fetchMySafetyScorecard.mockResolvedValue(scorecard);
    fetchMyRecognitions.mockResolvedValue([]);
    fetchMyDisciplinaryActions.mockResolvedValue([
      { ...unacknowledged, acknowledgedAt: 1_799_500_000, workerComment: "Understood" },
    ]);
    renderCard();

    const row = within(await screen.findByTestId("dash-discipline-wdac_1"));
    expect(row.queryByRole("button", { name: "I've read this" })).not.toBeInTheDocument();
    expect(row.getByText(/Understood/)).toBeInTheDocument();
  });
});
