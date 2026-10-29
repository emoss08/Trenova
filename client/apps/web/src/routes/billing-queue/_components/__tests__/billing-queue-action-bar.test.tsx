import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueActionBar } from "../billing-queue-action-bar";

vi.mock("@/lib/queries/shipment", () => ({
  shipment: {
    billingReadiness: (shipmentId: string) => ({
      queryKey: ["billing-readiness", shipmentId],
      queryFn: async () => ({ canMarkReadyToInvoice: true, missingRequirements: [] }),
    }),
  },
}));

vi.mock("@/lib/graphql/billing-queue", () => ({
  assignBillingQueueBillerGraphQL: vi.fn(),
  updateBillingQueueStatusGraphQL: vi.fn(),
}));

afterEach(cleanup);

function inReview(blocker: string): BillingQueueItem {
  return {
    id: "bqi_1",
    shipmentId: "shp_1",
    status: "InReview",
    detentionHolds: [],
    ratingReviewHold: null,
    review: {
      checks: [],
      issues: [],
      needsCount: blocker === "" ? 0 : 1,
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
});
