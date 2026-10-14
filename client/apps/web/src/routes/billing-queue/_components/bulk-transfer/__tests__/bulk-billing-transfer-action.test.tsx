import { act, cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { setLocale } from "@trenova/shared/i18n/runtime";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BulkBillingTransferAction } from "../bulk-billing-transfer-action";

const mocks = vi.hoisted(() => {
  let release: () => void = () => {};
  const ready = new Promise<void>((resolve) => {
    release = resolve;
  });
  return {
    permission: vi.fn(),
    dialogRequested: vi.fn(),
    dialogCode: { ready, release: () => release() },
  };
});

vi.mock("@/hooks/use-permission", () => ({
  usePermission: mocks.permission,
}));

// The dialog's code is held until the test lets it arrive, so what the person
// sees in between can be asserted.
vi.mock("../bulk-billing-transfer-dialog", async () => {
  mocks.dialogRequested();
  await mocks.dialogCode.ready;
  return { BulkBillingTransferDialog: TestDialog };
});

function TestDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return open ? (
    <div role="dialog" aria-label="Transfer to billing">
      <button type="button" onClick={() => onOpenChange(false)}>
        close
      </button>
    </div>
  ) : null;
}

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
  // Runs first: the dialog's module is read once per file, and this is the
  // test that watches it arrive.
  it("opens the dialog's frame at once and swaps in the dialog when its code arrives", async () => {
    mocks.permission.mockReturnValue({ allowed: true, isLoading: false });
    const user = userEvent.setup();
    render(<Harness />);

    expect(mocks.permission).toHaveBeenCalledWith(Resource.Shipment, Operation.Update);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(mocks.dialogRequested).not.toHaveBeenCalled();

    const trigger = screen.getByRole("button", { name: "Transfer to billing" });
    await user.hover(trigger);
    expect(mocks.dialogRequested).toHaveBeenCalledTimes(1);

    await user.click(trigger);
    const frame = await screen.findByRole("dialog", { name: "Transfer to billing" });
    expect(within(frame).getByLabelText("Loading shipments")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByRole("button", { name: "close" })).not.toBeInTheDocument();

    await act(async () => {
      mocks.dialogCode.release();
      await mocks.dialogCode.ready;
    });

    await user.click(await screen.findByRole("button", { name: "close" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(mocks.dialogRequested).toHaveBeenCalledTimes(1);
  });

  it("is not offered to someone who cannot update shipments", () => {
    mocks.permission.mockReturnValue({ allowed: false, isLoading: false });
    render(<Harness />);

    expect(screen.queryByRole("button", { name: "Transfer to billing" })).not.toBeInTheDocument();
  });
});
