import type { IftaMileageEntryRow } from "@/lib/graphql/ifta-jurisdiction-mileage";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeleteIftaMileageEntryDialog } from "../delete-ifta-mileage-entry-dialog";

const { deleteIftaMileageEntry, handleMutationError, toastSuccess } = vi.hoisted(() => ({
  deleteIftaMileageEntry: vi.fn(),
  handleMutationError: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/graphql/ifta-jurisdiction-mileage", () => ({
  deleteIftaMileageEntry,
  IFTA_MILEAGE_ENTRY_LIST_KEY: "ifta-mileage-entry-list",
}));

vi.mock("@/hooks/use-api-mutation", () => ({ handleMutationError }));

vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: vi.fn(), info: vi.fn() },
}));

const entry: IftaMileageEntryRow = {
  id: "ijme_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  tractorId: "trk_118",
  jurisdictionId: "ij_tx",
  traveledAt: 1_759_968_000,
  year: 2025,
  quarter: 4,
  miles: "412.50",
  loaded: false,
  source: "Manual",
  shipmentMoveId: null,
  notes: "Deadhead from Amarillo to the yard",
  createdById: "usr_1",
  version: 3,
  createdAt: 1,
  updatedAt: 2,
  tractor: { id: "trk_118", code: "118" },
  jurisdiction: { id: "ij_tx", countryCode: "US", code: "TX", name: "Texas" },
};

function renderDialog(row: IftaMileageEntryRow | null = entry) {
  const onOpenChange = vi.fn();
  const onDeleted = vi.fn();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <DeleteIftaMileageEntryDialog
        open
        onOpenChange={onOpenChange}
        entry={row}
        onDeleted={onDeleted}
      />
    </QueryClientProvider>,
  );
  return { onOpenChange, onDeleted };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("DeleteIftaMileageEntryDialog", () => {
  it("warns that the quarter's return drops the miles on its next recompute", async () => {
    renderDialog();

    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("Delete this entry?");
    expect(dialog).toHaveTextContent(
      "The quarter's return drops these miles on its next recompute.",
    );
    expect(dialog).toHaveTextContent("Q4 2025");
    expect(dialog).toHaveTextContent("412.50");
    expect(dialog).toHaveTextContent("TX");
    expect(dialog).toHaveTextContent("118");
  });

  it("deletes with the row's own id and version, then refreshes the list", async () => {
    deleteIftaMileageEntry.mockResolvedValue(true);
    const { onDeleted, onOpenChange } = renderDialog();

    fireEvent.click(await screen.findByRole("button", { name: /delete entry/i }));

    await waitFor(() => {
      expect(deleteIftaMileageEntry).toHaveBeenCalledExactlyOnceWith("ijme_1", 3);
    });
    await waitFor(() => {
      expect(onDeleted).toHaveBeenCalledTimes(1);
    });
    expect(toastSuccess).toHaveBeenCalledTimes(1);
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("keeps the dialog open and reports the failure when the delete is rejected", async () => {
    const failure = new Error("version mismatch");
    deleteIftaMileageEntry.mockRejectedValue(failure);
    const { onDeleted, onOpenChange } = renderDialog();

    fireEvent.click(await screen.findByRole("button", { name: /delete entry/i }));

    await waitFor(() => {
      expect(handleMutationError).toHaveBeenCalledWith({
        error: failure,
        resourceName: "Jurisdiction Mileage",
      });
    });
    expect(onDeleted).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(toastSuccess).not.toHaveBeenCalled();
  });

  it("says a system-written entry was not keyed by hand", async () => {
    renderDialog({ ...entry, source: "RouteCalculation", shipmentMoveId: "smv_1" });

    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent(/route calculation/i);
  });

  it("cannot be confirmed with no entry selected", async () => {
    renderDialog(null);

    expect(await screen.findByRole("button", { name: /delete entry/i })).toBeDisabled();
    expect(deleteIftaMileageEntry).not.toHaveBeenCalled();
  });
});
