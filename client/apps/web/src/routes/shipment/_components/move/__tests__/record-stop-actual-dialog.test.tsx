import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { RecordStopActualDialog } from "../record-stop-actual-dialog";

function renderDialog(overrides: Partial<Parameters<typeof RecordStopActualDialog>[0]> = {}) {
  const onConfirm = vi.fn();
  const onOpenChange = vi.fn();
  render(
    <RecordStopActualDialog
      open
      onOpenChange={onOpenChange}
      action="Arrive"
      stopDescription="Pickup · Chicago, IL"
      isSubmitting={false}
      onConfirm={onConfirm}
      {...overrides}
    />,
  );
  return { onConfirm, onOpenChange };
}

function toDatetimeLocal(date: Date): string {
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(
    date.getHours(),
  )}:${pad(date.getMinutes())}`;
}

describe("RecordStopActualDialog", () => {
  it("renders the arrival copy with the stop description", () => {
    renderDialog();

    expect(screen.getByRole("heading", { name: "Record Arrival" })).toBeInTheDocument();
    expect(screen.getByText(/Pickup · Chicago, IL/)).toBeInTheDocument();
  });

  it("renders the departure copy when departing", () => {
    renderDialog({ action: "Depart" });

    expect(screen.getByRole("heading", { name: "Record Departure" })).toBeInTheDocument();
  });

  it("confirms with no event time when the field is left empty", async () => {
    const user = userEvent.setup();
    const { onConfirm } = renderDialog();

    await user.click(screen.getByRole("button", { name: "Record Arrival" }));

    expect(onConfirm).toHaveBeenCalledWith(undefined);
  });

  it("converts a backdated event time to unix seconds", async () => {
    const user = userEvent.setup();
    const { onConfirm } = renderDialog();

    const past = new Date(Date.now() - 60 * 60 * 1000);
    past.setSeconds(0, 0);
    await user.type(screen.getByLabelText(/Event time/), toDatetimeLocal(past));
    await user.click(screen.getByRole("button", { name: "Record Arrival" }));

    expect(onConfirm).toHaveBeenCalledWith(Math.floor(past.getTime() / 1000));
  });

  it("rejects a future event time without confirming", async () => {
    const user = userEvent.setup();
    const { onConfirm } = renderDialog();

    const future = new Date(Date.now() + 24 * 60 * 60 * 1000);
    await user.type(screen.getByLabelText(/Event time/), toDatetimeLocal(future));
    await user.click(screen.getByRole("button", { name: "Record Arrival" }));

    expect(onConfirm).not.toHaveBeenCalled();
    expect(screen.getByText("The event time cannot be in the future")).toBeInTheDocument();
  });
});
