import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerReviewsTab from "../../worker-reviews-tab";

const {
  fetchWorkerPerformanceReviews,
  fetchActivePerformanceReviewTemplates,
  submitPerformanceReview,
  closePerformanceReview,
  toast,
} = vi.hoisted(() => ({
  fetchWorkerPerformanceReviews: vi.fn(),
  fetchActivePerformanceReviewTemplates: vi.fn(),
  submitPerformanceReview: vi.fn(),
  closePerformanceReview: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

vi.mock("@/lib/graphql/performance-review", () => ({
  fetchWorkerPerformanceReviews,
  fetchActivePerformanceReviewTemplates,
  submitPerformanceReview,
  closePerformanceReview,
  WORKER_REVIEWS_KEY: "worker-reviews",
  REVIEW_TEMPLATES_KEY: "review-templates",
}));

vi.mock("sonner", () => ({ toast }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../review-editor-dialog", () => ({
  ReviewEditorDialog: ({ open, review }: { open: boolean; review?: { id: string } | null }) =>
    open ? <div data-testid="review-editor">{review?.id ?? "new"}</div> : null,
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

function renderTab() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <WorkerReviewsTab workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerReviewsTab", () => {
  it("shows the open review with its score and what it is waiting on", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([
      review("prev_open"),
      review("prev_closed", {
        status: "Closed",
        submittedAt: 1_796_000_000,
        acknowledgedAt: 1_796_500_000,
        closedAt: 1_797_000_000,
        nextReviewAt: 1_830_000_000,
      }),
    ]);
    fetchActivePerformanceReviewTemplates.mockResolvedValue([template]);
    renderTab();

    const open = within(await screen.findByTestId("review-prev_open"));
    expect(open.getByText("4.25")).toBeInTheDocument();
    expect(open.getByText("Draft")).toBeInTheDocument();
    expect(open.getByText(/Waiting on you to submit/)).toBeInTheDocument();
    expect(open.getByRole("button", { name: "Submit" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /History \(1\)/ }));
    const closed = within(screen.getByTestId("review-prev_closed"));
    expect(closed.getByText("Closed")).toBeInTheDocument();
  });

  it("submits a draft and reports what happens next", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([review("prev_open")]);
    fetchActivePerformanceReviewTemplates.mockResolvedValue([template]);
    submitPerformanceReview.mockResolvedValue(
      review("prev_open", { status: "Submitted", submittedAt: 1_800_000_000 }),
    );
    renderTab();

    fireEvent.click(await screen.findByRole("button", { name: "Submit" }));
    await waitFor(() =>
      expect(submitPerformanceReview).toHaveBeenCalledWith({ id: "prev_open", version: 1 }),
    );
    expect(toast.success).toHaveBeenCalledWith(
      "Review submitted",
      expect.objectContaining({ description: expect.stringContaining("sign off") }),
    );
  });

  it("offers Close once the worker has signed and shows their comment", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([
      review("prev_acked", {
        status: "Acknowledged",
        submittedAt: 1_796_000_000,
        acknowledgedAt: 1_796_500_000,
        workerComment: "Fair review.",
      }),
    ]);
    fetchActivePerformanceReviewTemplates.mockResolvedValue([template]);
    closePerformanceReview.mockResolvedValue(review("prev_acked", { status: "Closed" }));
    renderTab();

    const card = within(await screen.findByTestId("review-prev_acked"));
    expect(card.getByText(/Fair review\./)).toBeInTheDocument();
    expect(card.queryByRole("button", { name: "Submit" })).not.toBeInTheDocument();
    fireEvent.click(card.getByRole("button", { name: "Close review" }));
    await waitFor(() =>
      expect(closePerformanceReview).toHaveBeenCalledWith({ id: "prev_acked", version: 1 }),
    );
  });

  it("starts a review when none is open", async () => {
    fetchWorkerPerformanceReviews.mockResolvedValue([]);
    fetchActivePerformanceReviewTemplates.mockResolvedValue([template]);
    renderTab();

    expect(await screen.findByText("No reviews yet")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start a review" }));
    expect(screen.getByTestId("review-editor")).toHaveTextContent("new");
  });
});
