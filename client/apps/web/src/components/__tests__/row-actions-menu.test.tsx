import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RowActionsMenu } from "../row-actions-menu";

afterEach(cleanup);

describe("RowActionsMenu", () => {
  // A row the viewer cannot act on must not grow a menu that opens onto
  // nothing; the absence of the trigger is the signal.
  it("renders no trigger when there is nothing to do", () => {
    render(<RowActionsMenu label="Actions for CDL" actions={[]} />);

    expect(screen.queryByRole("button", { name: "Actions for CDL" })).toBeNull();
  });

  it("opens on the trigger and runs the chosen action", async () => {
    const renew = vi.fn();
    const archive = vi.fn();
    render(
      <RowActionsMenu
        label="Actions for CDL"
        actions={[
          { id: "renew", label: "Renew", onSelect: renew },
          { id: "archive", label: "Archive", onSelect: archive, destructive: true },
        ]}
      />,
    );

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Actions for CDL" }));
    await user.click(await screen.findByRole("menuitem", { name: "Renew" }));

    expect(renew).toHaveBeenCalledOnce();
    expect(archive).not.toHaveBeenCalled();
  });
});
