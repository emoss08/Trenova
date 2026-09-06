import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PTOReasonDialog, type PTOReasonDialogMode } from "../pto-reason-dialog";

const { rejectWorkerPTO, cancelWorkerPTO, bulkWorkerPTOAction } = vi.hoisted(() => ({
  rejectWorkerPTO: vi.fn(),
  cancelWorkerPTO: vi.fn(),
  bulkWorkerPTOAction: vi.fn(),
}));

const { toastSuccess, toastWarning } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastWarning: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-mutations", () => ({
  rejectWorkerPTO,
  cancelWorkerPTO,
  bulkWorkerPTOAction,
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    worker: {
      listUpcomingPTO: { _def: ["worker", "list-upcoming-pto"] },
      ptoChartData: { _def: ["worker", "pto-chart-data"] },
    },
  },
}));

vi.mock("sonner", () => ({
  toast: { success: toastSuccess, warning: toastWarning, error: vi.fn(), info: vi.fn() },
}));

function renderDialog(ptoIds: string[], mode: PTOReasonDialogMode, skipped = 0) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={queryClient}>
      <PTOReasonDialog
        open
        onOpenChange={onOpenChange}
        ptoIds={ptoIds}
        mode={mode}
        skipped={skipped}
      />
    </QueryClientProvider>,
  );
  return { invalidateSpy, onOpenChange };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("PTOReasonDialog", () => {
  it("refuses to reject without a reason and never calls the API", async () => {
    const user = userEvent.setup();
    renderDialog(["wrkpto_1"], "reject");

    await user.click(screen.getByRole("button", { name: /confirm rejection/i }));

    expect(await screen.findByText(/reason is required/i)).toBeInTheDocument();
    expect(rejectWorkerPTO).not.toHaveBeenCalled();
    expect(bulkWorkerPTOAction).not.toHaveBeenCalled();
  });

  it("rejects a single request through the single mutation and refreshes every PTO query", async () => {
    rejectWorkerPTO.mockResolvedValue({});
    const user = userEvent.setup();
    const { invalidateSpy, onOpenChange } = renderDialog(["wrkpto_1"], "reject");

    await user.type(screen.getByLabelText("Reason"), "  No coverage  ");
    await user.click(screen.getByRole("button", { name: /confirm rejection/i }));

    await waitFor(() => {
      expect(rejectWorkerPTO).toHaveBeenCalledExactlyOnceWith("wrkpto_1", "No coverage");
    });
    expect(bulkWorkerPTOAction).not.toHaveBeenCalled();
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));

    const keys = invalidateSpy.mock.calls.map((call) => call[0]?.queryKey);
    expect(keys).toEqual(
      expect.arrayContaining([
        ["worker-pto-list"],
        ["worker", "list-upcoming-pto"],
        ["worker", "pto-chart-data"],
        ["worker-list"],
      ]),
    );
    expect(keys).not.toContainEqual(["worker", "upcoming-pto"]);
  });

  it("rejects several requests with exactly one bulk call and surfaces partial failures", async () => {
    bulkWorkerPTOAction.mockResolvedValue({
      successCount: 1,
      failureCount: 1,
      results: [
        { ptoId: "wrkpto_1", success: true, error: "" },
        { ptoId: "wrkpto_2", success: false, error: "PTO is approved and cannot be rejected" },
      ],
    });
    const user = userEvent.setup();
    renderDialog(["wrkpto_1", "wrkpto_2"], "reject", 1);

    await user.type(screen.getByLabelText("Reason"), "Coverage gap");
    await user.click(screen.getByRole("button", { name: /confirm rejection/i }));

    await waitFor(() => {
      expect(bulkWorkerPTOAction).toHaveBeenCalledExactlyOnceWith({
        ptoIds: ["wrkpto_1", "wrkpto_2"],
        action: "Reject",
        reason: "Coverage gap",
      });
    });
    expect(rejectWorkerPTO).not.toHaveBeenCalled();
    await waitFor(() => {
      expect(toastWarning).toHaveBeenCalledWith(
        "Rejected 1 PTO request; 1 failed (1 ineligible skipped)",
        { description: "PTO is approved and cannot be rejected" },
      );
    });
  });

  it("cancels without a reason, sending null so the server stores nothing", async () => {
    bulkWorkerPTOAction.mockResolvedValue({ successCount: 2, failureCount: 0, results: [] });
    const user = userEvent.setup();
    renderDialog(["wrkpto_1", "wrkpto_2"], "cancel");

    await user.click(screen.getByRole("button", { name: /confirm cancellation/i }));

    await waitFor(() => {
      expect(bulkWorkerPTOAction).toHaveBeenCalledExactlyOnceWith({
        ptoIds: ["wrkpto_1", "wrkpto_2"],
        action: "Cancel",
        reason: null,
      });
    });
  });

  it("cancels a single request through the single mutation with the trimmed reason", async () => {
    cancelWorkerPTO.mockResolvedValue({});
    const user = userEvent.setup();
    renderDialog(["wrkpto_9"], "cancel");

    await user.type(screen.getByLabelText("Reason"), " Schedule change ");
    await user.click(screen.getByRole("button", { name: /confirm cancellation/i }));

    await waitFor(() => {
      expect(cancelWorkerPTO).toHaveBeenCalledExactlyOnceWith("wrkpto_9", "Schedule change");
    });
  });
});
