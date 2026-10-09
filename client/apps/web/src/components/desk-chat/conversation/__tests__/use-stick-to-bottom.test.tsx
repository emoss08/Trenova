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

function Thread({ box }: { box: Box }) {
  const { scrollRef, away } = useStickToBottom({ replies: 1, ownMessages: 1 });

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
