import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ConfirmDialog, type ConfirmDialogProps } from "../confirm-dialog";

function renderDialog(props: Partial<ConfirmDialogProps> = {}) {
  const onOpenChange = vi.fn<(open: boolean) => void>();
  const onConfirm = vi.fn<() => Promise<unknown> | undefined>();
  render(
    <ConfirmDialog
      open
      onOpenChange={onOpenChange}
      title="Delete this safety event?"
      description="The speeding event on Nov 4 comes off the record. This cannot be undone."
      confirmLabel="Delete event"
      onConfirm={onConfirm}
      {...props}
    />,
  );
  return {
    onOpenChange,
    onConfirm: (props.onConfirm as typeof onConfirm | undefined) ?? onConfirm,
  };
}

function deferred() {
  let resolve: () => void = () => undefined;
  let reject: (error: Error) => void = () => undefined;
  const promise = new Promise<void>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe("ConfirmDialog", () => {
  it("asks with its title and sentence, backing out with Cancel", async () => {
    const user = userEvent.setup();
    const { onOpenChange, onConfirm } = renderDialog();

    const dialog = screen.getByRole("alertdialog", { name: "Delete this safety event?" });
    expect(dialog).toHaveTextContent(
      "The speeding event on Nov 4 comes off the record. This cannot be undone.",
    );
    expect(screen.getByRole("button", { name: "Delete event" })).toHaveClass("bg-danger");

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("confirms in the plain tone, under its own cancel words, with what else it holds", () => {
    renderDialog({
      variant: "default",
      cancelLabel: "Keep access",
      children: <p>Settlement history stays.</p>,
    });

    expect(screen.getByRole("button", { name: "Delete event" })).toHaveClass("bg-ink");
    expect(screen.getByRole("button", { name: "Keep access" })).toBeInTheDocument();
    expect(screen.getByText("Settlement history stays.")).toBeInTheDocument();
  });

  it("holds while the step runs, refuses a second press, and closes when it succeeds", async () => {
    const user = userEvent.setup();
    const step = deferred();
    const onConfirm = vi.fn(() => step.promise);
    const { onOpenChange } = renderDialog({ onConfirm });

    await user.click(screen.getByRole("button", { name: "Delete event" }));
    await user.click(screen.getByRole("button", { name: "Delete event" }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "Delete event" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    expect(onOpenChange).not.toHaveBeenCalled();

    step.resolve();

    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Delete event" })).not.toBeDisabled(),
    );
  });

  it("stays open on a refusal and lets the person try again", async () => {
    const user = userEvent.setup();
    const step = deferred();
    const onConfirm = vi.fn(() => step.promise);
    const { onOpenChange } = renderDialog({ onConfirm });

    await user.click(screen.getByRole("button", { name: "Delete event" }));
    step.reject(new Error("refused"));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Delete event" })).not.toBeDisabled(),
    );
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("leaves closing to a caller whose step returns nothing, holding while it says it is pending", () => {
    const onConfirm = vi.fn(() => undefined);
    const { onOpenChange } = renderDialog({ onConfirm, pending: true });

    expect(screen.getByRole("button", { name: "Delete event" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    expect(onOpenChange).not.toHaveBeenCalled();
  });

  it("runs a step that returns nothing and does not close by itself", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn(() => undefined);
    const { onOpenChange } = renderDialog({ onConfirm });

    await user.click(screen.getByRole("button", { name: "Delete event" }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(onOpenChange).not.toHaveBeenCalled();
  });
});
