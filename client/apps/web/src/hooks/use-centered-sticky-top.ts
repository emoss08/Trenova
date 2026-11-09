import { scrollParent } from "@/lib/dom";
import { useEffect, useRef } from "react";

/**
 * A sticky element held in the middle of what scrolls past it: its `top` is half the room its
 * scroll area has left over, never less than `min`, so the gaps above and below it stay even
 * however tall it is. Written straight to the element as it or the scroll area resizes, so
 * scrolling never re-renders.
 */
export function useCenteredStickyTop<T extends HTMLElement>(min = 16) {
  const ref = useRef<T>(null);

  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    const scroller = scrollParent(element);
    const place = () => {
      const room = (scroller?.clientHeight ?? window.innerHeight) - element.offsetHeight;
      element.style.top = `${Math.max(min, Math.round(room / 2))}px`;
    };

    place();
    const observer = new ResizeObserver(place);
    observer.observe(element);
    if (scroller) observer.observe(scroller);
    window.addEventListener("resize", place);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", place);
    };
  }, [min]);

  return ref;
}
