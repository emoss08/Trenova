/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import type { VirtualizedDiffViewerProps } from "@/types/json-viewer";
import { useVirtualizer } from "@tanstack/react-virtual";
import { forwardRef, useRef } from "react";
import { DiffLine } from "./json-diff-line";

export function VirtualizedDiffViewer({ lines }: VirtualizedDiffViewerProps) {
  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: lines.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 24, // * Estimated line height
    overscan: 10, // * Number of items to render before and after the visible items
  });

  return (
    <JsonDiffLineOuter ref={parentRef}>
      <JsonDiffLineInner getTotalSize={virtualizer.getTotalSize}>
        {virtualizer.getVirtualItems().map((virtualItem) => {
          const { line, lineNumber, type } = lines[virtualItem.index];
          return (
            <DiffLineInner
              key={virtualItem.key}
              size={virtualItem.size}
              start={virtualItem.start}
            >
              <DiffLine line={line} lineNumber={lineNumber} type={type} />
            </DiffLineInner>
          );
        })}
      </JsonDiffLineInner>
    </JsonDiffLineOuter>
  );
}

const JsonDiffLineOuter = forwardRef<
  HTMLDivElement,
  { children: React.ReactNode }
>(({ children }, ref) => {
  return (
    <div
      ref={ref}
      className="overflow-auto max-h-[calc(80vh-100px)]"
      style={{ position: "relative" }}
    >
      {children}
    </div>
  );
});
JsonDiffLineOuter.displayName = "JsonDiffLineOuter";

function JsonDiffLineInner({
  children,
  getTotalSize,
}: {
  children: React.ReactNode;
  getTotalSize: () => number;
}) {
  return (
    <div
      style={{
        height: `${getTotalSize()}px`,
        width: "100%",
        position: "relative",
      }}
    >
      {children}
    </div>
  );
}

function DiffLineInner({
  children,
  size,
  start,
}: {
  children: React.ReactNode;
  size: number;
  start: number;
}) {
  return (
    <div
      style={{
        position: "absolute",
        top: 0,
        left: 0,
        width: "100%",
        height: `${size}px`,
        transform: `translateY(${start}px)`,
      }}
    >
      {children}
    </div>
  );
}
