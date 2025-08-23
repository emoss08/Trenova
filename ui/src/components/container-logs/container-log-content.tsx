/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */
import {
  ScrollAreaShadow,
  VirtualCompatibleScrollArea,
} from "@/components/ui/scroll-area";
import { classifyLevel, escapeRegex } from "@/lib/docker-utils";

import { cn } from "@/lib/utils";
import { useContainerLogStore } from "@/stores/docker-store";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Activity, AlertCircle } from "lucide-react";
import { useCallback } from "react";

export default function ContainerLogContent({
  filteredCount,
  needle,
  filteredLogs,
  scrollAreaRef,
  isLoading,
  error,
}: {
  filteredCount: number;
  needle: string;
  filteredLogs: string[];
  scrollAreaRef: React.RefObject<HTMLDivElement | null>;
  isLoading: boolean;
  error: Error | null;
}) {
  const wrap = useContainerLogStore.get("wrap");
  const showLineNumbers = useContainerLogStore.get("showLineNumbers");
  const rowVirtualizer = useVirtualizer({
    count: filteredCount,
    getScrollElement: () => scrollAreaRef.current,
    estimateSize: useCallback(
      (index: number) => {
        const line = filteredLogs[index];
        return line.length + 18;
      },
      [filteredLogs],
    ),
    overscan: 5,
    measureElement: (element) => {
      return element?.getBoundingClientRect().height ?? 70;
    },
  });

  return (
    <VirtualCompatibleScrollArea
      viewPortRef={scrollAreaRef}
      className="h-[520px]"
      viewPortClassName="h-[520px]"
    >
      <div
        className={cn(
          "min-h-full font-mono text-xs pb-4 px-2",
          wrap ? "whitespace-pre-wrap break-words" : "whitespace-pre",
          "bg-background text-muted-foreground",
        )}
        aria-live="polite"
        role="log"
      >
        {isLoading ? (
          <div className="space-y-2">
            <div className="h-3 w-3/4 animate-pulse rounded bg-primary/30" />
            <div className="h-3 w-2/3 animate-pulse rounded bg-primary/30" />
            <div className="h-3 w-1/2 animate-pulse rounded bg-primary/30" />
          </div>
        ) : error ? (
          <div className="flex items-center gap-2 text-red-500">
            <AlertCircle className="h-4 w-4" />
            <span>
              Failed to load logs.{" "}
              <span className="opacity-70">Try refresh.</span>
            </span>
          </div>
        ) : filteredCount > 0 ? (
          <ul
            style={{
              height: `${rowVirtualizer.getTotalSize()}px`,
              width: "100%",
              position: "relative",
            }}
          >
            {rowVirtualizer.getVirtualItems().map((vi) => {
              const idx = vi.index;
              const line = filteredLogs[idx];
              const ln = idx + 1;
              const level = classifyLevel(line);
              return (
                <li
                  key={vi.key}
                  ref={rowVirtualizer.measureElement}
                  data-index={idx}
                  style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: "100%",
                    transform: `translateY(${vi.start}px)`,
                  }}
                >
                  {showLineNumbers && (
                    <span className="select-none max-w-12 mr-1 shrink-0 text-right tabular-nums text-foreground font-medium">
                      {ln}
                    </span>
                  )}
                  <span
                    className={cn(
                      "mt-1 inline-block size-2 rounded-full mr-1",
                      level === "error" && "bg-rose-500",
                      level === "warn" && "bg-amber-500",
                      level === "info" && "bg-blue-500",
                      level === "debug" && "bg-muted-foreground",
                      level === "other" && "bg-muted-foreground/50",
                    )}
                  />
                  <span className="flex-1">
                    <HighlightCompiled text={line} needle={needle} />
                  </span>
                </li>
              );
            })}
          </ul>
        ) : (
          <div className="flex items-center gap-2 text-muted-foreground">
            <Activity className="size-4" />
            <span>No logs found.</span>
          </div>
        )}
      </div>
      <ScrollAreaShadow />
    </VirtualCompatibleScrollArea>
  );
}

function HighlightCompiled({
  text,
  needle,
  className,
}: {
  text: string;
  needle: string;
  className?: string;
}) {
  if (!needle.trim() || !text) {
    return <span className={className}>{text}</span>;
  }
  const rx = new RegExp(`(${escapeRegex(needle)})`, "gi");
  const out: Array<{ str: string; match: boolean }> = [];
  let lastIndex = 0;
  for (const m of text.matchAll(rx)) {
    const idx = m.index ?? 0;
    if (idx > lastIndex)
      out.push({ str: text.slice(lastIndex, idx), match: false });
    out.push({ str: m[0], match: true });
    lastIndex = idx + m[0].length;
  }
  if (lastIndex < text.length)
    out.push({ str: text.slice(lastIndex), match: false });

  return (
    <span className={className}>
      {out.map((seg, i) =>
        seg.match ? (
          <span
            key={i}
            className="bg-yellow-400/80 shrink-0 font-medium dark:bg-yellow-400/40"
          >
            {seg.str}
          </span>
        ) : (
          <span key={i}>{seg.str}</span>
        ),
      )}
    </span>
  );
}
