import { cn } from "@trenova/shared/lib/utils";
import { useRef, useState, type PointerEvent, type ReactNode } from "react";

const TILE_SELECTOR = "[data-rail-tile]";

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
 * One hover background for a run of rail tiles (`data-rail-tile`), drawn
 * once and moved rather than lit on each tile: it slides from tile to tile
 * while the pointer stays in the group, appears in place when the pointer
 * arrives, and fades when it leaves. Crossing a gap or a rule keeps it where
 * it is; another control in the group (the user menu) puts it away. A tile's
 * flyout and tooltip are portalled out of the group, so pointer travel inside
 * them leaves it on the tile that opened them.
 */
export function RailGlideGroup({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  const groupRef = useRef<HTMLDivElement>(null);
  const [glide, setGlide] = useState<Glide>(HIDDEN);

  const hide = () =>
    setGlide((current) => (current.shown ? { ...current, shown: false } : current));

  const handlePointerOver = (event: PointerEvent<HTMLDivElement>) => {
    const group = groupRef.current;
    const target = event.target;
    if (!group || !(target instanceof Element) || !group.contains(target)) return;

    const tile = target.closest(TILE_SELECTOR);
    if (!tile || !group.contains(tile)) {
      if (target.closest("a, button")) hide();
      return;
    }

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
      className={cn("relative", className)}
    >
      <span
        aria-hidden
        data-slot="rail-glide"
        data-state={glide.shown ? "shown" : "hidden"}
        className={cn(
          "bg-surface-hover pointer-events-none absolute top-0 left-0 rounded-md duration-200",
          glide.travel ? "transition-[transform,opacity]" : "transition-opacity",
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
