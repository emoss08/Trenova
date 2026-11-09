import type { AuditReview } from "@/lib/graphql/audit-overview";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AuditReviewCard } from "../audit-review-card";

const mocks = vi.hoisted(() => ({
  mark: vi.fn(),
  reviews: vi.fn(),
}));

vi.mock("@/lib/graphql/audit-overview", () => ({
  markAuditReviewReviewed: mocks.mark,
  reopenAuditReview: vi.fn(),
}));

vi.mock("@/lib/queries/audit", () => ({
  audit: {
    reviews: (limit: number) => ({ queryKey: ["audit", "reviews", limit], queryFn: mocks.reviews }),
    overview: { _def: ["audit", "overview"] },
    entryReview: { _def: ["audit", "entry-review"] },
  },
}));

vi.mock("@trenova/shared/hooks/use-now", () => ({ useNow: () => 1_800_000_000 }));

/** The AuditReview shape of services/tms/internal/api/graphql/schema/audit_log.graphqls. */
function review(overrides: Partial<AuditReview> = {}): AuditReview {
  return {
    id: "arv_1",
    kind: "BulkDelete",
    entryId: "ae_1",
    action: "accessorial_rate.delete",
    resource: "accessorial_rate",
    resourceId: null,
    entityRef: null,
    principalId: "usr_1",
    principalType: "session_user",
    actorName: "Dana Whitfield",
    count: 50,
    threshold: 50,
    windowMinutes: 10,
    occurredAt: 1_800_000_000 - 2 * 86_400,
    offHours: true,
    title: "Dana Whitfield deleted 50 records within 10 minutes",
    reviewedAt: null,
    reviewedBy: null,
    version: 0,
    ...overrides,
  };
}

function renderCard(reviews: AuditReview[], openCount = reviews.length) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onOpenReview = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <AuditReviewCard reviews={reviews} openCount={openCount} onOpenReview={onOpenReview} />
    </QueryClientProvider>,
  );
  return { onOpenReview };
}

describe("AuditReviewCard", () => {
  it("shows each open review with when it happened, off hours and its rule", async () => {
    mocks.reviews.mockResolvedValue({ open: [], openCount: 0, reviewed: [], reviewedCount: 0 });
    renderCard([review()]);

    expect(
      screen.getByText("Dana Whitfield deleted 50 records within 10 minutes"),
    ).toBeInTheDocument();
    expect(screen.getByText("2 days ago · off hours · Many deletes")).toBeInTheDocument();
    expect(screen.getByText("1 open")).toBeInTheDocument();
    expect(await screen.findByText("None reviewed this week")).toBeInTheDocument();
  });

  it("opens the entry behind a review", () => {
    mocks.reviews.mockResolvedValue({ open: [], openCount: 0, reviewed: [], reviewedCount: 0 });
    const { onOpenReview } = renderCard([review()]);

    fireEvent.click(screen.getByText("Dana Whitfield deleted 50 records within 10 minutes"));

    expect(onOpenReview).toHaveBeenCalledWith(expect.objectContaining({ entryId: "ae_1" }));
  });

  it("holds Mark reviewed while the server answers and sends it once", async () => {
    mocks.reviews.mockResolvedValue({ open: [], openCount: 0, reviewed: [], reviewedCount: 0 });
    let settle: (value: AuditReview) => void = () => undefined;
    mocks.mark.mockReturnValue(
      new Promise<AuditReview>((resolve) => {
        settle = resolve;
      }),
    );
    renderCard([review()]);

    const button = screen.getByRole("button", { name: "Mark reviewed" });
    fireEvent.click(button);
    fireEvent.click(button);

    await vi.waitFor(() => expect(button).toBeDisabled());
    expect(mocks.mark).toHaveBeenCalledTimes(1);
    expect(mocks.mark).toHaveBeenCalledWith("arv_1");
    settle(review({ reviewedAt: 1_800_000_000 }));
  });

  it("says what the rules watch when nothing is open, and who reviewed last", async () => {
    mocks.reviews.mockResolvedValue({
      open: [],
      openCount: 0,
      reviewed: [
        review({ reviewedAt: 1_800_000_000, reviewedBy: { id: "usr_2", name: "Sam Patel" } }),
      ],
      reviewedCount: 4,
    });
    renderCard([]);

    expect(screen.getByText("Nothing needs a review")).toBeInTheDocument();
    expect(screen.getByText("0 open")).toBeInTheDocument();
    expect(
      await screen.findByText("4 reviewed this week, latest by Sam Patel"),
    ).toBeInTheDocument();
  });
});
