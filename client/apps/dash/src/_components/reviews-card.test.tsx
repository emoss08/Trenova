import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReviewsCard } from "./reviews-card";

const { fetchMyReviews, acknowledgeMyReview, toast } = vi.hoisted(() => ({
  fetchMyReviews: vi.fn(),
  acknowledgeMyReview: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchMyReviews,
  acknowledgeMyReview,
}));

vi.mock("sonner", () => ({ toast }));

function review(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    title: "Annual review — 2025",
    status: "Submitted",
    periodStart: 1_760_000_000,
    periodEnd: 1_795_000_000,
    overallScore: "4.25",
    summary: "Dependable year on the road.",
    strengths: "Clean inspections.",
    improvements: "Receipts same day.",
    ratings: [
      { key: "safety", label: "Safe driving", weight: 3, score: 5, comment: "Spotless" },
      { key: "service", label: "Customer service", weight: 1, score: 3, comment: null },
    ],
    goals: [{ id: "goal_1", title: "Receipts same day", dueAt: null, status: "Open" }],
    submittedAt: 1_796_000_000,
    acknowledgedAt: null,
    workerComment: null,
    closedAt: null,
    reviewer: { id: "usr_1", name: "Dana" },
    ...overrides,
  };
}

function renderCard() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <ReviewsCard />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ReviewsCard", () => {
  it("shows a review waiting for sign-off with its scores and goals", async () => {
    fetchMyReviews.mockResolvedValue([review("prev_1")]);
    renderCard();

    const card = within(await screen.findByTestId("dash-review-prev_1"));
    expect(card.getByText("4.25")).toBeInTheDocument();
    expect(card.getByText("Dependable year on the road.")).toBeInTheDocument();
    expect(card.getByText("Safe driving")).toBeInTheDocument();
    expect(card.getByText("Receipts same day")).toBeInTheDocument();
    expect(card.getByText(/Dana/)).toBeInTheDocument();
    expect(card.getByRole("button", { name: "Sign off" })).toBeInTheDocument();
  });

  it("signs off with an optional comment", async () => {
    fetchMyReviews.mockResolvedValue([review("prev_1")]);
    acknowledgeMyReview.mockResolvedValue(
      review("prev_1", { status: "Acknowledged", acknowledgedAt: 1_800_000_000 }),
    );
    renderCard();

    const card = within(await screen.findByTestId("dash-review-prev_1"));
    fireEvent.change(card.getByLabelText("Your comment"), {
      target: { value: "Fair review." },
    });
    fireEvent.click(card.getByRole("button", { name: "Sign off" }));
    await waitFor(() => expect(acknowledgeMyReview).toHaveBeenCalledWith("prev_1", "Fair review."));
    expect(toast.success).toHaveBeenCalled();
  });

  it("stops asking once signed and says nothing when there are no reviews", async () => {
    fetchMyReviews.mockResolvedValue([
      review("prev_1", {
        status: "Closed",
        acknowledgedAt: 1_797_000_000,
        workerComment: "Agreed.",
        closedAt: 1_798_000_000,
      }),
    ]);
    renderCard();

    const card = within(await screen.findByTestId("dash-review-prev_1"));
    expect(card.queryByRole("button", { name: "Sign off" })).not.toBeInTheDocument();
    expect(card.getByText(/Agreed\./)).toBeInTheDocument();

    cleanup();
    fetchMyReviews.mockResolvedValue([]);
    renderCard();
    expect(await screen.findByText(/No reviews yet/)).toBeInTheDocument();
  });
});
