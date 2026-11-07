import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { cleanup, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { stepMove, useDispatchHotkeys } from "../use-dispatch-hotkeys";

const moves = ["sm_a", "sm_b", "sm_c"].map((moveId) => ({ moveId }) as DispatchBoardMove);

describe("stepMove", () => {
  it("starts at the first move shown when none is open", () => {
    expect(stepMove(moves, null, 1)).toBe("sm_a");
  });

  it("walks forward and back through the moves shown", () => {
    expect(stepMove(moves, "sm_a", 1)).toBe("sm_b");
    expect(stepMove(moves, "sm_b", -1)).toBe("sm_a");
  });

  it("stays on the last and first move rather than wrapping", () => {
    expect(stepMove(moves, "sm_c", 1)).toBe("sm_c");
    expect(stepMove(moves, "sm_a", -1)).toBe("sm_a");
  });

  it("starts over at the top when the open move is no longer shown", () => {
    expect(stepMove(moves, "sm_gone", 1)).toBe("sm_a");
  });

  it("has nowhere to go on an empty queue", () => {
    expect(stepMove([], null, 1)).toBeNull();
  });
});

function Harness({
  onSelectMove,
  queue,
}: {
  onSelectMove: (moveId: string) => void;
  queue: HTMLElement | null;
}) {
  useDispatchHotkeys({
    enabled: true,
    orderedMoves: moves,
    selectedMoveId: null,
    canUndo: false,
    queueScope: queue,
    onSelectMove,
    onClearSelection: vi.fn(),
    onShiftWindow: vi.fn(),
    onToday: vi.fn(),
    onUndo: vi.fn(),
  });
  return null;
}

afterEach(cleanup);

describe("useDispatchHotkeys", () => {
  it("answers J from inside the queue, which is a listbox of its own", async () => {
    const queue = document.createElement("div");
    queue.setAttribute("role", "listbox");
    const row = document.createElement("div");
    row.tabIndex = 0;
    queue.append(row);
    document.body.append(queue);
    const onSelectMove = vi.fn();
    render(<Harness onSelectMove={onSelectMove} queue={queue} />);

    row.focus();
    await userEvent.keyboard("j");

    expect(onSelectMove).toHaveBeenCalledWith("sm_a");
    queue.remove();
  });

  it("leaves J to another popup's list, which owns its own keys", async () => {
    const popup = document.createElement("div");
    popup.setAttribute("role", "listbox");
    const option = document.createElement("div");
    option.tabIndex = 0;
    popup.append(option);
    document.body.append(popup);
    const onSelectMove = vi.fn();
    render(<Harness onSelectMove={onSelectMove} queue={null} />);

    option.focus();
    await userEvent.keyboard("j");

    expect(onSelectMove).not.toHaveBeenCalled();
    popup.remove();
  });
});
