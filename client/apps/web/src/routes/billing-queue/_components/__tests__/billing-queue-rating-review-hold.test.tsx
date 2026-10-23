import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { RatingReviewHold } from "@trenova/shared/types/billing-queue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueRatingReviewHold } from "../billing-queue-rating-review-hold";

const mocks = vi.hoisted(() => ({ approve: vi.fn() }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/lib/graphql/shipment", () => ({
  approveShipmentRatingReviewGraphQL: mocks.approve,
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderHold(hold: RatingReviewHold | null) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  const invalidate = vi.spyOn(client, "invalidateQueries");
  render(
    <QueryClientProvider client={client}>
      <BillingQueueRatingReviewHold hold={hold} />
    </QueryClientProvider>,
  );
  return { invalidate };
}

const hold: RatingReviewHold = {
  shipmentId: "shp_1",
  reasons: ["Holiday pickup"],
  ratedAt: 1_800_000_000,
  formulaTemplateName: "Dock rates",
};

describe("BillingQueueRatingReviewHold", () => {
  it("shows nothing when the tariff holds nothing on the item", () => {
    renderHold(null);
    expect(screen.queryByTestId("billing-queue-rating-review-hold")).not.toBeInTheDocument();

    renderHold({ ...hold, reasons: [] });
    expect(screen.queryByTestId("billing-queue-rating-review-hold")).not.toBeInTheDocument();
  });

  it("shows the review the tariff asked for, with the action that approves it", () => {
    renderHold(hold);

    expect(
      screen.getByText("The tariff asks for 1 review before this shipment bills"),
    ).toBeInTheDocument();
    expect(screen.getByRole("list", { name: "Reviews" })).toHaveTextContent("Holiday pickup");
    expect(screen.getByRole("button", { name: "Approve rating" })).toBeEnabled();
  });

  it("approves the shipment's rating and refreshes the queue and the shipment", async () => {
    mocks.approve.mockResolvedValue({ id: "shp_1", version: 2 });
    const { invalidate } = renderHold(hold);

    await userEvent.click(screen.getByRole("button", { name: "Approve rating" }));

    await waitFor(() => expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billingQueue"] }));
    expect(mocks.approve).toHaveBeenCalledWith("shp_1");
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["billing-queue-list"] });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["shipment"] });
  });
});
