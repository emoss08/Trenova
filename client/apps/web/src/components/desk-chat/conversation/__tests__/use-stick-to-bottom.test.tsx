import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useStickToBottom } from "../use-stick-to-bottom";

type Box = { scrollHeight: number; clientHeight: number; scrollTop: number };

let resize: (() => void) | null = null;

class FakeResizeObserver {
  constructor(callback: () => void) {
    resize = callback;
  }
  observe() {}
  disconnect() {}
}

type Counts = { replies: number; ownMessages: number; oldest: string | null };

const ONE: Counts = { replies: 1, ownMessages: 1, oldest: "amsg_1" };

function Thread({ box, counts = ONE }: { box: Box; counts?: Counts }) {
  const { scrollRef, away, unread } = useStickToBottom(counts);

  return (
    <>
      <div
        data-testid="scroller"
        ref={(node) => {
          if (node && !Object.hasOwn(node, "scrollHeight")) {
            Object.defineProperty(node, "scrollHeight", { get: () => box.scrollHeight });
            Object.defineProperty(node, "clientHeight", { get: () => box.clientHeight });
            Object.defineProperty(node, "scrollTop", {
              get: () => box.scrollTop,
              set: (value: number) => {
                box.scrollTop = Math.min(value, box.scrollHeight - box.clientHeight);
              },
            });
          }
          scrollRef.current = node;
        }}
      >
        <div />
      </div>
      {away && <span>Jump to latest</span>}
      <output aria-label="Unread">{unread}</output>
    </>
  );
}

/**
 * The jump control says the reader is away from the newest message. The
 * Desk's dock grows and shrinks under the conversation without the scroller
 * ever scrolling, so where the reader is has to be read again on every
 * change in size, not only on a scroll.
 */
describe("useStickToBottom", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", FakeResizeObserver);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    resize = null;
  });

  it("drops the jump control when a resize brings the reader back to the bottom", () => {
    const box: Box = { scrollHeight: 2000, clientHeight: 500, scrollTop: 1500 };
    render(<Thread box={box} />);
    const scroller = screen.getByTestId("scroller");

    fireEvent.wheel(scroller, { deltaY: -100 });
    box.scrollTop = 1200;
    fireEvent.scroll(scroller);
    expect(screen.getByText("Jump to latest")).toBeInTheDocument();

    box.clientHeight = 800;
    act(() => resize?.());

    expect(screen.queryByText("Jump to latest")).toBeNull();
  });

  it("raises the jump control when a resize leaves the reader away from the bottom", () => {
    const box: Box = { scrollHeight: 2000, clientHeight: 500, scrollTop: 1500 };
    render(<Thread box={box} />);
    const scroller = screen.getByTestId("scroller");

    fireEvent.wheel(scroller, { deltaY: -100 });
    box.scrollTop = 1400;
    fireEvent.scroll(scroller);
    expect(screen.queryByText("Jump to latest")).toBeNull();

    box.scrollHeight = 2600;
    act(() => resize?.());

    expect(screen.getByText("Jump to latest")).toBeInTheDocument();
  });
});

/**
 * An earlier page of a long conversation arrives above the reader as they
 * scroll up to it. It holds the questions they asked back then, but it is
 * history: the reader stays where they are and nothing counts as unread.
 */
describe("useStickToBottom with history above", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", FakeResizeObserver);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    resize = null;
  });

  const away = (box: Box, scroller: HTMLElement) => {
    fireEvent.wheel(scroller, { deltaY: -100 });
    box.scrollTop = 200;
    fireEvent.scroll(scroller);
  };

  it("leaves the reader where they are when an earlier page arrives", () => {
    const box: Box = { scrollHeight: 2000, clientHeight: 500, scrollTop: 1500 };
    const counts: Counts = { replies: 3, ownMessages: 3, oldest: "amsg_40" };
    const { rerender } = render(<Thread box={box} counts={counts} />);
    away(box, screen.getByTestId("scroller"));

    box.scrollHeight = 3200;
    rerender(<Thread box={box} counts={{ replies: 8, ownMessages: 8, oldest: "amsg_1" }} />);
    act(() => resize?.());

    expect(box.scrollTop).toBe(200);
    expect(screen.getByLabelText("Unread")).toHaveTextContent("0");
  });

  it("still brings the reader down when they send a message", () => {
    const box: Box = { scrollHeight: 2000, clientHeight: 500, scrollTop: 1500 };
    const counts: Counts = { replies: 3, ownMessages: 3, oldest: "amsg_40" };
    const { rerender } = render(<Thread box={box} counts={counts} />);
    away(box, screen.getByTestId("scroller"));

    box.scrollHeight = 2200;
    rerender(<Thread box={box} counts={{ ...counts, ownMessages: 4 }} />);

    expect(box.scrollTop).toBe(1700);
  });

  it("counts a reply that arrives below while the reader is away", () => {
    const box: Box = { scrollHeight: 2000, clientHeight: 500, scrollTop: 1500 };
    const counts: Counts = { replies: 3, ownMessages: 3, oldest: "amsg_40" };
    const { rerender } = render(<Thread box={box} counts={counts} />);
    away(box, screen.getByTestId("scroller"));

    rerender(<Thread box={box} counts={{ ...counts, replies: 4 }} />);

    expect(screen.getByLabelText("Unread")).toHaveTextContent("1");
    expect(box.scrollTop).toBe(200);
  });

  // Dragging the scrollbar fires no wheel, touch or key: the scroll moving up
  // is what says the reader left.
  it("lets go when the reader drags the scrollbar up", () => {
    const box: Box = { scrollHeight: 2000, clientHeight: 500, scrollTop: 1500 };
    render(<Thread box={box} />);
    const scroller = screen.getByTestId("scroller");

    box.scrollTop = 600;
    fireEvent.scroll(scroller);
    box.scrollHeight = 2600;
    act(() => resize?.());

    expect(box.scrollTop).toBe(600);
  });

  it("stays pinned while content grows under a reader at the bottom", () => {
    const box: Box = { scrollHeight: 2000, clientHeight: 500, scrollTop: 1500 };
    render(<Thread box={box} />);

    box.scrollHeight = 2400;
    act(() => resize?.());

    expect(box.scrollTop).toBe(1900);
  });
});
