import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { StreamingTurn } from "../streaming-turn";
import { initialTurnState, type TurnState } from "../turn-stream";
import { DESK_SETTLE_MS, DeskMark, DeskThinking } from "../voice/desk-thinking";

const motion = vi.hoisted(() => ({ reduce: false }));

vi.mock("motion/react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("motion/react")>();
  return { ...actual, useReducedMotion: () => motion.reduce };
});

const MOVING = '[class*="animate-desk-"]';

function mark(container: HTMLElement) {
  return container.querySelector('[data-slot="desk-mark"]');
}

beforeEach(() => {
  motion.reduce = false;
});

afterEach(() => {
  vi.useRealTimers();
});

/**
 * Each pose moves one part of the desk, so the three phases of a turn read
 * apart: the chair for the model, the desk for a tool, the screen for words.
 */
describe("DeskMark poses", () => {
  it("rolls the chair while waiting on the model", () => {
    const { container } = render(<DeskMark pose="arrive" />);

    expect(container.querySelector('[data-part="chair"]')).toHaveClass("animate-desk-roll");
    expect(container.querySelector('[data-part="desk"]')).not.toHaveClass("animate-desk-shake");
    expect(container.querySelector('[data-part="screen"]')).not.toHaveClass("animate-desk-screen");
  });

  it("shakes the desk while a tool runs", () => {
    const { container } = render(<DeskMark pose="busy" />);

    expect(container.querySelector('[data-part="desk"]')).toHaveClass("animate-desk-shake");
    expect(container.querySelector('[data-part="chair"]')).not.toHaveClass("animate-desk-roll");
  });

  it("pulses the screen while the answer streams", () => {
    const { container } = render(<DeskMark pose="write" />);

    expect(container.querySelector('[data-part="screen"]')).toHaveClass("animate-desk-screen");
    expect(container.querySelector('[data-part="desk"]')).not.toHaveClass("animate-desk-shake");
  });

  it("holds the idle desk still unless the page has just opened", () => {
    const { container } = render(<DeskMark pose="idle" timeOfDay="morning" />);

    expect(container.querySelector(MOVING)).toBeNull();
    expect(container.querySelector('[data-part="sun"]')).not.toBeNull();
    expect(container.querySelector('[data-part="lamp-light"]')).toBeNull();
  });

  it("switches the lamp on once in the evening and draws the moon", () => {
    const { container } = render(<DeskMark pose="idle" timeOfDay="evening" welcome />);

    expect(container.querySelector('[data-part="lamp-light"]')).toHaveClass("animate-desk-lamp");
    expect(container.querySelector('[data-part="chair"]')).toHaveClass("animate-desk-arrive");
    expect(container.querySelector('[data-part="moon"]')).not.toBeNull();
    expect(container.querySelector('[data-part="sun"]')).toBeNull();
  });

  it("never exposes the drawing to assistive technology", () => {
    const { container } = render(<DeskMark pose="busy" />);

    expect(mark(container)).toHaveAttribute("aria-hidden", "true");
  });
});

/**
 * Reduced motion keeps the drawing and the state it shows, and drops every
 * loop and one-shot, so the mark is a still picture of the same moment.
 */
describe("DeskThinking with reduced motion", () => {
  beforeEach(() => {
    motion.reduce = true;
  });

  it.each(["arrive", "busy", "write"] as const)("draws the %s pose still", (pose) => {
    const { container } = render(<DeskThinking working pose={pose} />);

    expect(mark(container)).toHaveAttribute("data-motion", "still");
    expect(mark(container)).toHaveAttribute("data-pose", pose);
    expect(container.querySelector(MOVING)).toBeNull();
  });

  it("draws the chair pulled out, not rolling, for the model's turn", () => {
    const { container } = render(<DeskThinking working pose="arrive" />);
    const chair = container.querySelector('[data-part="chair"]');

    expect(chair?.getAttribute("style") ?? "").toContain("translate3d");
  });

  it("still says that the work is under way", () => {
    render(<DeskThinking working pose="busy" />);

    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
  });

  it("the greeting's desk skips its entrance", () => {
    const { container } = render(
      <DeskMark pose="idle" timeOfDay="evening" welcome animate={false} />,
    );

    expect(container.querySelector(MOVING)).toBeNull();
    expect(container.querySelector('[data-part="lamp-light"]')).not.toBeNull();
  });
});

describe("DeskThinking announcements", () => {
  it("is one status with a fixed name while working", () => {
    render(<DeskThinking working pose="write" />);

    const status = screen.getByRole("status", { name: "Working on your answer" });
    expect(status.textContent).toBe("");
    expect(status.querySelector('[data-slot="desk-mark"]')).toHaveAttribute("aria-hidden", "true");
  });

  it("is hidden when the words beside it already say it", () => {
    render(<DeskThinking working pose="write" decorative />);

    expect(screen.queryByRole("status")).toBeNull();
  });

  it("does not move in a list, where many are marked at once", () => {
    const { container } = render(<DeskThinking working pose="busy" still decorative />);

    expect(mark(container)).toHaveAttribute("data-motion", "still");
    expect(container.querySelector(MOVING)).toBeNull();
  });
});

/**
 * The mark is on screen exactly as long as the work: it stops the instant the
 * turn ends, tucks the chair in for one beat, and is gone.
 */
describe("DeskThinking when the work ends", () => {
  it("settles for a beat and then unmounts", () => {
    vi.useFakeTimers();
    const { container, rerender } = render(<DeskThinking working pose="busy" />);

    rerender(<DeskThinking working={false} pose="busy" />);

    expect(mark(container)).toHaveAttribute("data-pose", "settle");
    expect(container.querySelector('[data-part="desk"]')).not.toHaveClass("animate-desk-shake");
    expect(container.querySelector('[data-part="chair"]')).toHaveClass("animate-desk-settle");
    expect(screen.queryByRole("status")).toBeNull();

    act(() => {
      vi.advanceTimersByTime(DESK_SETTLE_MS);
    });

    expect(container).toBeEmptyDOMElement();
  });

  it("comes back rather than finishing the settle when the work resumes", () => {
    vi.useFakeTimers();
    const { container, rerender } = render(<DeskThinking working pose="write" />);

    rerender(<DeskThinking working={false} pose="write" />);
    rerender(<DeskThinking working pose="write" />);
    act(() => {
      vi.advanceTimersByTime(DESK_SETTLE_MS * 2);
    });

    expect(mark(container)).toHaveAttribute("data-pose", "write");
    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
  });

  it("unmounts at once under reduced motion", () => {
    motion.reduce = true;
    const { container, rerender } = render(<DeskThinking working pose="arrive" />);

    rerender(<DeskThinking working={false} pose="arrive" />);

    expect(container).toBeEmptyDOMElement();
  });

  it("unmounts at once when it was a still mark", () => {
    const { container, rerender } = render(<DeskThinking working pose="busy" still />);

    rerender(<DeskThinking working={false} pose="busy" still />);

    expect(container).toBeEmptyDOMElement();
  });

  it("is never drawn for work that is not happening", () => {
    const { container } = render(<DeskThinking working={false} pose="arrive" />);

    expect(container).toBeEmptyDOMElement();
  });
});

/** The turn in progress: the desk leads the working line and leaves with the turn. */
describe("StreamingTurn working line", () => {
  const noop = () => {};

  function renderTurn(turn: TurnState) {
    return render(<StreamingTurn turn={turn} onDismiss={noop} onAnswer={noop} />);
  }

  it("draws the large desk before anything has arrived", () => {
    vi.useFakeTimers();
    const { container } = renderTurn(initialTurnState("Where is S1?"));

    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
    expect(mark(container)).toHaveAttribute("data-pose", "arrive");
    expect(mark(container)).toHaveClass("h-8");
  });

  it("stops the moment the turn ends and is gone after the settle", () => {
    vi.useFakeTimers();
    const live: TurnState = {
      ...initialTurnState("Where is S1?"),
      status: "streaming",
      segments: [{ kind: "text", text: "S1 is in Dallas.", closed: false }],
    };
    const { container, rerender } = renderTurn(live);

    expect(mark(container)).toHaveAttribute("data-pose", "write");
    expect(screen.getByText("Writing the answer…")).toBeInTheDocument();

    rerender(
      <StreamingTurn
        turn={{
          ...live,
          status: "done",
          segments: [{ kind: "text", text: "S1 is in Dallas.", closed: true }],
        }}
        onDismiss={noop}
        onAnswer={noop}
      />,
    );

    expect(screen.queryByRole("status")).toBeNull();
    expect(screen.queryByText("Writing the answer…")).toBeNull();
    expect(mark(container)).toHaveAttribute("data-pose", "settle");

    act(() => {
      vi.advanceTimersByTime(DESK_SETTLE_MS);
    });

    expect(mark(container)).toBeNull();
  });
});
