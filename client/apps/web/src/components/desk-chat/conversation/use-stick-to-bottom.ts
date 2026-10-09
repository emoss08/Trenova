import { useCallback, useLayoutEffect, useRef, useState } from "react";

/** How far from the bottom still counts as reading the newest message. */
const NEAR_BOTTOM = 40;
/** How far up the reader has to be before the jump control offers itself. */
const AWAY = 160;

/**
 * Keeps a conversation pinned to its newest message while the reader is
 * there, and lets go the moment they scroll up by wheel, touch or keys. While
 * they are away it says so, and counts the replies that arrive meanwhile, so
 * the jump control can say what it would take them to.
 */
export function useStickToBottom({
  replies,
  ownMessages,
}: {
  /** How many replies the conversation holds; a rise while away is unread. */
  replies: number;
  /** How many messages the reader has sent; sending one always brings them down. */
  ownMessages: number;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const stick = useRef(true);
  const [away, setAway] = useState(false);
  const [unread, setUnread] = useState(0);
  const seenReplies = useRef(replies);
  const seenOwn = useRef(ownMessages);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element) {
      return;
    }
    // Where the reader is, read from the scroller as it is now. A scroll
    // asks, and so does any change in size: the dock under the conversation
    // grows and shrinks (a case, its checklist, the queue) and content
    // settles without the scroller ever firing a scroll, which left the jump
    // control up over a reader sitting at the bottom.
    const measure = () => {
      const distance = element.scrollHeight - element.scrollTop - element.clientHeight;
      if (distance < NEAR_BOTTOM) {
        stick.current = true;
        setUnread(0);
      }
      setAway(distance > AWAY);
    };
    const pin = () => {
      if (stick.current) {
        element.scrollTop = element.scrollHeight;
      }
    };
    const onResize = () => {
      pin();
      measure();
    };
    const letGo = () => {
      stick.current = false;
    };
    const onWheel = (event: WheelEvent) => {
      if (event.deltaY < 0) {
        letGo();
      }
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "ArrowUp" || event.key === "PageUp" || event.key === "Home") {
        letGo();
      }
    };
    element.addEventListener("scroll", measure);
    element.addEventListener("wheel", onWheel, { passive: true });
    element.addEventListener("touchmove", letGo, { passive: true });
    element.addEventListener("keydown", onKeyDown);
    const observer = new ResizeObserver(onResize);
    observer.observe(element);
    if (element.firstElementChild) {
      observer.observe(element.firstElementChild);
    }
    onResize();

    return () => {
      observer.disconnect();
      element.removeEventListener("scroll", measure);
      element.removeEventListener("wheel", onWheel);
      element.removeEventListener("touchmove", letGo);
      element.removeEventListener("keydown", onKeyDown);
    };
  }, []);

  useLayoutEffect(() => {
    if (ownMessages > seenOwn.current) {
      stick.current = true;
    } else if (!stick.current && replies > seenReplies.current) {
      setUnread((count) => count + (replies - seenReplies.current));
    }
    seenOwn.current = ownMessages;
    seenReplies.current = replies;
    const element = scrollRef.current;
    if (element && stick.current) {
      element.scrollTop = element.scrollHeight;
    }
  }, [ownMessages, replies]);

  const jumpToLatest = useCallback(() => {
    const element = scrollRef.current;
    if (!element) {
      return;
    }
    stick.current = true;
    setUnread(0);
    element.scrollTo({ top: element.scrollHeight, behavior: "smooth" });
  }, []);

  return { scrollRef, away, unread, jumpToLatest };
}
