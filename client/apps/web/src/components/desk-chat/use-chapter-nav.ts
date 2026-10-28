import { useCallback, useEffect, useRef, useState, type RefObject } from "react";

/** How far below the top of the conversation a heading may sit and still be the chapter in view. */
const IN_VIEW_PX = 96;

export const CHAPTER_ATTRIBUTE = "data-chapter-turn";

/** How the conversation moves between its chapters. */
export type DeskChapterNav = {
  /** The turn whose chapter is in view, or null above the first one. */
  current: string | null;
  /** Brings a chapter's heading into view, reading earlier pages first when it is above them. */
  reveal: (turnId: string) => void;
};

type OlderPages = { has: boolean; loading: boolean; load: () => void };

function headingOf(scroller: HTMLElement, turnId: string): HTMLElement | null {
  for (const heading of scroller.querySelectorAll<HTMLElement>(`[${CHAPTER_ATTRIBUTE}]`)) {
    if (heading.getAttribute(CHAPTER_ATTRIBUTE) === turnId) {
      return heading;
    }
  }
  return null;
}

/** The chapter whose heading is the last one at or above the top of the view. */
export function chapterInView(scroller: HTMLElement): string | null {
  const top = scroller.getBoundingClientRect().top + IN_VIEW_PX;
  let current: string | null = null;
  for (const heading of scroller.querySelectorAll<HTMLElement>(`[${CHAPTER_ATTRIBUTE}]`)) {
    if (heading.getBoundingClientRect().top > top) {
      break;
    }
    current = heading.getAttribute(CHAPTER_ATTRIBUTE);
  }
  return current;
}

/**
 * Follows which chapter the person is reading as they scroll, and scrolls to
 * one they pick. A chapter above the pages read so far is reached by reading
 * earlier pages until its heading is there.
 */
export function useChapterNav({
  scrollRef,
  older,
  version,
  smooth,
}: {
  scrollRef: RefObject<HTMLElement | null>;
  older: OlderPages;
  /** Changes whenever the conversation's rows do, so a waiting reveal looks again. */
  version: number;
  smooth: boolean;
}): DeskChapterNav {
  const [current, setCurrent] = useState<string | null>(null);
  const pending = useRef<string | null>(null);
  const { has: hasOlder, loading: loadingOlder, load: loadOlder } = older;

  useEffect(() => {
    const scroller = scrollRef.current;
    if (!scroller) {
      return;
    }
    let frame = 0;
    const measure = () => {
      frame = 0;
      setCurrent(chapterInView(scroller));
    };
    const onScroll = () => {
      if (frame === 0) {
        frame = window.requestAnimationFrame(measure);
      }
    };
    measure();
    scroller.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      scroller.removeEventListener("scroll", onScroll);
      if (frame !== 0) {
        window.cancelAnimationFrame(frame);
      }
    };
  }, [scrollRef, version]);

  const show = useCallback(
    (heading: HTMLElement, turnId: string) => {
      pending.current = null;
      heading.scrollIntoView({ block: "start", behavior: smooth ? "smooth" : "auto" });
      setCurrent(turnId);
    },
    [smooth],
  );

  const reveal = useCallback(
    (turnId: string) => {
      const scroller = scrollRef.current;
      if (!scroller) {
        return;
      }
      const heading = headingOf(scroller, turnId);
      if (heading) {
        show(heading, turnId);
        return;
      }
      if (!hasOlder) {
        pending.current = null;
        return;
      }
      pending.current = turnId;
      if (!loadingOlder) {
        loadOlder();
      }
    },
    [hasOlder, loadOlder, loadingOlder, scrollRef, show],
  );

  useEffect(() => {
    const turnId = pending.current;
    const scroller = scrollRef.current;
    if (turnId === null || !scroller || loadingOlder) {
      return;
    }
    const heading = headingOf(scroller, turnId);
    if (heading) {
      show(heading, turnId);
    } else if (hasOlder) {
      loadOlder();
    } else {
      pending.current = null;
    }
  }, [hasOlder, loadOlder, loadingOlder, scrollRef, show, version]);

  return { current, reveal };
}
