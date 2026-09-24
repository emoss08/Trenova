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
 * The lamp tells the moment apart by its light and its head: the light
 * coming on, breathing, lines lit on the desk, and one motion for each kind
 * of tool, each with its own things on the desk and nothing else's.
 */
describe("DeskMark poses", () => {
  const EXTRAS = ["dashes", "tick", "card", "helper", "waves"] as const;

  function extrasShown(container: HTMLElement) {
    return EXTRAS.filter((name) => part(container, name) !== null);
  }

  it("switches the light on while the question is being checked", () => {
    const { container } = render(<DeskMark pose="start" />);

    expect(part(container, "beam")).toHaveClass("animate-desk-light-on");
    expect(part(container, "glow")).toHaveClass("animate-desk-light-on");
    expect(part(container, "bulb")).toHaveClass("animate-desk-light-on");
    expect(extrasShown(container)).toEqual([]);
  });

  it("breathes its light while the model thinks, with nothing on the desk", () => {
    const { container } = render(<DeskMark pose="think" />);

    expect(part(container, "cone")).toHaveClass("animate-desk-breathe-cone");
    expect(part(container, "pool")).toHaveClass("animate-desk-breathe-pool");
    expect(part(container, "head")?.getAttribute("class") ?? "").not.toContain("animate-");
    expect(extrasShown(container)).toEqual([]);
  });

  it("lights lines on the desk one after another while the answer arrives", () => {
    const { container } = render(<DeskMark pose="write" />);

    const lines = part(container, "dashes")?.querySelectorAll("rect") ?? [];
    expect(lines).toHaveLength(3);
    expect(lines[0]).toHaveClass("animate-desk-dash-1");
    expect(lines[1]).toHaveClass("animate-desk-dash-2");
    expect(lines[2]).toHaveClass("animate-desk-dash-3");
    expect(part(container, "glow")).toHaveClass("opacity-0");
    expect(extrasShown(container)).toEqual(["dashes"]);
  });

  it("cuts its light out and shakes its head while a reply starts over", () => {
    const { container } = render(<DeskMark pose="retry" />);

    expect(part(container, "head")).toHaveClass("animate-desk-shake");
    expect(part(container, "beam")).toHaveClass("animate-desk-light-retry");
    expect(part(container, "glow")).toHaveClass("animate-desk-light-retry");
    expect(part(container, "bulb")).toHaveClass("animate-desk-light-retry");
  });

  it.each<[DeskPose, string, string, (typeof EXTRAS)[number] | null]>([
    ["lookup", "animate-desk-scan", "animate-desk-scan-pool", null],
    ["discover", "animate-desk-hunt", "animate-desk-hunt-pool", null],
    ["navigate", "animate-desk-travel", "animate-desk-travel-pool", null],
    ["change", "animate-desk-stamp", "animate-desk-stamp-pool", "tick"],
    ["present", "", "animate-desk-present-pool", "card"],
    ["ask", "animate-desk-turn", "animate-desk-away-pool", null],
    ["delegate", "", "animate-desk-handoff-pool", "helper"],
    ["web", "animate-desk-lookout", "animate-desk-away-pool", "waves"],
  ])("draws a %s call by its head, its pool and what is on the desk", (pose, head, pool, extra) => {
    const { container } = render(<DeskMark pose={pose} />);

    if (head) {
      expect(part(container, "head")).toHaveClass(head);
    } else {
      expect(part(container, "head")?.getAttribute("class") ?? "").not.toContain("animate-");
    }
    expect(part(container, "pool")).toHaveClass(pool);
    expect(extrasShown(container)).toEqual(extra ? [extra] : []);
  });

  it("widens and narrows its beam while discovering", () => {
    const { container } = render(<DeskMark pose="discover" />);

    expect(part(container, "cone")).toHaveClass("animate-desk-widen");
  });

  it("stamps a tick into the light for a change", () => {
    const { container } = render(<DeskMark pose="change" />);

    expect(part(container, "tick")).toHaveClass("animate-desk-tick");
  });

  it("raises a card in the light to present something", () => {
    const { container } = render(<DeskMark pose="present" />);

    expect(part(container, "card")).toHaveClass("animate-desk-card");
  });

  it("turns to the reader, shortens its beam and blinks to ask", () => {
    const { container } = render(<DeskMark pose="ask" />);

    expect(part(container, "cone")).toHaveClass("scale-y-60");
    expect(part(container, "bulb")).toHaveClass("animate-desk-blink");
  });

  it("hands its light to a second lamp to delegate", () => {
    const { container } = render(<DeskMark pose="delegate" />);

    expect(part(container, "helper")).toHaveClass("animate-desk-helper");
    expect(part(container, "helper-beam")).toHaveClass("animate-desk-helper-light");
    expect(part(container, "helper-bulb")).toHaveClass("animate-desk-helper-light");
    expect(part(container, "beam")).toHaveClass("animate-desk-handoff-light");
  });

  it("sends three waves out past the desk to read the web", () => {
    const { container } = render(<DeskMark pose="web" />);

    const waves = part(container, "waves")?.querySelectorAll("path") ?? [];
    expect(waves).toHaveLength(3);
    expect(waves[0]).toHaveClass("animate-desk-wave-1");
    expect(waves[2]).toHaveClass("animate-desk-wave-3");
    expect(part(container, "cone")).toHaveClass("scale-y-60");
  });

  it("dips its head, puts the light out and fades when the turn is done", () => {
    const { container } = render(<DeskMark pose="done" />);

    expect(container.querySelector(DESK)).toHaveClass("animate-desk-fade");
    expect(part(container, "head")).toHaveClass("animate-desk-nod");
    expect(part(container, "beam")).toHaveClass("animate-desk-light-off");
    expect(part(container, "glow")).toHaveClass("animate-desk-light-off");
    expect(part(container, "bulb")).toHaveClass("animate-desk-light-off");
    expect(part(container, "fault")).toBeNull();
    expect(extrasShown(container)).toEqual([]);
  });

  it("flickers out, droops and shows a red bulb when the turn fails", () => {
    const { container } = render(<DeskMark pose="failed" />);

    expect(container.querySelector(DESK)).toHaveClass("animate-desk-fade");
    expect(part(container, "head")).toHaveClass("animate-desk-droop");
    expect(part(container, "bulb")).toHaveClass("animate-desk-light-fail");
    expect(part(container, "fault")).toHaveClass("fill-desk-failed", "animate-desk-fault");
  });

  it("closes on an amber light when a proposed write is waiting on the reader", () => {
    const { container } = render(<DeskMark pose="await" />);

    expect(container.querySelector(DESK)).toHaveClass("animate-desk-fade");
    expect(part(container, "lamp")).toHaveClass(
      "[--desk-accent:var(--desk-await)]",
      "[--desk-light:var(--desk-await)]",
    );
    expect(part(container, "bulb")?.getAttribute("class") ?? "").not.toContain("animate-");
    expect(part(container, "fault")).toBeNull();
  });

  it("only turns amber for a write waiting on the reader", () => {
    const { container } = render(<DeskMark pose="done" />);

    expect(part(container, "lamp")?.getAttribute("class") ?? "").not.toContain("desk-await");
  });

  it("is drawn at the height of a line of text, in the agent's light", () => {
    const { container } = render(<DeskMark pose="lookup" />);
    const mark = container.querySelector(DESK);

    expect(mark).toHaveClass("h-4", "w-5", "ui-desk-mark");
    expect(mark).toHaveAttribute("aria-hidden", "true");
    expect(part(container, "cone")).toHaveClass("fill-desk-accent");
    expect(part(container, "pool")).toHaveClass("fill-desk-light");
    expect(part(container, "bulb")).toHaveClass("fill-desk-light");
    expect(part(container, "desk")).toHaveClass("fill-desk-lamp");
  });

  it("keeps two marks on one page from sharing a clip path", () => {
    const { container } = render(
      <>
        <DeskMark pose="delegate" />
        <DeskMark pose="delegate" />
      </>,
    );
    const ids = [...container.querySelectorAll("clipPath")].map((clip) => clip.id);
    const used = [...container.querySelectorAll("[clip-path]")].map((node) =>
      node.getAttribute("clip-path"),
    );

    expect(new Set(ids).size).toBe(2);
    expect(new Set(used)).toEqual(new Set(ids.map((id) => `url(#${id})`)));
  });
});

/**
 * Reduced motion keeps each pose as one still frame of the same moment, so
 * the lamp still says what is happening without anything moving.
 */
describe("DeskMark still", () => {
  const ALL: DeskPose[] = [
    "start",
    "think",
    "write",
    "retry",
    "lookup",
    "discover",
    "navigate",
    "change",
    "present",
    "ask",
    "delegate",
    "web",
    "done",
    "await",
    "failed",
  ];

  it.each(ALL)("draws %s without anything that moves", (pose) => {
    const { container } = render(<DeskMark pose={pose} animate={false} />);

    expect(container.querySelector(DESK)).toHaveAttribute("data-motion", "still");
    expect(container.querySelector(DESK)).toHaveAttribute("data-pose", pose);
    expect(container.querySelector(MOVING)).toBeNull();
  });

  it("keeps what each moment puts on the desk, lit", () => {
    const write = render(<DeskMark pose="write" animate={false} />).container;
    expect(part(write, "dashes")?.querySelectorAll("rect.opacity-0")).toHaveLength(0);
    expect(part(write, "dashes")?.querySelectorAll("rect")).toHaveLength(3);

    const change = render(<DeskMark pose="change" animate={false} />).container;
    expect(part(change, "tick")).not.toBeNull();

    const present = render(<DeskMark pose="present" animate={false} />).container;
    expect(part(present, "card")).not.toBeNull();

    const web = render(<DeskMark pose="web" animate={false} />).container;
    expect(part(web, "waves")?.querySelectorAll("path.opacity-0")).toHaveLength(0);
  });

  it("holds the head where the motion is about", () => {
    const ask = render(<DeskMark pose="ask" animate={false} />).container;
    expect(part(ask, "head")).toHaveClass("-rotate-58");
    expect(part(ask, "glow")).toHaveClass("opacity-0");

    const web = render(<DeskMark pose="web" animate={false} />).container;
    expect(part(web, "head")).toHaveClass("-rotate-96");
  });

  it("shows the second lamp lit and the first one's light handed over", () => {
    const { container } = render(<DeskMark pose="delegate" animate={false} />);

    expect(part(container, "helper")).not.toBeNull();
    expect(part(container, "beam")).toHaveClass("opacity-0");
    expect(part(container, "helper-bulb")?.getAttribute("class") ?? "").not.toContain("opacity-0");
  });

  it("draws the closing beats as their last frame", () => {
    const done = render(<DeskMark pose="done" animate={false} />).container;
    expect(part(done, "bulb")).toHaveClass("opacity-0");
    expect(part(done, "beam")).toHaveClass("opacity-0");

    const failed = render(<DeskMark pose="failed" animate={false} />).container;
    expect(part(failed, "fault")).toHaveClass("opacity-100");
    expect(part(failed, "head")).toHaveClass("rotate-14");
  });
});

describe("DeskThinking with reduced motion", () => {
  it.each<DeskPose>(["start", "think", "write", "lookup", "web"])("draws %s still", (pose) => {
    motion.reduce = true;
    const { container } = render(<DeskThinking pose={pose} />);

    expect(container.querySelector(DESK)).toHaveAttribute("data-motion", "still");
    expect(container.querySelector(DESK)).toHaveAttribute("data-pose", pose);
    expect(container.querySelector(MOVING)).toBeNull();
  });

  it("still announces the work", () => {
    motion.reduce = true;
    render(<DeskThinking pose="lookup" />);

    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
  });
});

describe("DeskThinking announcements", () => {
  it("is one status with a fixed name while the work runs", () => {
    render(<DeskThinking pose="write" />);

    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
  });

  it.each<DeskPose>(["done", "await", "failed"])(
    "says nothing while it closes on %s, because the work is already over",
    (pose) => {
      const { container } = render(<DeskThinking pose={pose} />);

      expect(screen.queryByRole("status")).toBeNull();
      expect(container.querySelector('[data-slot="desk-thinking"]')).toHaveAttribute(
        "aria-hidden",
        "true",
      );
    },
  );
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

/** The turn in progress: the lamp leads the working line and leaves with the turn. */
describe("StreamingTurn working line", () => {
  const noop = () => {};
  const startedAt = 0;

  function renderTurn(turn: TurnState) {
    return render(<StreamingTurn turn={turn} onDismiss={noop} onAnswer={noop} />);
  }

  it("draws the lamp switching on beside the words before anything has arrived", () => {
    vi.useFakeTimers();
    const { container } = renderTurn(initialTurnState("Where is S1?", null, { startedAt }));

    expect(screen.getByRole("status", { name: "Working on your answer" })).toBeInTheDocument();
    expect(screen.getByText("Checking the question…")).toBeInTheDocument();
    const marks = container.querySelectorAll(DESK);
    expect(marks).toHaveLength(1);
    expect(marks[0]).toHaveAttribute("data-pose", "start");
    expect(marks[0]).toHaveClass("h-4");
  });

  it("closes the moment the turn ends and is gone after the closing beat", () => {
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
    expect(container.querySelector(DESK)).toHaveAttribute("data-pose", "done");

    act(() => {
      vi.advanceTimersByTime(DESK_SETTLE_MS);
    });

    expect(container.querySelector(DESK)).toBeNull();
  });

  it("draws the running call's kind beside the words that name it", () => {
    const { container } = renderTurn({
      ...initialTurnState("Where is S1?", null, { startedAt }),
      status: "working",
      segments: [
        {
          kind: "tool",
          callId: "c1",
          name: "update_shipment",
          arguments: {},
          status: "running",
          content: "",
        },
      ],
    });

    expect(container.querySelector(DESK)).toHaveAttribute("data-pose", "change");
  });

  it("closes on the failure when the turn errors partway through a reply, then is gone", () => {
    vi.useFakeTimers();
    const live: TurnState = {
      ...initialTurnState("Where is S1?", null, { startedAt }),
      status: "streaming",
      segments: [{ kind: "text", text: "S1 is", closed: false }],
    };
    const { container, rerender } = renderTurn(live);

    rerender(
      <StreamingTurn
        turn={{ ...live, status: "error", error: "Provider down" }}
        onDismiss={noop}
        onAnswer={noop}
      />,
    );

    expect(container.querySelector(DESK)).toHaveAttribute("data-pose", "failed");

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
    const { container } = render(<DeskThinking pose="start" />);

    expect(container.querySelector(MOVING)).toBeNull();
  });
});
