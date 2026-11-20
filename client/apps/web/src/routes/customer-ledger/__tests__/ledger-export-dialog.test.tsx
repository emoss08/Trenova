import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FormFieldStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { endOfChosenDay, LedgerExportDialog } from "../_components/ledger-export-dialog";

const mocks = vi.hoisted(() => ({
  exportCustomerLedger: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/graphql/table-export", () => ({
  exportCustomerLedger: mocks.exportCustomerLedger,
}));
vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: FormFieldStub,
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: mocks.toastSuccess, error: vi.fn() },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ledger export dialog", () => {
  it("starts a server export of the whole ledger named for the customer", async () => {
    mocks.exportCustomerLedger.mockResolvedValue({ definitionId: "rdef_1" });
    const onOpenChange = vi.fn();

    renderAccountingPage(
      <LedgerExportDialog
        open
        onOpenChange={onOpenChange}
        customerId="cus_1"
        customerName="Acme Freight"
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Export" }));

    await waitFor(() => expect(mocks.exportCustomerLedger).toHaveBeenCalledTimes(1));
    expect(mocks.exportCustomerLedger).toHaveBeenCalledWith({
      name: "Acme Freight ledger",
      customerId: "cus_1",
      startDate: undefined,
      endDate: undefined,
      format: "xlsx",
    });
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Your export is being built", {
      description: "You will be told when the file is ready. It is kept in Reports.",
    });
  });

  it("refuses an export without a name", async () => {
    renderAccountingPage(
      <LedgerExportDialog open onOpenChange={vi.fn()} customerId="cus_1" customerName="Acme" />,
    );
    fireEvent.change(screen.getByRole("textbox", { name: /Name/ }), { target: { value: "  " } });
    fireEvent.click(screen.getByRole("button", { name: "Export" }));

    expect(await screen.findByText("Give the export a name")).toBeInTheDocument();
    expect(mocks.exportCustomerLedger).not.toHaveBeenCalled();
  });
});

describe("endOfChosenDay", () => {
  it("takes in the whole day the picker chose", () => {
    expect(endOfChosenDay(1_767_225_600)).toBe(1_767_311_999);
  });

  it("leaves an open end open", () => {
    expect(endOfChosenDay(null)).toBeUndefined();
  });
});
