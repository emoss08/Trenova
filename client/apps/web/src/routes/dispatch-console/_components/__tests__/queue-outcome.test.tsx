import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { QueueOutcome } from "../move-queue";
import type { DispatchOutcome } from "../use-dispatch-actions";

// Written from DispatchOutcome in use-dispatch-actions.ts.
function outcome(overrides: Partial<DispatchOutcome> = {}): DispatchOutcome {
  return {
    id: 1,
    tone: "ok",
    text: "PRO-1048813 assigned to Dana Ortiz",
    undoable: true,
    ...overrides,
  };
}

function outcomeRow(text: string): HTMLElement {
  return screen.getByText(text).parentElement as HTMLElement;
}

describe("QueueOutcome", () => {
  afterEach(cleanup);

  it("draws a success on the success tint with its rule", () => {
    render(<QueueOutcome outcome={outcome()} canUndo onUndo={vi.fn()} />);

    const row = outcomeRow("PRO-1048813 assigned to Dana Ortiz");
    expect(row).toHaveClass(
      "bg-success-subtle",
      "border-success-border",
      "text-success-subtle-foreground",
    );
    expect(row).not.toHaveClass("bg-warning-subtle");
  });

  it("draws a refusal on the warning tint with its rule", () => {
    render(
      <QueueOutcome
        outcome={outcome({ tone: "warn", text: "PRO-1048813 was not assigned", undoable: false })}
        canUndo
        onUndo={vi.fn()}
      />,
    );

    const row = outcomeRow("PRO-1048813 was not assigned");
    expect(row).toHaveClass(
      "bg-warning-subtle",
      "border-warning-border",
      "text-warning-subtle-foreground",
    );
    expect(row).not.toHaveClass("bg-success-subtle");
    expect(screen.queryByRole("button", { name: /Undo/ })).toBeNull();
  });

  it("offers undo only for an undoable outcome while undo is possible", () => {
    const onUndo = vi.fn();
    const { rerender } = render(<QueueOutcome outcome={outcome()} canUndo onUndo={onUndo} />);

    fireEvent.click(screen.getByRole("button", { name: /Undo/ }));
    expect(onUndo).toHaveBeenCalledTimes(1);

    rerender(<QueueOutcome outcome={outcome()} canUndo={false} onUndo={onUndo} />);
    expect(screen.queryByRole("button", { name: /Undo/ })).toBeNull();
  });

  it("keeps the status region with nothing in it when there is no outcome", () => {
    render(<QueueOutcome outcome={null} canUndo onUndo={vi.fn()} />);

    expect(screen.getByRole("status")).toBeEmptyDOMElement();
  });
});
