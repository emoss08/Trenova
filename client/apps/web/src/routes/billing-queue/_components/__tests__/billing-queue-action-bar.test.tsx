import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueActionBar } from "../billing-queue-action-bar";

const readiness = vi.hoisted(() => ({
  current: { canMarkReadyToInvoice: true, missingRequirements: [], validationFailures: [] } as {
    canMarkReadyToInvoice: boolean;
    missingRequirements: unknown[];
    validationFailures: { message: string }[];
  },
}));

vi.mock("@/lib/queries/shipment", () => ({
  shipment: {
    billingReadiness: (shipmentId: string) => ({
      queryKey: ["billing-readiness", shipmentId],
      queryFn: async () => readiness.current,
    }),
  },
}));

vi.mock("@/lib/graphql/billing-queue", () => ({
  assignBillingQueueBillerGraphQL: vi.fn(),
  updateBillingQueueStatusGraphQL: vi.fn(),
}));

afterEach(() => {
  cleanup();
  readiness.current = {
    canMarkReadyToInvoice: true,
    missingRequirements: [],
    validationFailures: [],
  };
});

function inReview(blocker: string, needsCount = blocker === "" ? 0 : 1): BillingQueueItem {
  return {
    id: "bqi_1",
    shipmentId: "shp_1",
    status: "InReview",
    detentionHolds: [],
    ratingReviewHold: null,
    review: {
      checks: [],
      issues: [],
      needsCount,
      ready: blocker === "",
      blocker,
      documents: [],
      billerName: "Dana Whitfield",
    },
  } as unknown as BillingQueueItem;
}

function renderBar(item: BillingQueueItem) {
  render(
    <QueryClientProvider client={new QueryClient()}>
      <BillingQueueActionBar item={item} onAssignBiller={() => {}} />
    </QueryClientProvider>,
  );
}

describe("BillingQueueActionBar", () => {
  it("keeps Approve off while a check is flagged", () => {
    renderBar(inReview("issue"));
    expect(screen.getByRole("button", { name: "Approve" })).toBeDisabled();
  });

  it("offers Approve once every check is clear", async () => {
    renderBar(inReview(""));
    expect(await screen.findByRole("button", { name: "Approve" })).toBeEnabled();
  });

  it("says why Approve is closed and sends the biller to the review items", async () => {
    const user = userEvent.setup();
    const showReview = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <BillingQueueActionBar
          item={inReview("issue", 2)}
          onAssignBiller={() => {}}
          onShowReview={showReview}
        />
      </QueryClientProvider>,
    );

    expect(screen.getByRole("status")).toHaveTextContent(
      "2 review items need you before you can approve",
    );
    await user.click(screen.getByRole("button", { name: "Show me" }));

    expect(showReview).toHaveBeenCalledTimes(1);
  });

  it("names a detention charge waiting on an approver before the review", () => {
    renderBar({
      ...inReview("issue"),
      detentionHolds: [{ occurrenceId: "dto_1" }],
    } as unknown as BillingQueueItem);

    expect(screen.getByRole("status")).toHaveTextContent(
      "1 detention charge needs approval before you can approve",
    );
    expect(screen.queryByRole("button", { name: "Show me" })).not.toBeInTheDocument();
  });

  it("names the missing paperwork when the shipment's requirements close Approve", async () => {
    readiness.current = {
      canMarkReadyToInvoice: false,
      missingRequirements: [{}, {}],
      validationFailures: [],
    };
    renderBar(inReview(""));

    expect(await screen.findByRole("status")).toHaveTextContent(
      "2 required documents are missing; add them on the Documents tab",
    );
    expect(screen.getByRole("button", { name: "Approve" })).toBeDisabled();
  });

  it("shows the shortcuts, not a reason, once nothing stands in the way", async () => {
    renderBar(inReview(""));

    expect(await screen.findByRole("button", { name: "Approve" })).toBeEnabled();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.getByText("approve")).toBeInTheDocument();
  });
});
