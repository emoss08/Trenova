import { act, render, renderHook, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ComposerHints } from "../composer-hints";
import { OutcomeIcon } from "../decision-chrome";
import { LiveReplyLabel } from "../live-reply-label";
import { StreamingTurn } from "../streaming-turn";
import { initialTurnState, type TurnState } from "../turn-stream";
import type { DeskPose } from "../voice/desk-pose";
import {
  DESK_SETTLE_MS,
  DeskMark,
  DeskThinking,
  useThinkingPresence,
} from "../voice/desk-thinking";

const motion = vi.hoisted(() => ({ reduce: false }));

vi.mock("motion/react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("motion/react")>();
  return { ...actual, useReducedMotion: () => motion.reduce };
});

const MOVING = '[class*="animate-desk-"]';
const DESK = '[data-slot="desk-mark"]';

function part(container: HTMLElement, name: string) {
  return container.querySelector(`[data-part="${name}"]`);
}

beforeEach(() => {
  motion.reduce = false;
});

afterEach(() => {
  vi.useRealTimers();
});

/**
 * The screen tells the phase apart, frame by frame: dots while the model
 * thinks, lines scrolling past while a tool runs, lines written onto a
 * glowing screen while the answer arrives.
 */
describe("DeskMark poses", () => {
  it("sits down and puts thinking dots on the screen while waiting on the model", () => {
    const { container } = render(<DeskMark pose="arrive" />);

    expect(part(container, "seat")).toHaveClass("animate-desk-sit");
    expect(part(container, "figure")).toHaveClass("animate-desk-sit-figure");
    const dots = part(container, "thinking")?.querySelectorAll("circle") ?? [];
    expect(dots).toHaveLength(3);
    expect(dots[0]).toHaveClass("animate-desk-think-1");
    expect(dots[2]).toHaveClass("animate-desk-think-3");
    expect(part(container, "scrolling")).toBeNull();
    expect(part(container, "writing")).toBeNull();
    expect(part(container, "arms")).not.toHaveClass("animate-desk-type");
  });

  it("types and scrolls the screen while a tool runs", () => {
    const { container } = render(<DeskMark pose="busy" />);

    expect(part(container, "arms")).toHaveClass("animate-desk-type");
    expect(part(container, "scrolling")).toHaveClass("animate-desk-scroll");
    expect(part(container, "thinking")).toBeNull();
    expect(part(container, "glow")).not.toHaveClass("animate-desk-glow");
  });

  it("writes lines onto a glowing screen while the answer streams", () => {
    const { container } = render(<DeskMark pose="write" />);

    const lines = part(container, "writing")?.querySelectorAll("rect") ?? [];
    expect(lines).toHaveLength(3);
    expect(lines[0]).toHaveClass("animate-desk-write-1");
    expect(lines[1]).toHaveClass("animate-desk-write-2");
    expect(lines[2]).toHaveClass("animate-desk-write-3");
    expect(part(container, "glow")).toHaveClass("animate-desk-glow");
    expect(part(container, "arms")).toHaveClass("animate-desk-type");
  });

  it("pushes back, clears the screen and fades when the turn is over", () => {
    const { container } = render(<DeskMark pose="settle" />);

    expect(container.querySelector(DESK)).toHaveClass("animate-desk-fade");
    expect(part(container, "seat")).toHaveClass("animate-desk-settle");
    expect(part(container, "seat")).not.toHaveClass("animate-desk-sit");
    expect(part(container, "thinking")).toBeNull();
    expect(part(container, "scrolling")).toBeNull();
    expect(part(container, "writing")).toBeNull();
  });

  it("is drawn at the height of a line of text, in the agent's accent", () => {
    const { container } = render(<DeskMark pose="busy" />);
    const mark = container.querySelector(DESK);

    expect(mark).toHaveClass("h-4", "w-5", "ui-desk-mark");
    expect(mark).toHaveAttribute("aria-hidden", "true");
    expect(part(container, "figure")).toHaveClass("fill-desk-figure");
    expect(part(container, "screen")).toHaveClass("fill-desk-accent");
  });

  it("keeps two marks on one page from sharing a clip path", () => {
    const { container } = render(
      <>
        <DeskMark pose="busy" />
        <DeskMark pose="busy" />
      </>,
    );
    const ids = [...container.querySelectorAll("clipPath")].map((clip) => clip.id);

    expect(new Set(ids).size).toBe(2);
  });
});

/**
 * Reduced motion keeps each pose as one still frame of the same moment, so
 * the screen still says what is happening without anything moving.
 */
describe("DeskThinking with reduced motion", () => {
  it.each<[DeskPose, string]>([
    ["arrive", "thinking"],
    ["busy", "scrolling"],
    ["write", "writing"],
  ])("draws %s still, with its screen", (pose, screenPart) => {
    motion.reduce = true;
    const { container } = render(<DeskThinking pose={pose} />);

    expect(container.querySelector(DESK)).toHaveAttribute("data-motion", "still");
    expect(container.querySelector(DESK)).toHaveAttribute("data-pose", pose);
    expect(container.querySelector(MOVING)).toBeNull();
    expect(part(container, screenPart)).not.toBeNull();
  });

  it("draws the writing screen lit rather than dark", () => {
    motion.reduce = true;
    const { container } = render(<DeskThinking pose="write" />);

    expect(part(container, "glow")).toHaveClass("opacity-(--desk-glow)");
  });

  it("still announces the work", () => {
    motion.reduce = true;
    render(<DeskThinking pose="busy" />);

    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
  });
});

describe("DeskThinking announcements", () => {
  it("is one status with a fixed name while the work runs", () => {
    render(<DeskThinking pose="write" />);

    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
  });

  it("says nothing while it settles, because the work is already over", () => {
    const { container } = render(<DeskThinking pose="settle" />);

    expect(screen.queryByRole("status")).toBeNull();
    expect(container.querySelector('[data-slot="desk-thinking"]')).toHaveAttribute(
      "aria-hidden",
      "true",
    );
  });
});

/**
 * The mark is on screen exactly as long as the work: it takes the closing
 * pose for one beat when the work stops, and is gone.
 */
describe("useThinkingPresence", () => {
  it("settles for a beat and then is gone", () => {
    vi.useFakeTimers();
    const { result, rerender } = renderHook(
      ({ working }: { working: boolean }) => useThinkingPresence(working, true),
      { initialProps: { working: true } },
    );

    expect(result.current).toBe("working");
    rerender({ working: false });
    expect(result.current).toBe("settling");

    act(() => {
      vi.advanceTimersByTime(DESK_SETTLE_MS);
    });

    expect(result.current).toBe("gone");
  });

  it("comes back rather than finishing the settle when the work resumes", () => {
    vi.useFakeTimers();
    const { result, rerender } = renderHook(
      ({ working }: { working: boolean }) => useThinkingPresence(working, true),
      { initialProps: { working: true } },
    );

    rerender({ working: false });
    rerender({ working: true });
    act(() => {
      vi.advanceTimersByTime(DESK_SETTLE_MS * 2);
    });

    expect(result.current).toBe("working");
  });

  it("is gone at once when there is no settle to show", () => {
    const { result, rerender } = renderHook(
      ({ working }: { working: boolean }) => useThinkingPresence(working, false),
      { initialProps: { working: true } },
    );

    rerender({ working: false });

    expect(result.current).toBe("gone");
  });

  it("is never there for work that is not happening", () => {
    const { result } = renderHook(() => useThinkingPresence(false, true));

    expect(result.current).toBe("gone");
  });
});

/** The turn in progress: the desk leads the working line and leaves with the turn. */
describe("StreamingTurn working line", () => {
  const noop = () => {};
  const startedAt = 0;

  function renderTurn(turn: TurnState) {
    return render(<StreamingTurn turn={turn} onDismiss={noop} onAnswer={noop} />);
  }

  it("draws the small desk beside the words before anything has arrived", () => {
    vi.useFakeTimers();
    const { container } = renderTurn(initialTurnState("Where is S1?", null, { startedAt }));

    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
    expect(screen.getByText("Checking the question…")).toBeInTheDocument();
    const marks = container.querySelectorAll(DESK);
    expect(marks).toHaveLength(1);
    expect(marks[0]).toHaveAttribute("data-pose", "arrive");
    expect(marks[0]).toHaveClass("h-4");
  });

  it("stops the moment the turn ends and is gone after the settle", () => {
    vi.useFakeTimers();
    const live: TurnState = {
      ...initialTurnState("Where is S1?", null, { startedAt }),
      status: "streaming",
      segments: [{ kind: "text", text: "S1 is in Dallas.", closed: false }],
    };
    const { container, rerender } = renderTurn(live);

    expect(container.querySelector(DESK)).toHaveAttribute("data-pose", "write");
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
    expect(container.querySelector(DESK)).toHaveAttribute("data-pose", "settle");

    act(() => {
      vi.advanceTimersByTime(DESK_SETTLE_MS);
    });

    expect(container.querySelector(DESK)).toBeNull();
  });

  it("is gone the moment the turn ends under reduced motion", () => {
    motion.reduce = true;
    const live: TurnState = {
      ...initialTurnState("Where is S1?", null, { startedAt }),
      status: "streaming",
      segments: [{ kind: "text", text: "S1 is", closed: false }],
    };
    const { container, rerender } = renderTurn(live);

    expect(container.querySelector(DESK)).toHaveAttribute("data-motion", "still");

    rerender(<StreamingTurn turn={{ ...live, status: "done" }} onDismiss={noop} onAnswer={noop} />);

    expect(container.querySelector(DESK)).toBeNull();
  });
});

/**
 * The desk is drawn on the working line and nowhere else. Every other
 * surface that says work is still going keeps the small breathing dot.
 */
describe("the desk appears only on the working line", () => {
  it("leaves the composer's replying hint with the dot", () => {
    const { container } = render(
      <ComposerHints
        kind="replying"
        issue={null}
        onDismissIssue={() => {}}
        canMention
        canAttach
        canDictate
      />,
    );

    expect(screen.getByText("Replying. You can draft your next message.")).toBeInTheDocument();
    expect(container.querySelector(DESK)).toBeNull();
    expect(container.querySelector(".animate-breathe")).not.toBeNull();
  });

  it("leaves a conversation row's live label with a still dot", () => {
    const { container } = render(<LiveReplyLabel />);

    expect(screen.getByText("Writing a reply")).toBeInTheDocument();
    expect(container.querySelector(DESK)).toBeNull();
    expect(container.querySelector(".animate-breathe")).toBeNull();
    expect(container.querySelector(".rounded-full")).not.toBeNull();
  });

  it("leaves a running decision with the dot", () => {
    const { container } = render(<OutcomeIcon state="running" />);

    expect(container.querySelector(DESK)).toBeNull();
    expect(container.querySelector(".animate-breathe")).not.toBeNull();
  });

  it("draws nothing under reduced motion that the global rule would have to stop", () => {
    motion.reduce = true;
    const { container } = render(<DeskThinking pose="arrive" />);

    expect(container.querySelector(MOVING)).toBeNull();
  });
});
