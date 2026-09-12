import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { OffCycleInvoiceDialog, offCycleWarningFrom } from "../off-cycle-invoice-dialog";

afterEach(cleanup);

/** Both the REST and the GraphQL error classes expose this method. */
function apiError(fieldErrors: { field: string; message: string }[]) {
  return {
    getFieldErrors: (field?: string) =>
      field === undefined ? fieldErrors : fieldErrors.filter((e) => e.field === field),
  };
}

describe("offCycleWarningFrom", () => {
  // The server's message names the customer and their cycle, which is the whole
  // point of surfacing it rather than writing our own.
  it("returns the server's own message for a cadence refusal", () => {
    const warning = offCycleWarningFrom(
      apiError([
        {
          field: "offCycleReason",
          message: "Acme Freight is billed on a monthly statement.",
        },
      ]),
    );

    expect(warning).toBe("Acme Freight is billed on a monthly statement.");
  });

  // An ordinary validation failure must stay an error toast. Treating it as the
  // cadence guard would offer a reason box for a problem a reason cannot fix.
  it("ignores validation errors on other fields", () => {
    expect(
      offCycleWarningFrom(
        apiError([{ field: "shipmentIds", message: "Shipment must be completed" }]),
      ),
    ).toBeNull();
  });

  it("ignores anything that is not an API error", () => {
    expect(offCycleWarningFrom(new Error("network down"))).toBeNull();
    expect(offCycleWarningFrom(null)).toBeNull();
    expect(offCycleWarningFrom("offCycleReason")).toBeNull();
    expect(offCycleWarningFrom(apiError([]))).toBeNull();
  });
});

describe("OffCycleInvoiceDialog", () => {
  it("shows the server's warning and will not submit without a reason", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(
      <OffCycleInvoiceDialog
        open
        warning="Acme Freight is billed on a monthly statement."
        pending={false}
        onOpenChange={vi.fn()}
        onConfirm={onConfirm}
      />,
    );

    expect(screen.getByText("Acme Freight is billed on a monthly statement.")).toBeInTheDocument();

    const confirm = screen.getByRole("button", { name: /invoice anyway/i });
    expect(confirm).toBeDisabled();

    await user.click(confirm);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("passes the trimmed reason back once one is typed", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(
      <OffCycleInvoiceDialog
        open
        warning="Acme Freight is billed on a monthly statement."
        pending={false}
        onOpenChange={vi.fn()}
        onConfirm={onConfirm}
      />,
    );

    await user.type(
      screen.getByLabelText(/reason for invoicing outside the statement/i),
      "  Billing to the broker  ",
    );
    await user.click(screen.getByRole("button", { name: /invoice anyway/i }));

    expect(onConfirm).toHaveBeenCalledWith("Billing to the broker");
  });

  // Whitespace is not a reason. A biller who types a space to get past the guard
  // leaves nothing for the next person to read.
  it("treats whitespace as no reason at all", async () => {
    const user = userEvent.setup();
    render(
      <OffCycleInvoiceDialog
        open
        warning="Acme Freight is billed on a monthly statement."
        pending={false}
        onOpenChange={vi.fn()}
        onConfirm={vi.fn()}
      />,
    );

    await user.type(screen.getByLabelText(/reason for invoicing outside the statement/i), "   ");

    expect(screen.getByRole("button", { name: /invoice anyway/i })).toBeDisabled();
  });
});
