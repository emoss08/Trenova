import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerReviewsTab from "../worker-reviews-tab";

const {
  fetchWorkerPerformanceReviews,
  fetchActivePerformanceReviewTemplates,
  submitPerformanceReview,
  closePerformanceReview,
  deletePerformanceReview,
  toast,
} = vi.hoisted(() => ({
  fetchWorkerPerformanceReviews: vi.fn(),
  fetchActivePerformanceReviewTemplates: vi.fn(),
  submitPerformanceReview: vi.fn(),
  closePerformanceReview: vi.fn(),
  deletePerformanceReview: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

vi.mock("@/lib/graphql/performance-review", () => ({
  fetchWorkerPerformanceReviews,
  fetchActivePerformanceReviewTemplates,
  submitPerformanceReview,
  closePerformanceReview,
  deletePerformanceReview,
  reopenPerformanceReview: vi.fn(),
  WORKER_REVIEWS_KEY: "worker-reviews",
  REVIEW_TEMPLATES_KEY: "review-templates",
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({ toast }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

const template = {
  id: "prt_1",
  code: "DRIVER-ANNUAL",
  name: "Annual Driver Review",
  cadenceMonths: 12,
  items: [
    { key: "safety", label: "Safe driving", description: null, weight: 3 },
    { key: "service", label: "Customer service", description: null, weight: 1 },
  ],
};

function review(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    businessUnitId: "bu_1",
    organizationId: "org_1",
    workerId: "wrk_1",
    templateId: "prt_1",
    template,
    reviewerId: "usr_1",
    reviewer: { id: "usr_1", name: "Dana" },
    status: "Draft",
    title: "Annual review — 2025",
    periodStart: 1_760_000_000,
    periodEnd: 1_795_000_000,
    ratings: [
      { key: "safety", label: "Safe driving", weight: 3, score: 4, comment: "Clean record" },
      { key: "service", label: "Customer service", weight: 1, score: 5, comment: null },
    ],
    overallScore: "4.25",
    summary: "Dependable year.",
    strengths: null,
    improvements: null,
    goals: [{ id: "goal_1", title: "Receipts same day", dueAt: null, status: "Open" }],
    submittedAt: null,
    acknowledgedAt: null,
    workerComment: null,
    closedAt: null,
    closedById: null,
    nextReviewAt: null,
    version: 1,
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  };
}

const handlers = { onStart: vi.fn(), onEdit: vi.fn(), onRead: vi.fn() };

function renderTab(focusId: string | null = null) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <WorkerReviewsTab
        workerId="wrk_1"
        workerFirstName="Maria"
        focusId={focusId}
        now={1_800_000_000}
        {...handlers}
      />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const closed = review("prev_closed", {
  title: "2025 annual review",
  status: "Closed",
  submittedAt: 1_796_000_000,
  acknowledgedAt: 1_796_500_000,
  closedAt: 1_797_000_000,
  nextReviewAt: 1_830_000_000,
  overallScore: "4.00",
});

describe("WorkerReviewsTab", () => {
  it("shows the open review with its overall score, each rating in words and what it waits on", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([review("prev_open"), closed]);
    renderTab();

    const open = within(await screen.findByTestId("review-prev_open"));
    expect(open.getByText("4.3")).toBeInTheDocument();
    expect(open.getByText("Draft")).toBeInTheDocument();
    expect(open.getByText(/Waiting on you to submit/)).toBeInTheDocument();
    expect(open.getByRole("img", { name: "Safe driving: 4 of 5, Strong" })).toBeInTheDocument();
    expect(open.getByText("Exceptional")).toBeInTheDocument();
    expect(open.getByRole("button", { name: "Submit to Maria" })).toBeInTheDocument();
    expect(open.getByText("Receipts same day")).toBeInTheDocument();
  });

  it("reads the cadence from the last closed review and lists it in History", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([review("prev_open"), closed]);
    renderTab();
    const user = userEvent.setup();

    const cadence = within(await screen.findByRole("region", { name: "Cadence" }));
    expect(cadence.getByText(/in 347 days/)).toBeInTheDocument();
    expect(cadence.getByText("Annual Driver Review · every 12 months")).toBeInTheDocument();

    await user.click(screen.getByTestId("review-history-prev_closed"));
    expect(handlers.onRead).toHaveBeenCalledWith(closed);
  });

  it("submits a draft once, however often the button is pressed", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([review("prev_open")]);
    submitPerformanceReview.mockResolvedValue(
      review("prev_open", { status: "Submitted", submittedAt: 1_800_000_000 }),
    );
    renderTab();

    const submit = await screen.findByRole("button", { name: "Submit to Maria" });
    fireEvent.click(submit);
    fireEvent.click(submit);
    await waitFor(() =>
      expect(submitPerformanceReview).toHaveBeenCalledWith({ id: "prev_open", version: 1 }),
    );
    expect(submitPerformanceReview).toHaveBeenCalledTimes(1);
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("offers Close once the worker has signed and shows their reply", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([
      review("prev_acked", {
        status: "Acknowledged",
        submittedAt: 1_796_000_000,
        acknowledgedAt: 1_796_500_000,
        workerComment: "Fair review.",
      }),
    ]);
    closePerformanceReview.mockResolvedValue(review("prev_acked", { status: "Closed" }));
    renderTab();

    const card = within(await screen.findByTestId("review-prev_acked"));
    expect(card.getByText(/Fair review\./)).toBeInTheDocument();
    expect(card.queryByRole("button", { name: "Submit to Maria" })).not.toBeInTheDocument();
    fireEvent.click(card.getByRole("button", { name: "Close review" }));
    await waitFor(() =>
      expect(closePerformanceReview).toHaveBeenCalledWith({ id: "prev_acked", version: 1 }),
    );
  });

  it("asks before deleting a draft", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([review("prev_open")]);
    deletePerformanceReview.mockResolvedValue(true);
    renderTab();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Delete" }));
    expect(deletePerformanceReview).not.toHaveBeenCalled();
    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(/Maria never saw it/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Delete draft" }));
    await waitFor(() => expect(deletePerformanceReview).toHaveBeenCalledWith("prev_open"));
  });

  it("starts a review from the empty state when none is open", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([]);
    renderTab();

    expect(await screen.findByText("No review open")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start a review" }));
    expect(handlers.onStart).toHaveBeenCalledTimes(1);
  });

  it("says the read failed instead of claiming there are no reviews", async () => {
    fetchWorkerPerformanceReviews.mockRejectedValue(new Error("boom"));
    renderTab();
    expect(await screen.findByText("Couldn't load the reviews")).toBeInTheDocument();
    expect(screen.queryByText("No review open")).not.toBeInTheDocument();
  });
});
