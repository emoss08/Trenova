import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { HoldButton } from "../hold-button";

describe("HoldButton", () => {
  let now = 0;

  beforeEach(() => {
    now = 0;
    vi.spyOn(performance, "now").mockImplementation(() => now);
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) =>
      window.setTimeout(() => callback(now), 16),
    );
    vi.stubGlobal("cancelAnimationFrame", (id: number) => window.clearTimeout(id));
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("acts once held for the whole time", () => {
    const onConfirm = vi.fn();
    render(<HoldButton label="Pause" doneLabel="Paused" done={false} onConfirm={onConfirm} />);

    fireEvent.pointerDown(screen.getByRole("button"));
    act(() => {
      now = 950;
      vi.advanceTimersByTime(40);
    });

    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("does nothing when let go early", () => {
    const onConfirm = vi.fn();
    render(<HoldButton label="Pause" doneLabel="Paused" done={false} onConfirm={onConfirm} />);
    const button = screen.getByRole("button");

    fireEvent.pointerDown(button);
    act(() => {
      now = 400;
      vi.advanceTimersByTime(20);
    });
    fireEvent.pointerUp(button);
    act(() => {
      now = 1200;
      vi.advanceTimersByTime(40);
    });

    expect(onConfirm).not.toHaveBeenCalled();
  });
});
