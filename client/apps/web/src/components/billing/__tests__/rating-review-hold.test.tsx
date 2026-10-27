import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RatingReviewHold } from "../rating-review-hold";

const mocks = vi.hoisted(() => ({
  canUpdate: true,
  approve: vi.fn(),
  handleError: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canUpdate, isLoading: false }),
}));

vi.mock("@/lib/graphql/shipment", () => ({
  approveShipmentRatingReviewGraphQL: mocks.approve,
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: mocks.handleError, warning: vi.fn(), info: vi.fn() },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.canUpdate = true;
});

function renderHold(props: Partial<React.ComponentProps<typeof RatingReviewHold>> = {}) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RatingReviewHold
        shipmentId="shp_1"
        reasons={["Holiday pickup", "Liftgate billed on a dock delivery"]}
        awaiting
        {...props}
      />
    </QueryClientProvider>,
  );
}

describe("RatingReviewHold", () => {
  it("shows nothing when the tariff asked for no review", () => {
    renderHold({ reasons: [] });

    expect(screen.queryByTestId("rating-review-hold")).not.toBeInTheDocument();
    expect(screen.queryByTestId("rating-review-approved")).not.toBeInTheDocument();
  });

  it("lists every review the tariff asked for while the rating is held", () => {
    renderHold();

    expect(
      screen.getByText("The tariff asks for 2 reviews before this shipment bills"),
    ).toBeInTheDocument();
    const reviews = screen.getByRole("list", { name: "Reviews" });
    expect(reviews).toHaveTextContent("Holiday pickup");
    expect(reviews).toHaveTextContent("Liftgate billed on a dock delivery");
    expect(screen.getByRole("button", { name: "Approve rating" })).toBeEnabled();
  });

  it("approves the rating and hands back the saved shipment", async () => {
    const saved = { id: "shp_1", version: 4 } as Shipment;
    mocks.approve.mockResolvedValue(saved);
    const onApproved = vi.fn();
    renderHold({ onApproved });

    await userEvent.click(screen.getByRole("button", { name: "Approve rating" }));

    await waitFor(() => expect(onApproved).toHaveBeenCalledWith(saved));
    expect(mocks.approve).toHaveBeenCalledWith("shp_1");
  });

  it("keeps the hold and reports the error when the approval fails", async () => {
    const failure = new Error("The rating was already approved");
    mocks.approve.mockRejectedValue(failure);
    const onApproved = vi.fn();
    renderHold({ onApproved });

    await userEvent.click(screen.getByRole("button", { name: "Approve rating" }));

    await waitFor(() => expect(mocks.handleError).toHaveBeenCalled());
    expect(onApproved).not.toHaveBeenCalled();
    expect(screen.getByTestId("rating-review-hold")).toBeInTheDocument();
  });

  it("disables the approve action while the approval is saving", async () => {
    mocks.approve.mockReturnValue(new Promise(() => {}));
    renderHold();

    await userEvent.click(screen.getByRole("button", { name: "Approve rating" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Approving…" })).toBeDisabled());
  });

  it("says who approves it to someone who cannot", () => {
    mocks.canUpdate = false;
    renderHold();

    expect(screen.queryByRole("button", { name: "Approve rating" })).not.toBeInTheDocument();
    expect(
      screen.getByText("Someone who works the billing queue approves it."),
    ).toBeInTheDocument();
  });

  it("says who approved the reviews once they are approved", () => {
    renderHold({
      awaiting: false,
      reasons: ["Holiday pickup"],
      approvedAt: 1_800_000_000,
      approvedByName: "Avery Lane",
    });

    expect(screen.queryByTestId("rating-review-hold")).not.toBeInTheDocument();
    const approved = screen.getByTestId("rating-review-approved");
    expect(approved).toHaveTextContent("Rating reviewed");
    expect(approved).toHaveTextContent(/Approved for billing by Avery Lane on/);
    expect(approved).toHaveTextContent("Holiday pickup");
    expect(screen.queryByRole("button", { name: "Approve rating" })).not.toBeInTheDocument();
  });
});
