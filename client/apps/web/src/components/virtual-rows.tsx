import { cn } from "@trenova/shared/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useRef, type ReactNode } from "react";

export type VirtualRow = {
  key: string;
  render: () => ReactNode;
};

type VirtualRowsProps = {
  rows: readonly VirtualRow[];
  /** A guess at a row's height before it is measured. */
  estimateSize?: number;
  /** The scroll container's height before it is measured; sets the first window. */
  initialHeight?: number;
  overscan?: number;
  /** Classes on the scroll container: its height cap lives here. */
  className?: string;
  rowClassName?: string;
  /** Shown in place of the list when there are no rows. */
  empty?: ReactNode;
  "aria-label"?: string;
};

/**
 * A bounded list that only mounts the rows near the viewport.
 *
 * Rows are measured after they mount, so a row that wraps to two lines takes
 * the room it needs. The list has no anchor of its own: it reads from the
 * top, as a catalog does, where the thread's list reads from the end.
 */
export function VirtualRows({
  rows,
  estimateSize = 44,
  initialHeight = 320,
  overscan = 8,
  className,
  rowClassName,
  empty,
  "aria-label": ariaLabel,
}: VirtualRowsProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => estimateSize,
    getItemKey: (index) => rows[index].key,
    overscan,
    initialRect: { width: 0, height: initialHeight },
  });

  if (rows.length === 0) {
    return empty ? <>{empty}</> : null;
  }

  return (
    <div
      ref={scrollRef}
      aria-label={ariaLabel}
      className={cn("scrollbar-overlay overflow-y-auto overscroll-contain", className)}
    >
      <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
        {virtualizer.getVirtualItems().map((item) => (
          <div
            key={item.key}
            data-index={item.index}
            ref={virtualizer.measureElement}
            className={cn("absolute inset-x-0 top-0", rowClassName)}
            style={{ transform: `translateY(${item.start}px)` }}
          >
            {rows[item.index].render()}
          </div>
        ))}
      </div>
    </div>
  );
}
