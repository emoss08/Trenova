import { useRef, useState } from "react";

/**
 * Whether a scrolling area has left its top, for a header that draws its edge only once
 * content slides under it. Hand `ref` to the scrolling element (a ScrollArea's
 * `viewportRef`); it follows that element through remounts.
 */
export function useScrolled(): { scrolled: boolean; ref: (node: HTMLDivElement | null) => void } {
  const [scrolled, setScrolled] = useState(false);
  const stop = useRef<(() => void) | null>(null);

  const ref = (node: HTMLDivElement | null) => {
    stop.current?.();
    stop.current = null;
    if (!node) return;
    const update = () => setScrolled(node.scrollTop > 0);
    update();
    node.addEventListener("scroll", update, { passive: true });
    stop.current = () => node.removeEventListener("scroll", update);
  };

  return { scrolled, ref };
}
