import { act, cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskLeavingNotes, NOTES_LEAVE_MS } from "../desk-leaving-notes";

const motion = vi.hoisted(() => ({ reduce: false }));

vi.mock("motion/react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("motion/react")>()),
  useReducedMotion: () => motion.reduce,
}));

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  motion.reduce = false;
});

function notes(leaving: boolean) {
  return (
    <DeskLeavingNotes leaving={leaving}>
      <p>Looking up the lanes.</p>
    </DeskLeavingNotes>
  );
}

/*
The running turn's notes fold away once the board they led to is known: out of
reach of keyboard and screen reader while they go, and gone from the page when
the fold ends. jsdom draws no layout, so the fold's height and opacity are not
asserted; the classes that drive them are CSS transitions with no measuring.
*/
describe("DeskLeavingNotes", () => {
  it("draws the notes in reach while they stay", () => {
    render(notes(false));

    const note = screen.getByText("Looking up the lanes.");
    expect(note.closest("[inert]")).toBeNull();
    expect(note.closest("[aria-hidden]")).toBeNull();
  });

  it("puts leaving notes out of reach, then takes them off the page when the fold ends", () => {
    const { rerender } = render(notes(false));
    rerender(notes(true));

    const wrapper = screen.getByText("Looking up the lanes.").closest("[inert]");
    expect(wrapper).toHaveAttribute("aria-hidden", "true");
    expect(wrapper).toHaveClass("grid-rows-[0fr]", "opacity-0");

    act(() => vi.advanceTimersByTime(NOTES_LEAVE_MS - 1));
    expect(screen.getByText("Looking up the lanes.")).toBeInTheDocument();

    act(() => vi.advanceTimersByTime(1));
    expect(screen.queryByText("Looking up the lanes.")).not.toBeInTheDocument();
  });

  it("draws nothing for notes already leaving when first drawn, so no fold replays", () => {
    const { container } = render(notes(true));

    expect(container).toBeEmptyDOMElement();
  });

  it("takes leaving notes away at once when motion is reduced", () => {
    motion.reduce = true;
    const { rerender, container } = render(notes(false));
    expect(screen.getByText("Looking up the lanes.")).toBeInTheDocument();

    rerender(notes(true));

    expect(container).toBeEmptyDOMElement();
  });

  it("brings the notes back if the board they led to is no longer coming", () => {
    const { rerender } = render(notes(false));
    rerender(notes(true));
    act(() => vi.advanceTimersByTime(NOTES_LEAVE_MS));

    rerender(notes(false));

    const note = screen.getByText("Looking up the lanes.");
    expect(note.closest("[inert]")).toBeNull();
  });
});
