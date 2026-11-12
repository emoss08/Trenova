import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState, type PointerEvent, type ReactNode } from "react";

const TARGET_SELECTOR = "[data-glide-target]";

type Glide = {
  x: number;
  y: number;
  width: number;
  height: number;
  shown: boolean;
  travel: boolean;
};

const HIDDEN: Glide = { x: 0, y: 0, width: 0, height: 0, shown: false, travel: false };

/**
 * One hover background for a run of targets (`data-glide-target`): the module
 * rail's tiles, the billing queue's items. It is drawn once and moved rather
 * than lit on each target: it slides from target to target while the pointer
 * stays in the group, appears in place when the pointer arrives, and fades when
 * it leaves. Crossing a gap or a rule keeps it where it is; another control in
 * the group (the rail's user menu, a lane heading) puts it away. A target's
 * flyout, tooltip or menu is portalled out of the group, so pointer travel inside
 * them leaves it on the target that opened them.
 *
 * Targets give up their own hover background so the glide is the only one. In a
 * scrolling list the group goes inside the scrolled content, so the glide scrolls
 * with the target it sits on.
 */
export function HoverGlideGroup({
  className,
  glideClassName,
  children,
}: {
  className?: string;
  /** The glide's own look where the surface differs: its fill and its corner. */
  glideClassName?: string;
  children: ReactNode;
}) {
  const groupRef = useRef<HTMLDivElement>(null);
  const [glide, setGlide] = useState<Glide>(HIDDEN);

  const hide = () =>
    setGlide((current) => (current.shown ? { ...current, shown: false } : current));

  // The target the glide sits on can go without the pointer ever leaving the
  // group: a click swaps the list for another, or the row is removed. Left
  // alone the glide would stay lit over nothing, so it goes with its target.
  const targetRef = useRef<Element | null>(null);
  useEffect(() => {
    if (glide.shown && targetRef.current !== null && !targetRef.current.isConnected) {
      targetRef.current = null;
      hide();
    }
  });

  const handlePointerOver = (event: PointerEvent<HTMLDivElement>) => {
    const group = groupRef.current;
    const target = event.target;
    if (!group || !(target instanceof Element) || !group.contains(target)) return;

    const tile = target.closest(TARGET_SELECTOR);
    if (!tile || !group.contains(tile)) {
      if (target.closest("a, button")) hide();
      return;
    }

    targetRef.current = tile;
    const groupRect = group.getBoundingClientRect();
    const tileRect = tile.getBoundingClientRect();
    const x = tileRect.left - groupRect.left;
    const y = tileRect.top - groupRect.top;

    setGlide((current) =>
      current.shown && current.x === x && current.y === y
        ? current
        : {
            x,
            y,
            width: tileRect.width,
            height: tileRect.height,
            shown: true,
            travel: current.shown,
          },
    );
  };

  return (
    <div
      ref={groupRef}
      onPointerOver={handlePointerOver}
      onPointerLeave={hide}
      className={cn("relative bg-background", className)}
    >
      <span
        aria-hidden
        data-slot="hover-glide"
        data-state={glide.shown ? "shown" : "hidden"}
        className={cn(
          "pointer-events-none absolute top-0 left-0 rounded-md bg-surface-hover duration-200",
          glide.travel ? "transition-[transform,opacity]" : "transition-opacity",
          glideClassName,
        )}
        style={{
          width: glide.width,
          height: glide.height,
          transform: `translate(${glide.x}px, ${glide.y}px)`,
          opacity: glide.shown ? 1 : 0,
        }}
      />
      {children}
    </div>
  );
}
