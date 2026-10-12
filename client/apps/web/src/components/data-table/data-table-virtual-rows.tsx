"use no memo";
// TanStack Virtual hands back one virtualizer object that it mutates as the reader
// scrolls; the compiler would memoize what it returns and freeze the window.
import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";

/** What a drawn row needs so the window can measure it. */
export type VirtualRowSlot = {
  index: number;
  measureRef: (element: Element | null) => void;
};

type DataTableVirtualRowsProps<TItem> = {
  items: readonly TItem[];
  getKey: (item: TItem) => string;
  /** The table, so the window can find the element that scrolls it. */
  tableRef: React.RefObject<HTMLTableElement | null>;
  /** A row's height before it is measured. */
  estimateSize: number;
  colSpan: number;
  renderItem: (item: TItem, slot: VirtualRowSlot) => ReactNode;
};

const OVERSCAN = 12;

function scrollParentOf(table: HTMLTableElement | null): HTMLElement | null {
  return table?.closest<HTMLElement>('[data-slot="scroll-area-viewport"]') ?? null;
}

/**
 * Draws only the rows near the visible part of the table, with a spacer row above
 * and below standing in for the rest, so the table keeps its real layout: the header
 * stays stuck, columns keep their widths and the scrollbar reads true. Each row is
 * measured once drawn, so an open row's panel takes the room it needs.
 */
export function DataTableVirtualRows<TItem>({
  items,
  getKey,
  tableRef,
  estimateSize,
  colSpan,
  renderItem,
}: DataTableVirtualRowsProps<TItem>) {
  const [scrollElement, setScrollElement] = useState<HTMLElement | null>(null);
  const [margin, setMargin] = useState(0);
  const markerRef = useRef<HTMLTableRowElement>(null);
  useEffect(() => {
    setScrollElement(scrollParentOf(tableRef.current));
  }, [tableRef]);
  // Where the window starts: below the header and any pinned rows, which scroll
  // with the table but are not part of the window. Measured again whenever the
  // table changes size, which is what a pinned row coming or going does.
  useLayoutEffect(() => {
    const table = tableRef.current;
    const measure = () => setMargin(markerRef.current?.offsetTop ?? 0);
    measure();
    if (!table) return;
    const observer = new ResizeObserver(measure);
    observer.observe(table);
    return () => observer.disconnect();
  }, [tableRef]);

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollElement,
    estimateSize: () => estimateSize,
    getItemKey: (index) => getKey(items[index]),
    overscan: OVERSCAN,
    scrollMargin: margin,
  });

  const virtualItems = virtualizer.getVirtualItems();
  const top = virtualItems.length > 0 ? virtualItems[0].start - margin : 0;
  const bottom =
    virtualItems.length > 0
      ? virtualizer.getTotalSize() - (virtualItems[virtualItems.length - 1].end - margin)
      : 0;

  return (
    <>
      <tr ref={markerRef} aria-hidden className="h-0" />
      {top > 0 ? (
        <tr aria-hidden style={{ height: top }}>
          <td colSpan={colSpan} className="p-0" />
        </tr>
      ) : null}
      {virtualItems.map((virtualItem) =>
        renderItem(items[virtualItem.index], {
          index: virtualItem.index,
          measureRef: virtualizer.measureElement,
        }),
      )}
      {bottom > 0 ? (
        <tr aria-hidden style={{ height: bottom }}>
          <td colSpan={colSpan} className="p-0" />
        </tr>
      ) : null}
    </>
  );
}
