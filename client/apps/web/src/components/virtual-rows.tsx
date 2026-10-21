"use no memo";
import { useDeferredMeasure } from "@/hooks/use-deferred-measure";
import { cn } from "@trenova/shared/lib/utils";
import { defaultRangeExtractor, useVirtualizer, type Range } from "@tanstack/react-virtual";
import {
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  type ReactNode,
  type Ref,
} from "react";

export type VirtualRow = {
  key: string;
  render: () => ReactNode;
  /**
   * A heading that stays pinned to the top while the rows under it scroll,
   * until the next sticky row takes its place.
   */
  sticky?: boolean;
};

/** Moves a list from outside, as keyboard navigation does. */
export type VirtualRowsHandle = {
  scrollToIndex: (index: number, options?: { align?: "start" | "center" | "end" | "auto" }) => void;
  /** Scrolls the row with this key into view, if the list has it. */
  scrollToKey: (key: string, options?: { align?: "start" | "center" | "end" | "auto" }) => void;
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
  /** The scroll container's role, such as "listbox" for a list of choosable rows. */
  role?: string;
  "aria-multiselectable"?: boolean;
  /** Imperative handle for scrolling a row into view. */
  handleRef?: Ref<VirtualRowsHandle>;
  /** The scroll container itself, for callers that observe it. */
  scrollRef?: Ref<HTMLDivElement>;
  /** Called when the window reaches within `endReachedThreshold` rows of the end: load more here. */
  onEndReached?: () => void;
  endReachedThreshold?: number;
  /** Drawn after the last row, inside the scroll container: a "loading more" line. */
  footer?: ReactNode;
};

function assignRef<T>(ref: Ref<T> | undefined, value: T | null) {
  if (typeof ref === "function") {
    ref(value);
  } else if (ref !== null && ref !== undefined) {
    (ref as { current: T | null }).current = value;
  }
}

/**
 * A bounded list that only mounts the rows near the viewport.
 *
 * Rows are measured after they mount, so a row that wraps to two lines takes
 * the room it needs. The list has no anchor of its own: it reads from the
 * top, as a catalog does, where the thread's list reads from the end. A row
 * marked sticky is a group heading: it stays pinned while its rows scroll
 * under it.
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
  role,
  "aria-multiselectable": ariaMultiselectable,
  handleRef,
  scrollRef: outerScrollRef,
  onEndReached,
  endReachedThreshold = 5,
  footer,
}: VirtualRowsProps) {
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const setScrollElement = useCallback(
    (element: HTMLDivElement | null) => {
      scrollRef.current = element;
      assignRef(outerScrollRef, element);
    },
    [outerScrollRef],
  );

  const stickyIndexes = useMemo(
    () => rows.flatMap((row, index) => (row.sticky === true ? [index] : [])),
    [rows],
  );
  const activeSticky = useRef(-1);

  const rangeExtractor = useCallback(
    (range: Range) => {
      let active = -1;
      for (const index of stickyIndexes) {
        if (index > range.startIndex) {
          break;
        }
        active = index;
      }
      activeSticky.current = active;
      if (active === -1) {
        return defaultRangeExtractor(range);
      }
      const indexes = new Set([active, ...defaultRangeExtractor(range)]);
      return [...indexes].sort((a, b) => a - b);
    },
    [stickyIndexes],
  );

  // oxlint-disable-next-line react/incompatible-library -- TanStack Virtual is incompatible with the React Compiler; file is already "use no memo", which oxlint 1.89 ignores
  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => estimateSize,
    getItemKey: (index) => rows[index].key,
    overscan,
    initialRect: { width: 0, height: initialHeight },
    rangeExtractor: stickyIndexes.length > 0 ? rangeExtractor : undefined,
  });

  const measureRow = useDeferredMeasure(virtualizer);

  useImperativeHandle(
    handleRef,
    () => ({
      scrollToIndex: (index, options) => {
        if (index >= 0 && index < rows.length) {
          virtualizer.scrollToIndex(index, { align: options?.align ?? "auto" });
        }
      },
      scrollToKey: (key, options) => {
        const index = rows.findIndex((row) => row.key === key);
        if (index !== -1) {
          virtualizer.scrollToIndex(index, { align: options?.align ?? "auto" });
        }
      },
    }),
    [rows, virtualizer],
  );

  const items = virtualizer.getVirtualItems();
  const lastIndex = items.at(-1)?.index ?? -1;

  useEffect(() => {
    if (onEndReached === undefined || rows.length === 0) {
      return;
    }
    if (lastIndex >= rows.length - 1 - endReachedThreshold) {
      onEndReached();
    }
  }, [lastIndex, rows.length, endReachedThreshold, onEndReached]);

  if (rows.length === 0) {
    return empty ? <>{empty}</> : null;
  }

  return (
    <div
      ref={setScrollElement}
      role={role}
      aria-label={ariaLabel}
      aria-multiselectable={ariaMultiselectable}
      className={cn("scrollbar-overlay overflow-y-auto overscroll-contain", className)}
    >
      <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
        {items.map((item) => {
          const pinned = rows[item.index].sticky === true && item.index === activeSticky.current;
          return (
            <div
              key={item.key}
              data-index={item.index}
              ref={measureRow}
              className={cn(
                pinned ? "sticky top-0 z-10" : "absolute inset-x-0 top-0",
                rowClassName,
              )}
              style={pinned ? undefined : { transform: `translateY(${item.start}px)` }}
            >
              {rows[item.index].render()}
            </div>
          );
        })}
      </div>
      {footer}
    </div>
  );
}
