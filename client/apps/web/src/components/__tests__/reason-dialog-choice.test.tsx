import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReasonDialog } from "../reason-dialog";

afterEach(cleanup);

function renderChoice() {
  const onConfirm = vi.fn<(reason: string, choice?: string) => void>();
  render(
    <ReasonDialog
      open
      title="Hold settlement DS-204"
      description="It keeps its place in the batch."
      placeholder="What the hold waits on"
      confirmLabel="Hold settlement"
      label="Note"
      optional
      choice={{
        label: "Why",
        placeholder: "Pick a reason",
        options: [
          { value: "RateDispute", label: "Rate dispute" },
          { value: "Other", label: "Other" },
        ],
        needsWords: ["Other"],
      }}
      pending={false}
      onConfirm={onConfirm}
      onOpenChange={() => {}}
    />,
  );
  return onConfirm;
}

async function pick(label: string) {
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Pick a reason" }));
  await user.click((await screen.findAllByText(label)).at(-1) as Element);
}

describe("ReasonDialog with a choice", () => {
  it("refuses to confirm until a reason is picked", async () => {
    const onConfirm = renderChoice();

    fireEvent.click(screen.getByRole("button", { name: "Hold settlement" }));

    expect(await screen.findByText("Pick one")).toBeInTheDocument();
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("confirms a picked reason with an empty note when the note is optional", async () => {
    const onConfirm = renderChoice();

    await pick("Rate dispute");
    fireEvent.click(screen.getByRole("button", { name: "Hold settlement" }));

    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith("", "RateDispute"));
  });

  it("asks for words when the pick needs them", async () => {
    const onConfirm = renderChoice();

    await pick("Other");
    fireEvent.click(screen.getByRole("button", { name: "Hold settlement" }));
    expect(await screen.findByText("Say why")).toBeInTheDocument();
    expect(onConfirm).not.toHaveBeenCalled();

    fireEvent.change(screen.getByRole("textbox"), { target: { value: " Garnishment order " } });
    fireEvent.click(screen.getByRole("button", { name: "Hold settlement" }));
    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith("Garnishment order", "Other"));
  });
});
