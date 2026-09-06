import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TrainingCard } from "./training-card";

const { fetchMyTraining, startMyTraining, acknowledgeMyTraining, toast } = vi.hoisted(() => ({
  fetchMyTraining: vi.fn(),
  startMyTraining: vi.fn(),
  acknowledgeMyTraining: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchMyTraining,
  startMyTraining,
  acknowledgeMyTraining,
}));

vi.mock("sonner", () => ({ toast }));

const rows = [
  {
    id: "wtrn_defensive",
    courseId: "trnc_defensive",
    name: "Defensive Driving",
    description: "Smith System refresher.",
    category: "Safety",
    delivery: "Online",
    contentUrl: "https://training.example.com/defensive",
    durationMinutes: 90,
    status: "Assigned",
    health: "DueSoon",
    required: true,
    requiresAcknowledgement: true,
    scored: false,
    dueAt: 1_800_300_000,
    daysUntilDue: 3,
    startedAt: null,
    completedAt: null,
    expiresAt: null,
    daysUntilExpiry: null,
    acknowledgedAt: null,
    score: null,
  },
  {
    id: "wtrn_hazmat",
    courseId: "trnc_hazmat",
    name: "Hazmat Awareness",
    description: null,
    category: "HazardousMaterials",
    delivery: "Classroom",
    contentUrl: null,
    durationMinutes: 180,
    status: "InProgress",
    health: "Overdue",
    required: true,
    requiresAcknowledgement: true,
    scored: true,
    dueAt: 1_799_000_000,
    daysUntilDue: -4,
    startedAt: 1_798_000_000,
    completedAt: null,
    expiresAt: null,
    daysUntilExpiry: null,
    acknowledgedAt: 1_798_500_000,
    score: null,
  },
  {
    id: "wtrn_orient",
    courseId: "trnc_orient",
    name: "New Driver Orientation",
    description: null,
    category: "Orientation",
    delivery: "Classroom",
    contentUrl: null,
    durationMinutes: 240,
    status: "Completed",
    health: "Current",
    required: true,
    requiresAcknowledgement: true,
    scored: false,
    dueAt: null,
    daysUntilDue: null,
    startedAt: 1_700_000_000,
    completedAt: 1_700_000_000,
    expiresAt: null,
    daysUntilExpiry: null,
    acknowledgedAt: 1_700_000_000,
    score: null,
  },
  {
    id: null,
    courseId: "trnc_hos",
    name: "Hours of Service Refresher",
    description: null,
    category: "Compliance",
    delivery: "Document",
    contentUrl: "https://training.example.com/hos.pdf",
    durationMinutes: 30,
    status: null,
    health: "Missing",
    required: true,
    requiresAcknowledgement: true,
    scored: false,
    dueAt: null,
    daysUntilDue: null,
    startedAt: null,
    completedAt: null,
    expiresAt: null,
    daysUntilExpiry: null,
    acknowledgedAt: null,
    score: null,
  },
];

function renderCard() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <TrainingCard />
    </QueryClientProvider>,
  );
  return queryClient;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("TrainingCard", () => {
  it("lists courses worst-first with what to do next", async () => {
    fetchMyTraining.mockResolvedValue(rows);
    renderCard();

    const items = await screen.findAllByTestId(/^dash-training-/);
    expect(items.map((item) => item.getAttribute("data-testid"))).toEqual([
      "dash-training-trnc_hos",
      "dash-training-trnc_hazmat",
      "dash-training-trnc_defensive",
      "dash-training-trnc_orient",
    ]);
    expect(screen.getByText("2 need attention")).toBeInTheDocument();

    const defensive = within(screen.getByTestId("dash-training-trnc_defensive"));
    expect(defensive.getByText("Due in 3 days")).toBeInTheDocument();
    expect(defensive.getByRole("link", { name: /Open course/ })).toHaveAttribute(
      "href",
      "https://training.example.com/defensive",
    );

    const hazmat = within(screen.getByTestId("dash-training-trnc_hazmat"));
    expect(hazmat.getByText("Overdue by 4 days")).toBeInTheDocument();
    expect(hazmat.getByText(/Waiting for your result/)).toBeInTheDocument();
    expect(hazmat.queryByRole("button", { name: /I've completed/ })).not.toBeInTheDocument();

    const hos = within(screen.getByTestId("dash-training-trnc_hos"));
    expect(hos.getByText("Not assigned")).toBeInTheDocument();
    expect(hos.queryByRole("link")).not.toBeInTheDocument();
  });

  it("marks a course started when the link is opened and completes it on acknowledgement", async () => {
    fetchMyTraining.mockResolvedValue(rows);
    startMyTraining.mockResolvedValue({ ...rows[0], status: "InProgress" });
    acknowledgeMyTraining.mockResolvedValue({
      ...rows[0],
      status: "Completed",
      health: "Current",
      acknowledgedAt: 1_800_000_000,
    });
    renderCard();

    const defensive = within(await screen.findByTestId("dash-training-trnc_defensive"));
    const link = defensive.getByRole("link", { name: /Open course/ });
    link.addEventListener("click", (event) => event.preventDefault());
    fireEvent.click(link);
    await waitFor(() => expect(startMyTraining).toHaveBeenCalledWith("wtrn_defensive"));

    fireEvent.click(defensive.getByRole("button", { name: /I've completed Defensive Driving/ }));
    await waitFor(() => expect(acknowledgeMyTraining).toHaveBeenCalledWith("wtrn_defensive"));
    await waitFor(() => expect(fetchMyTraining).toHaveBeenCalledTimes(3));
    expect(toast.success).toHaveBeenCalled();
  });

  it("says when everything is current", async () => {
    fetchMyTraining.mockResolvedValue([rows[2]]);
    renderCard();
    expect(await screen.findByText("All current")).toBeInTheDocument();
  });
});
