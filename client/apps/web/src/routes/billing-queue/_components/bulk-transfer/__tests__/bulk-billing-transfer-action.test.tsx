import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setLocale } from "@trenova/shared/i18n/runtime";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BulkBillingTransferAction } from "../bulk-billing-transfer-action";

const mocks = vi.hoisted(() => ({
  permission: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: mocks.permission,
}));

vi.mock("../bulk-billing-transfer-dialog", () => ({
  BulkBillingTransferDialog: ({
    open,
    onOpenChange,
  }: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
  }) =>
    open ? (
      <div role="dialog" aria-label="Transfer to Billing">
        <button type="button" onClick={() => onOpenChange(false)}>
          close
        </button>
      </div>
    ) : null,
}));

afterEach(async () => {
  cleanup();
  vi.clearAllMocks();
  await act(() => setLocale("en"));
});

// The dialog is controlled by the page now, because the run it is watching
// lives in the URL. This stands in for that owner.
function Harness() {
  const [open, setOpen] = useState(false);
  const [runId, setRunId] = useState<string | null>(null);

  return (
    <BulkBillingTransferAction
      open={open}
      onOpenChange={setOpen}
      runId={runId}
      onRunIdChange={setRunId}
    />
  );
}

describe("BulkBillingTransferAction", () => {
  it("opens the bulk transfer for someone allowed to update shipments", async () => {
    mocks.permission.mockReturnValue({ allowed: true, isLoading: false });
    const user = userEvent.setup();
    render(<Harness />);

    expect(mocks.permission).toHaveBeenCalledWith(Resource.Shipment, Operation.Update);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Transfer to Billing" }));
    expect(await screen.findByRole("dialog", { name: "Transfer to Billing" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "close" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("is not offered to someone who cannot update shipments", () => {
    mocks.permission.mockReturnValue({ allowed: false, isLoading: false });
    render(<Harness />);

    expect(screen.queryByRole("button", { name: "Transfer to Billing" })).not.toBeInTheDocument();
  });
});
