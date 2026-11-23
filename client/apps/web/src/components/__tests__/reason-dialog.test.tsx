import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReasonDialog } from "../reason-dialog";

afterEach(cleanup);

function renderDialog(props: {
  optional?: boolean;
  label?: string;
  maxLength?: number;
  cancelLabel?: string;
  onOpenChange?: (open: boolean) => void;
}) {
  const onConfirm = vi.fn<(reason: string) => void>();
  render(
    <ReasonDialog
      open
      title="Decline this plan?"
      description="Say why."
      placeholder="Optional"
      confirmLabel="Decline plan"
      pending={false}
      onConfirm={onConfirm}
      onOpenChange={() => {}}
      {...props}
    />,
  );
  return onConfirm;
}

describe("ReasonDialog", () => {
  it("refuses an empty reason when one is required", async () => {
    const onConfirm = renderDialog({});

    fireEvent.click(screen.getByRole("button", { name: "Decline plan" }));

    expect(await screen.findByText("Say why")).toBeInTheDocument();
    expect(onConfirm).not.toHaveBeenCalled();
    expect(screen.getByText("Reason")).toBeInTheDocument();
  });

  it("confirms with nothing written when the note is optional, under its own label", async () => {
    const onConfirm = renderDialog({ optional: true, label: "Note for the agent" });

    expect(screen.getByText("Note for the agent")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Decline plan" }));

    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith(""));
  });

  it("still holds an optional note to the longest the server keeps", async () => {
    const onConfirm = renderDialog({ optional: true, maxLength: 5 });

    fireEvent.change(screen.getByRole("textbox"), { target: { value: "too long" } });
    fireEvent.click(screen.getByRole("button", { name: "Decline plan" }));

    expect(await screen.findByText("Keep the reason under 5 characters")).toBeInTheDocument();
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("backs out under its own words when given them, and reads Back otherwise", () => {
    const onOpenChange = vi.fn();
    renderDialog({ cancelLabel: "Keep checklist", onOpenChange });

    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Keep checklist" }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("reads Back on the button that backs out by default", () => {
    renderDialog({});

    expect(screen.getByRole("button", { name: "Back" })).toBeInTheDocument();
  });

  it("opens on its default reason and keeps what is typed when the default changes while open", () => {
    const props = {
      title: "Close this event?",
      description: "Say how it ended.",
      placeholder: "Resolution",
      confirmLabel: "Close event",
      pending: false,
      onConfirm: vi.fn(),
      onOpenChange: vi.fn(),
    };
    const { rerender } = render(<ReasonDialog open defaultReason="Coached" {...props} />);
    const field = screen.getByRole("textbox") as HTMLTextAreaElement;
    expect(field.value).toBe("Coached");

    fireEvent.change(field, { target: { value: "Coached and retrained" } });
    rerender(<ReasonDialog open defaultReason="Refetched" {...props} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe(
      "Coached and retrained",
    );
  });
});
