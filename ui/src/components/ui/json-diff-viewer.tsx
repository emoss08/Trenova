import { cn } from "@/lib/utils";
import type {
  DiffLineProps,
  JsonDiffViewerProps,
  VirtualizedDiffViewerProps,
} from "@/types/json-viewer";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useMemo, useRef } from "react";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./dialog";

function DiffLine({ line, lineNumber, type }: DiffLineProps) {
  // * Define background colors based on type and theme
  const bgColor = useMemo(() => {
    if (type === "added") {
      return "bg-green-50 dark:bg-green-950/40";
    } else if (type === "removed") {
      return "bg-red-50 dark:bg-red-950/40";
    }
    return "";
  }, [type]);

  // * Define text colors based on type
  const textColor = useMemo(() => {
    if (type === "added") {
      return "text-green-600 dark:text-green-400";
    } else if (type === "removed") {
      return "text-red-600 dark:text-red-400";
    }
    return "text-foreground";
  }, [type]);

  // * Add a symbol at the beginning based on type
  const linePrefix = useMemo(() => {
    if (type === "added") {
      return <span className="text-green-600 dark:text-green-400 mr-2">+</span>;
    } else if (type === "removed") {
      return <span className="text-red-600 dark:text-red-400 mr-2">-</span>;
    }
    return <span className="mr-2"> </span>;
  }, [type]);

  const syntaxHighlightedLine = useMemo(() => {
    if (!line) return null;

    const parts = [];
    let currentIndex = 0;

    // * Match property keys and their quotes
    const keyRegex = /"([^"]+)"(?=\s*:)/g;
    let match;

    while ((match = keyRegex.exec(line)) !== null) {
      // * Add any text before the match
      if (match.index > currentIndex) {
        parts.push(
          <span key={`pre-${match.index}`}>
            {line.substring(currentIndex, match.index)}
          </span>,
        );
      }

      // * Add the property key with highlighting
      parts.push(
        <span key={`key-${match.index}`} className="text-vitess-node">
          {match[0]}
        </span>,
      );

      currentIndex = match.index + match[0].length;
    }

    // * Add any remaining text
    if (currentIndex < line.length) {
      const remainingText = line.substring(currentIndex);

      // * Highlight string values
      const stringValueRegex = /: "([^"]*)"/g;
      const valueMatch = stringValueRegex.exec(remainingText);

      if (valueMatch) {
        const preValueText = remainingText.substring(0, valueMatch.index);
        const valueText = valueMatch[0];
        const postValueText = remainingText.substring(
          valueMatch.index + valueText.length,
        );

        parts.push(<span key="pre-value">{preValueText}</span>);
        parts.push(
          <span key="value" className="text-vitess-string">
            {valueText}
          </span>,
        );
        parts.push(<span key="post-value">{postValueText}</span>);
      } else {
        // * Highlight other values (numbers, booleans, null)
        const formattedText = remainingText
          .replace(
            /(: -?\d+(\.\d+)?)/g,
            '<span class="text-vitess-number">$1</span>',
          )
          .replace(
            /(: true|: false)/g,
            '<span class="text-vitess-boolean">$1</span>',
          )
          .replace(/(: null)/g, '<span class="text-gray-500">$1</span>');

        if (formattedText !== remainingText) {
          parts.push(
            <span
              key="values"
              dangerouslySetInnerHTML={{ __html: formattedText }}
            />,
          );
        } else {
          parts.push(<span key="remaining">{remainingText}</span>);
        }
      }
    }

    return parts.length > 0 ? parts : line;
  }, [line]);

  return (
    <div className={cn("flex py-1 px-2", bgColor)}>
      <span className="w-8 text-muted-foreground text-xs font-mono pr-2 text-right select-none">
        {lineNumber}
      </span>
      <div className={cn("font-mono text-sm flex-1", textColor)}>
        {linePrefix}
        {syntaxHighlightedLine}
      </div>
    </div>
  );
}

function VirtualizedDiffViewer({ lines }: VirtualizedDiffViewerProps) {
  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: lines.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 24, // * Estimated line height
    overscan: 10, // * Number of items to render before and after the visible items
  });

  return (
    <div
      ref={parentRef}
      className="overflow-auto max-h-[calc(80vh-100px)]"
      style={{ position: "relative" }}
    >
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: "100%",
          position: "relative",
        }}
      >
        {virtualizer.getVirtualItems().map((virtualItem) => {
          const { line, lineNumber, type } = lines[virtualItem.index];
          return (
            <div
              key={virtualItem.key}
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                height: `${virtualItem.size}px`,
                transform: `translateY(${virtualItem.start}px)`,
              }}
            >
              <DiffLine line={line} lineNumber={lineNumber} type={type} />
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function JsonCodeDiffViewer({
  oldData,
  newData,
  title = { old: "Previous Version", new: "Current Version" },
  className,
}: JsonDiffViewerProps) {
  // * Format the JSON data for display
  const oldJson = useMemo(() => {
    if (!oldData) return [];
    try {
      return JSON.stringify(oldData, null, 2).split("\n");
    } catch (error) {
      console.error("Error formatting old data:", error);
      return [];
    }
  }, [oldData]);

  const newJson = useMemo(() => {
    if (!newData) return [];
    try {
      return JSON.stringify(newData, null, 2).split("\n");
    } catch (error) {
      console.error("Error formatting new data:", error);
      return [];
    }
  }, [newData]);

  // * Prepare data for virtualized lists
  const oldLines = useMemo(
    () =>
      oldJson.map((line, index) => ({
        line,
        lineNumber: index + 1,
        type: isLineRemoved(line, newJson) ? "removed" : ("unchanged" as const),
      })),
    [oldJson, newJson],
  );

  const newLines = useMemo(
    () =>
      newJson.map((line, index) => ({
        line,
        lineNumber: index + 1,
        type: isLineAdded(line, oldJson) ? "added" : ("unchanged" as const),
      })),
    [newJson, oldJson],
  );

  // * Check if diffs are large (to decide whether to use virtualization)
  const isLargeDiff = oldLines.length > 500 || newLines.length > 500;

  return (
    <div className={cn("grid grid-cols-1 md:grid-cols-2 gap-4", className)}>
      {/* Old Data Panel */}
      <div className="overflow-hidden rounded-md border border-border bg-card">
        <div className="p-2 border-b border-border bg-muted/30">
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium text-foreground">
              {title.old}
            </span>
            <span className="text-xs text-muted-foreground">
              {oldLines.length} lines
            </span>
          </div>
        </div>
        {isLargeDiff ? (
          <VirtualizedDiffViewer
            lines={
              oldLines as {
                line: string;
                lineNumber: number;
                type: "removed" | "unchanged" | "added";
              }[]
            }
          />
        ) : (
          <div className="overflow-auto max-h-[calc(80vh-100px)]">
            {oldLines.map(({ line, lineNumber, type }, index) => (
              <DiffLine
                key={`old-${index}`}
                line={line}
                lineNumber={lineNumber}
                type={type as "removed" | "unchanged" | "added"}
              />
            ))}
          </div>
        )}
      </div>

      {/* New Data Panel */}
      <div className="overflow-hidden rounded-md border border-border bg-card">
        <div className="p-2 border-b border-border bg-muted/30">
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium text-foreground">
              {title.new}
            </span>
            <span className="text-xs text-muted-foreground">
              {newLines.length} lines
            </span>
          </div>
        </div>

        {isLargeDiff ? (
          <VirtualizedDiffViewer
            lines={
              newLines as {
                line: string;
                lineNumber: number;
                type: "removed" | "unchanged" | "added";
              }[]
            }
          />
        ) : (
          <div className="overflow-auto max-h-[calc(80vh-100px)]">
            {newLines.map(({ line, lineNumber, type }, index) => (
              <DiffLine
                key={`new-${index}`}
                line={line}
                lineNumber={lineNumber}
                type={type as "removed" | "unchanged" | "added"}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function isLineRemoved(line: string, newLines: string[]): boolean {
  // * Quick sanity check - if the exact line exists in newLines, it wasn't removed
  if (newLines.includes(line)) {
    return false;
  }

  const trimmedLine = line.trim();
  // * Skip lines that are just structural (brackets, commas, etc)
  if (trimmedLine === "" || /^[{}[\],]$/.test(trimmedLine)) {
    return false;
  }

  // * For property values, we need to identify if the property exists with a different value
  if (trimmedLine.includes(":")) {
    // * Extract the property key (removing quotes and whitespace)
    const keyMatch = trimmedLine.match(/"([^"]+)"/);
    if (keyMatch) {
      const key = keyMatch[1];

      // * Look for the same key in new lines
      const keyPattern = new RegExp(`"${key}"\\s*:`);
      const newLineWithSameKey = newLines.find((nl) => keyPattern.test(nl));

      // * If the key exists in newLines but with a different value, it was modified (consider as removed)
      if (newLineWithSameKey && newLineWithSameKey !== line) {
        return true;
      }

      // * If the key doesn't exist at all in newLines, it was removed completely
      if (!newLineWithSameKey) {
        return true;
      }
    }
  }

  // *For other significant content (not just structural elements),
  // *if it doesn't exist in newLines, consider it removed
  const isSignificantContent =
    trimmedLine.includes('"') ||
    trimmedLine.includes("true") ||
    trimmedLine.includes("false") ||
    trimmedLine.includes("null") ||
    /\d+/.test(trimmedLine);

  if (
    isSignificantContent &&
    !newLines.some((nl) => nl.includes(trimmedLine))
  ) {
    return true;
  }

  return false;
}

function isLineAdded(line: string, oldLines: string[]): boolean {
  // * Quick sanity check - if the exact line exists in oldLines, it wasn't added
  if (oldLines.includes(line)) {
    return false;
  }

  const trimmedLine = line.trim();
  // * Skip lines that are just structural (brackets, commas, etc)
  if (trimmedLine === "" || /^[{}[\],]$/.test(trimmedLine)) {
    return false;
  }

  // * For property values, we need to identify if the property exists with a different value
  if (trimmedLine.includes(":")) {
    // * Extract the property key (removing quotes and whitespace)
    const keyMatch = trimmedLine.match(/"([^"]+)"/);
    if (keyMatch) {
      const key = keyMatch[1];

      // * Look for the same key in old lines
      const keyPattern = new RegExp(`"${key}"\\s*:`);
      const oldLineWithSameKey = oldLines.find((ol) => keyPattern.test(ol));

      // * If the key exists in oldLines but with a different value, it was modified (consider as added)
      if (oldLineWithSameKey && oldLineWithSameKey !== line) {
        return true;
      }

      // * If the key doesn't exist at all in oldLines, it was added completely
      if (!oldLineWithSameKey) {
        return true;
      }
    }
  }

  // * For other significant content (not just structural elements),
  // * if it doesn't exist in oldLines, consider it added
  const isSignificantContent =
    trimmedLine.includes('"') ||
    trimmedLine.includes("true") ||
    trimmedLine.includes("false") ||
    trimmedLine.includes("null") ||
    /\d+/.test(trimmedLine);

  if (
    isSignificantContent &&
    !oldLines.some((ol) => ol.includes(trimmedLine))
  ) {
    return true;
  }

  return false;
}

export function ChangeDiffDialog({
  changes,
  open,
  onOpenChange,
}: {
  changes: Record<string, { from: any; to: any }>;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  // * Transform the changes object into consolidated before/after objects for comparison
  const { fromData, toData } = useMemo(() => {
    const fromData: Record<string, any> = {};
    const toData: Record<string, any> = {};

    Object.entries(changes).forEach(([key, change]) => {
      fromData[key] = change.from;
      toData[key] = change.to;
    });

    return { fromData, toData };
  }, [changes]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-5xl 4xl:max-w-8xl">
        <DialogHeader>
          <DialogTitle>Detailed Change Comparison</DialogTitle>
          <DialogDescription>
            Side-by-side view of all modified values in this record
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="p-4">
          <JsonCodeDiffViewer
            oldData={fromData}
            newData={toData}
            title={{ old: "Previous Version", new: "Current Version" }}
          />
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
