import { TestRouter } from "@trenova/shared/testing/test-router";
import { recordPath } from "@/config/record-links";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { DetentionHold } from "@trenova/shared/types/billing-queue";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueDetentionHolds } from "../billing-queue-detention-holds";

const service = vi.hoisted(() => ({ approve: vi.fn(), waive: vi.fn(), dispute: vi.fn() }));

vi.mock("@/services/detention", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/detention")>()),
  detentionService: service,
}));

beforeEach(() => seedPermissions({ [Resource.Detention]: Operation.Read }));
afterEach(() => {
  cleanup();
  clearPermissions();
  vi.clearAllMocks();
});

function hold(overrides: Partial<DetentionHold> = {}): DetentionHold {
  return {
    occurrenceId: "dto_1",
    stopId: "stp_1",
    stopType: "Delivery",
    locationName: "Riverside DC",
    clockStartAt: 1_790_000_000,
    billableAmount: 425,
    currency: "USD",
    reason: "OverApprovalThreshold",
    ...overrides,
  };
}

function renderHolds(holds: DetentionHold[]) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidated = vi.spyOn(queryClient, "invalidateQueries");
  const view = render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        <BillingQueueDetentionHolds holds={holds} />
      </TestRouter>
    </QueryClientProvider>,
  );

  return { ...view, invalidated };
}

function invalidatedKeys(invalidated: { mock: { calls: unknown[][] } }): unknown[] {
  return invalidated.mock.calls.map((call) => (call[0] as { queryKey: unknown }).queryKey);
}

describe("BillingQueueDetentionHolds", () => {
  it("shows nothing when no detention charge is held", () => {
    renderHolds([]);

    expect(screen.queryByTestId("billing-queue-detention-holds")).not.toBeInTheDocument();
  });

  it("lists each held charge with its amount and reason, linked to the charge", () => {
    renderHolds([
      hold(),
      hold({
        occurrenceId: "dto_2",
        locationName: "",
        billableAmount: 180,
        reason: "NoticeNotSent",
      }),
    ]);

    expect(screen.getByText("2 detention charges need approval")).toBeInTheDocument();
    expect(screen.getByText("$425.00")).toBeInTheDocument();
    expect(screen.getByText("Over the approval threshold")).toBeInTheDocument();
    expect(screen.getByText("Notice not sent in time")).toBeInTheDocument();

    expect(screen.getByRole("link", { name: "Riverside DC" })).toHaveAttribute(
      "href",
      recordPath("detention_occurrence", "dto_1"),
    );
    expect(screen.getByRole("link", { name: "Detention charge" })).toHaveAttribute(
      "href",
      recordPath("detention_occurrence", "dto_2"),
    );
  });

  it("names a held charge without a link for someone who cannot open the detention desk", () => {
    seedPermissions({});
    renderHolds([hold()]);

    expect(screen.queryByRole("link", { name: "Riverside DC" })).not.toBeInTheDocument();
    expect(screen.getByText("Riverside DC").closest("[data-slot=locked-reference]")).not.toBeNull();
    expect(screen.getByText("$425.00")).toBeInTheDocument();
  });

  it("offers no decision to someone who cannot decide detention", () => {
    renderHolds([hold()]);

    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Waive" })).not.toBeInTheDocument();
    expect(
      screen.getByText(
        "This item cannot be approved until each charge is approved or waived by someone who decides detention.",
      ),
    ).toBeInTheDocument();
  });

  it("approves a held charge where it is listed and refreshes the item", async () => {
    seedPermissions({ [Resource.Detention]: Operation.Read | Operation.Update });
    service.approve.mockResolvedValue({ id: "dto_2" });
    const { invalidated } = renderHolds([
      hold(),
      hold({ occurrenceId: "dto_2", locationName: "Plant 1" }),
    ]);

    const row = screen.getByText("Plant 1").closest("li") as HTMLElement;
    fireEvent.click(within(row).getByRole("button", { name: "Approve" }));

    await waitFor(() => expect(service.approve).toHaveBeenCalledWith("dto_2"));
    expect(service.approve).toHaveBeenCalledTimes(1);
    await waitFor(() =>
      expect(invalidatedKeys(invalidated)).toEqual(
        expect.arrayContaining([["billing-queue-list"], ["billingQueue"], ["detention"]]),
      ),
    );
  });

  it("waives a held charge with a coded reason from the item", async () => {
    seedPermissions({ [Resource.Detention]: Operation.Read | Operation.Update });
    service.waive.mockResolvedValue({ id: "dto_1" });
    renderHolds([hold()]);

    fireEvent.click(screen.getByRole("button", { name: "Waive" }));
    expect(await screen.findByRole("heading", { name: "Waive this charge" })).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(/Note/), {
      target: { value: "No signed POD on file" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Waive charge" }));

    await waitFor(() =>
      expect(service.waive).toHaveBeenCalledWith("dto_1", {
        reason: "CustomerGoodwill",
        note: "No signed POD on file",
      }),
    );
  });
});
