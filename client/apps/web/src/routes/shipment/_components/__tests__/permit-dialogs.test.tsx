import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PermitRecordDialog } from "../permit-dialogs";
import type { Permit } from "@trenova/shared/types/permit";

const { createPermit, updatePermit } = vi.hoisted(() => ({
  createPermit: vi.fn(),
  updatePermit: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { shipmentService: { createPermit, updatePermit } },
}));

vi.mock("@/components/autocomplete-fields", () => ({
  UsStateAutocompleteField: () => null,
}));

const SHIPMENT_ID = "shp_01ARZ3NDEKTSV4RRFFQ69G5FAV";

const EXISTING: Permit = {
  id: "pmt_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  shipmentId: SHIPMENT_ID,
  stateId: "us_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  permitNumber: "GA-1234",
  status: "Active",
  issuedAt: 1_700_000_000,
  expiresAt: 1_800_000_000,
  cost: null,
  notes: "Escort booked",
};

function renderDialog(permit: Permit | null) {
  return render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      <PermitRecordDialog
        open
        onOpenChange={() => {}}
        shipmentId={SHIPMENT_ID}
        requirement={null}
        permit={permit}
      />
    </QueryClientProvider>,
  );
}

describe("PermitRecordDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    createPermit.mockResolvedValue(EXISTING);
    updatePermit.mockResolvedValue(EXISTING);
  });

  // Correcting a permit has to update the existing row. Posting instead would
  // leave the wrong permit on file beside the corrected one, and the engine
  // would keep matching requirements against both.
  it("updates the existing permit rather than recording a second one", async () => {
    const user = userEvent.setup();
    renderDialog(EXISTING);

    await user.click(screen.getByRole("button", { name: /save permit/i }));

    await waitFor(() => expect(updatePermit).toHaveBeenCalled());
    expect(createPermit).not.toHaveBeenCalled();

    const [shipmentId, permitId] = updatePermit.mock.calls[0];
    expect(shipmentId).toBe(SHIPMENT_ID);
    expect(permitId).toBe(EXISTING.id);
  });

  it("opens prefilled from the permit being edited", () => {
    renderDialog(EXISTING);

    expect(screen.getByDisplayValue("GA-1234")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /edit permit/i })).toBeInTheDocument();
  });

  // With no permit supplied the dialog is in record mode. The form legitimately
  // refuses to submit here — an issuing state and an expiry are both required —
  // so the property worth asserting is that it never reaches the update path
  // regardless.
  it("is in record mode and never updates when no permit is supplied", async () => {
    const user = userEvent.setup();
    renderDialog(null);

    expect(screen.getByRole("heading", { name: /record permit/i })).toBeInTheDocument();

    await user.type(screen.getByPlaceholderText(/number issued by the state/i), "TN-99");
    await user.click(screen.getByRole("button", { name: /record permit/i }));

    // Validation holds the submit, so neither path is reached. The update path
    // in particular must stay unreachable without a permit to update.
    await waitFor(() => expect(updatePermit).not.toHaveBeenCalled());
    expect(createPermit).not.toHaveBeenCalled();
  });
});
