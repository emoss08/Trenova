/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { detectSensitiveDataType } from "@/lib/json-sensitive-utils";
import { cn } from "@/lib/utils";
import { MinusIcon, PlusIcon } from "@radix-ui/react-icons";
import React, { useMemo } from "react";
import { ScrollArea } from "./scroll-area";
import { SensitiveBadge } from "./sensitive-badge";

interface JsonLine {
  lineNumber: number;
  content: string;
  path: string;
  key?: string;
  value?: any;
  type: "structural" | "value";
  indent: number;
}

interface DiffResult {
  left: (JsonLine & { changeType?: "removed" | "modified" })[];
  right: (JsonLine & { changeType?: "added" | "modified" })[];
}

interface JsonSmartDiffProps {
  oldData: any;
  newData: any;
  title?: {
    old: string;
    new: string;
  };
  className?: string;
}

function flattenJson(
  obj: any,
  parentPath: string = "",
  lines: JsonLine[] = [],
  lineNum = { current: 1 },
  indent: number = 0,
): JsonLine[] {
  if (obj === null || obj === undefined) {
    lines.push({
      lineNumber: lineNum.current++,
      content: `${"  ".repeat(indent)}null`,
      path: parentPath,
      value: null,
      type: "value",
      indent,
    });
    return lines;
  }

  if (typeof obj !== "object") {
    const content = typeof obj === "string" ? `"${obj}"` : String(obj);
    lines.push({
      lineNumber: lineNum.current++,
      content: `${"  ".repeat(indent)}${content}`,
      path: parentPath,
      value: obj,
      type: "value",
      indent,
    });
    return lines;
  }

  const isArray = Array.isArray(obj);

  // Opening bracket
  lines.push({
    lineNumber: lineNum.current++,
    content: `${"  ".repeat(indent)}${isArray ? "[" : "{"}`,
    path: parentPath,
    type: "structural",
    indent,
  });

  const entries = isArray ? obj.map((v, i) => [i, v]) : Object.entries(obj);

  entries.forEach(([key, value], index) => {
    const isLast = index === entries.length - 1;
    const currentPath = isArray
      ? `${parentPath}[${key}]`
      : `${parentPath}.${key}`;
    const cleanPath = currentPath.startsWith(".")
      ? currentPath.slice(1)
      : currentPath;

    if (value !== null && typeof value === "object") {
      // Object or array - just add the key line for objects
      if (!isArray) {
        lines.push({
          lineNumber: lineNum.current++,
          content: `${"  ".repeat(indent + 1)}"${key}": ${Array.isArray(value) ? "[" : "{"}`,
          path: cleanPath,
          key: String(key),
          type: "structural",
          indent: indent + 1,
        });
      }

      // Recursively process the nested structure
      flattenJson(
        value,
        cleanPath,
        lines,
        lineNum,
        isArray ? indent + 1 : indent + 1,
      );

      // Closing bracket for objects (arrays handle their own)
      if (!isArray) {
        const lastLine = lines[lines.length - 1];
        lastLine.content += isLast ? "" : ",";
      }
    } else {
      // Primitive value
      const valueStr =
        value === null
          ? "null"
          : typeof value === "string"
            ? `"${value}"`
            : String(value);

      lines.push({
        lineNumber: lineNum.current++,
        content: `${"  ".repeat(indent + 1)}${isArray ? "" : `"${key}": `}${valueStr}${isLast ? "" : ","}`,
        path: cleanPath,
        key: isArray ? undefined : String(key),
        value: value,
        type: "value",
        indent: indent + 1,
      });
    }
  });

  // Closing bracket
  lines.push({
    lineNumber: lineNum.current++,
    content: `${"  ".repeat(indent)}}`,
    path: parentPath,
    type: "structural",
    indent,
  });

  return lines;
}

function createSmartDiff(oldData: any, newData: any): DiffResult {
  const oldLines = flattenJson(oldData);
  const newLines = flattenJson(newData);

  // Create maps for quick lookup by path
  const oldByPath = new Map<string, JsonLine>();
  const newByPath = new Map<string, JsonLine>();

  oldLines.forEach((line) => {
    if (line.type === "value" && line.path) {
      oldByPath.set(line.path, line);
    }
  });

  newLines.forEach((line) => {
    if (line.type === "value" && line.path) {
      newByPath.set(line.path, line);
    }
  });

  // Mark changes
  const leftLines = oldLines.map((line) => {
    const lineWithChange = { ...line } as JsonLine & {
      changeType?: "removed" | "modified";
    };

    if (line.type === "value" && line.path) {
      const newLine = newByPath.get(line.path);
      if (!newLine) {
        // This path doesn't exist in new data
        lineWithChange.changeType = "removed";
      } else if (JSON.stringify(line.value) !== JSON.stringify(newLine.value)) {
        // Value changed
        lineWithChange.changeType = "modified";
      }
    }

    return lineWithChange;
  });

  const rightLines = newLines.map((line) => {
    const lineWithChange = { ...line } as JsonLine & {
      changeType?: "added" | "modified";
    };

    if (line.type === "value" && line.path) {
      const oldLine = oldByPath.get(line.path);
      if (!oldLine) {
        // This path doesn't exist in old data
        lineWithChange.changeType = "added";
      } else if (JSON.stringify(line.value) !== JSON.stringify(oldLine.value)) {
        // Value changed
        lineWithChange.changeType = "modified";
      }
    }

    return lineWithChange;
  });

  return { left: leftLines, right: rightLines };
}

interface DiffLineComponentProps {
  line: JsonLine & { changeType?: "removed" | "modified" | "added" };
  side: "left" | "right";
}

function DiffLineComponent({ line, side }: DiffLineComponentProps) {
  // Determine if this line should have a subtle background
  const hasLineBackground = line.changeType && line.type === "value";
  const lineBgColor = hasLineBackground
    ? line.changeType === "removed"
      ? "bg-red-50 dark:bg-red-950"
      : line.changeType === "added"
        ? "bg-green-50 dark:bg-green-950"
        : line.changeType === "modified"
          ? side === "left"
            ? "bg-red-50 dark:bg-red-950"
            : "bg-green-50 dark:bg-green-950"
          : ""
    : "";

  const renderContent = () => {
    const parts: React.ReactNode[] = [];
    const content = line.content;

    // Handle indentation
    const indentMatch = content.match(/^(\s*)/);
    if (indentMatch) {
      parts.push(
        <span key="indent" className="whitespace-pre">
          {indentMatch[1]}
        </span>,
      );
    }

    // Remove indentation for parsing
    const trimmedContent = content.trimStart();

    // Parse different types of content
    if (line.type === "structural") {
      // Structural elements (brackets, object keys)
      const keyMatch = trimmedContent.match(/^"([^"]+)":\s*(.*)$/);
      if (keyMatch) {
        parts.push(
          <span key="key" className="text-blue-600 dark:text-blue-400">
            &quot;{keyMatch[1]}&quot;
          </span>,
          <span key="colon" className="text-muted-foreground">
            :{" "}
          </span>,
          <span key="bracket" className="text-muted-foreground">
            {keyMatch[2]}
          </span>,
        );
      } else {
        parts.push(
          <span key="structural" className="text-muted-foreground">
            {trimmedContent}
          </span>,
        );
      }
    } else if (line.type === "value") {
      // Handle key-value pairs
      const keyValueMatch = trimmedContent.match(/^"([^"]+)":\s*(.+?)(,?)$/);

      if (keyValueMatch) {
        // It's a key-value pair
        parts.push(
          <span key="key" className="text-blue-600 dark:text-blue-400">
            &quot;{keyValueMatch[1]}&quot;
          </span>,
          <span key="colon" className="text-muted-foreground">
            :{" "}
          </span>,
        );

        const valueContent = keyValueMatch[2];
        const hasComma = keyValueMatch[3];

        // Check if this line has changes
        const hasChange =
          line.changeType === "modified" ||
          (side === "left" && line.changeType === "removed") ||
          (side === "right" && line.changeType === "added");

        // Render the value with potential highlighting
        const valueElement = renderValue(
          line.value,
          valueContent,
          hasChange,
          line.changeType,
          side,
        );
        parts.push(valueElement);

        if (hasComma) {
          parts.push(
            <span key="comma" className="text-muted-foreground">
              ,
            </span>,
          );
        }
      } else {
        // It's just a value (in an array)
        const valueMatch = trimmedContent.match(/^(.+?)(,?)$/);
        if (valueMatch) {
          const valueContent = valueMatch[1];
          const hasComma = valueMatch[2];

          const hasChange =
            line.changeType === "modified" ||
            (side === "left" && line.changeType === "removed") ||
            (side === "right" && line.changeType === "added");

          const valueElement = renderValue(
            line.value,
            valueContent,
            hasChange,
            line.changeType,
            side,
          );
          parts.push(valueElement);

          if (hasComma) {
            parts.push(
              <span key="comma" className="text-muted-foreground">
                ,
              </span>,
            );
          }
        }
      }
    }

    return parts;
  };

  return (
    <div
      className={cn(
        "flex items-center px-2 py-0.5 font-mono text-sm",
        lineBgColor,
      )}
    >
      <span className="w-12 text-right text-xs text-muted-foreground select-none pr-4">
        {line.lineNumber}
      </span>
      <span className="flex-1">{renderContent()}</span>
    </div>
  );
}

function renderValue(
  value: any,
  displayContent: string,
  hasChange: boolean,
  changeType?: string,
  side?: "left" | "right",
): React.ReactNode {
  const sensitiveInfo = detectSensitiveDataType(value);

  // Determine the background color for changes
  const bgColor = hasChange
    ? changeType === "removed"
      ? "bg-red-300 dark:bg-red-700"
      : changeType === "added"
        ? "bg-green-300 dark:bg-green-700"
        : side === "left"
          ? "bg-red-300 dark:bg-red-700"
          : "bg-green-300 dark:bg-green-700"
    : "";

  if (sensitiveInfo.isSensitive) {
    return (
      <span
        key="value"
        className={cn(
          "inline-flex items-center gap-1",
          bgColor && "px-1 rounded",
        )}
      >
        <span
          className={cn(
            sensitiveInfo.type === "redacted" &&
              "text-red-600 dark:text-red-400",
          )}
        >
          {displayContent}
        </span>
        <SensitiveBadge
          size="xs"
          variant={
            sensitiveInfo.type === "redacted" ? "destructive" : "warning"
          }
        />
      </span>
    );
  }

  return (
    <span key="value" className={cn(bgColor && "px-1 rounded", bgColor)}>
      {displayContent}
    </span>
  );
}

export function JsonSmartDiff({
  oldData,
  newData,
  title = { old: "Previous Version", new: "Current Version" },
  className,
}: JsonSmartDiffProps) {
  const { left, right } = useMemo(
    () => createSmartDiff(oldData, newData),
    [oldData, newData],
  );

  // Count all changes on each side
  const leftChanges = left.filter(
    (l) => l.changeType === "removed" || l.changeType === "modified",
  ).length;
  const rightChanges = right.filter(
    (l) => l.changeType === "added" || l.changeType === "modified",
  ).length;

  return (
    <div className={cn("grid grid-cols-2 gap-4 h-full", className)}>
      {/* Old Version */}
      <div className="flex flex-col border border-border rounded-lg overflow-hidden">
        <div className="px-4 py-2 bg-muted border-b border-border">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-sm">{title.old}</h3>
            <span className="flex items-center gap-1 text-xs text-red-800 font-bold">
              <MinusIcon className="size-4 bg-red-100 dark:bg-red-400 rounded-full" />
              {leftChanges} {leftChanges === 1 ? "removal" : "removals"}
            </span>
          </div>
        </div>
        <ScrollArea className="flex-1">
          <div className="min-w-0">
            {left.map((line, idx) => (
              <DiffLineComponent key={idx} line={line} side="left" />
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* New Version */}
      <div className="flex flex-col border border-border rounded-lg overflow-hidden">
        <div className="px-4 py-2 bg-muted border-b border-border">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-sm">{title.new}</h3>
            <span className="flex items-center gap-1 text-xs text-green-800 font-bold">
              <PlusIcon className="size-4 bg-green-100 dark:bg-green-400 rounded-full" />
              {rightChanges} {rightChanges === 1 ? "addition" : "additions"}
            </span>
          </div>
        </div>
        <ScrollArea className="flex-1">
          <div className="min-w-0">
            {right.map((line, idx) => (
              <DiffLineComponent key={idx} line={line} side="right" />
            ))}
          </div>
        </ScrollArea>
      </div>
    </div>
  );
}
