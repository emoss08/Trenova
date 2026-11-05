import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskUndoBar } from "../desk-undo-bar";
import type { UndoWindow } from "../undo-window";

afterEach(cleanup);

const held: UndoWindow = {
  key: "proposal:aprop_1",
  title: "Assign driver",
  what: "Assign driver on S-240201",
  target: { proposalId: "aprop_1" },
  proposalIds: ["aprop_1"],
  planId: null,
  commitsAt: null,
  startedAt: 0,
};

function renderBar() {
  render(<DeskUndoBar held={held} left={5} busy={false} onUndo={vi.fn()} onNow={vi.fn()} />);
}

/**
 * Approve leaves with the approval, which used to drop focus to the page while the
 * seconds to take it back ran out. Undo takes that focus, and only that focus.
 */
describe("DeskUndoBar focus", () => {
  it("moves focus to Undo when the approval left focus on the page", () => {
    renderBar();

    expect(screen.getByRole("button", { name: "Undo" })).toHaveFocus();
  });

  it("leaves focus where the person put it, such as the message box", () => {
    const box = document.createElement("textarea");
    document.body.append(box);
    box.focus();

    renderBar();

    expect(box).toHaveFocus();
    box.remove();
  });
});
