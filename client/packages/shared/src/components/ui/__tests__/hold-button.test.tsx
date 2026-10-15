import { HoldButton } from "@trenova/shared/components/ui/hold-button";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("HoldButton", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("acts once held for the whole time", () => {
    const onDone = vi.fn();
    render(<HoldButton onDone={onDone}>Pause</HoldButton>);

    fireEvent.pointerDown(screen.getByRole("button"));
    act(() => {
      vi.advanceTimersByTime(899);
    });
    expect(onDone).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(onDone).toHaveBeenCalledTimes(1);
  });

  it("does nothing when let go early", () => {
    const onDone = vi.fn();
    render(<HoldButton onDone={onDone}>Pause</HoldButton>);
    const button = screen.getByRole("button");

    fireEvent.pointerDown(button);
    act(() => {
      vi.advanceTimersByTime(400);
    });
    fireEvent.pointerUp(button);
    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(onDone).not.toHaveBeenCalled();
  });

  it("acts on Enter held on the focused button and cancels on Escape", () => {
    const onDone = vi.fn();
    render(
      <HoldButton onDone={onDone} ms={500}>
        Pause
      </HoldButton>,
    );
    const button = screen.getByRole("button");

    fireEvent.keyDown(button, { key: "Enter" });
    act(() => {
      vi.advanceTimersByTime(200);
    });
    fireEvent.keyDown(button, { key: "Escape" });
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(onDone).not.toHaveBeenCalled();

    fireEvent.keyDown(button, { key: "Enter" });
    act(() => {
      vi.advanceTimersByTime(500);
    });
    expect(onDone).toHaveBeenCalledTimes(1);
  });

  it("does nothing when disabled", () => {
    const onDone = vi.fn();
    render(
      <HoldButton onDone={onDone} disabled>
        Pause
      </HoldButton>,
    );

    fireEvent.pointerDown(screen.getByRole("button"));
    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(onDone).not.toHaveBeenCalled();
  });

  it("shows the done label and no longer acts once done", () => {
    const onDone = vi.fn();
    render(
      <HoldButton onDone={onDone} done doneLabel="Paused" hint>
        Pause
      </HoldButton>,
    );
    const button = screen.getByRole("button");

    expect(button).toHaveTextContent("Paused");
    expect(button).not.toHaveTextContent("hold");
    expect(button).toHaveAttribute("aria-disabled", "true");

    fireEvent.pointerDown(button);
    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(onDone).not.toHaveBeenCalled();
  });

  it("asks to keep holding once the hold has started", () => {
    render(
      <HoldButton onDone={vi.fn()} hint>
        Pause
      </HoldButton>,
    );
    const button = screen.getByRole("button");

    expect(button).toHaveTextContent("Pausehold");

    fireEvent.pointerDown(button);
    expect(button).toHaveTextContent("Pausekeep holding");

    fireEvent.pointerUp(button);
    expect(button).toHaveTextContent("Pausehold");
  });
});
