import { act, cleanup, render } from "@testing-library/react";
import { useRef } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useChapterNav, type DeskChapterNav } from "../use-chapter-nav";

const originalScroll = Element.prototype.scrollIntoView;

/* jsdom has no scrolling: record which heading was asked to come into view. */
function recordScrolls() {
  const scrolled = vi.fn();
  Element.prototype.scrollIntoView = function (this: Element, options?: unknown) {
    scrolled(this.getAttribute("data-chapter-turn"), options);
  } as Element["scrollIntoView"];
  return scrolled;
}

afterEach(() => {
  cleanup();
  Element.prototype.scrollIntoView = originalScroll;
});

type Older = { has: boolean; loading: boolean; load: () => void };

function Conversation({
  turns,
  older,
  onNav,
}: {
  turns: string[];
  older: Older;
  onNav: (nav: DeskChapterNav) => void;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const nav = useChapterNav({ scrollRef, older, version: turns.length, smooth: false });
  onNav(nav);

  return (
    <div ref={scrollRef}>
      {turns.map((turn) => (
        <div key={turn} className="dk-r dk-chap">
          <div className="dk-c" data-chapter-turn={turn}>
            {turn}
          </div>
        </div>
      ))}
    </div>
  );
}

describe("useChapterNav.reveal", () => {
  it("scrolls the picked chapter's heading to the top", () => {
    const scrolled = recordScrolls();
    const held: { nav: DeskChapterNav | null } = { nav: null };
    render(
      <Conversation
        turns={["atrn_1", "atrn_2"]}
        older={{ has: false, loading: false, load: vi.fn() }}
        onNav={(value) => {
          held.nav = value;
        }}
      />,
    );

    act(() => held.nav?.reveal("atrn_2"));

    expect(scrolled).toHaveBeenCalledWith("atrn_2", { block: "start", behavior: "auto" });
    expect(held.nav?.current).toBe("atrn_2");
  });

  it("reads earlier pages until a chapter above them is there, then scrolls to it", () => {
    const scrolled = recordScrolls();
    const load = vi.fn();
    const held: { nav: DeskChapterNav | null } = { nav: null };
    const onNav = (value: DeskChapterNav) => {
      held.nav = value;
    };
    const { rerender } = render(
      <Conversation turns={["atrn_5"]} older={{ has: true, loading: false, load }} onNav={onNav} />,
    );

    act(() => held.nav?.reveal("atrn_1"));
    expect(load).toHaveBeenCalledTimes(1);
    expect(scrolled).not.toHaveBeenCalled();

    rerender(
      <Conversation
        turns={["atrn_1", "atrn_5"]}
        older={{ has: false, loading: false, load }}
        onNav={onNav}
      />,
    );

    expect(scrolled).toHaveBeenCalledWith("atrn_1", { block: "start", behavior: "auto" });
  });
});
