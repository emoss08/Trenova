import { isLineAdded, isLineRemoved } from "@/lib/json-viewer-utils";
import { cn } from "@/lib/utils";
import type { JsonDiffViewerProps, JsonViewerType } from "@/types/json-viewer";
import { useMemo } from "react";
import { JsonDiffPanel } from "../json-viewer/json-diff-panelt";
import { BetaTag } from "./beta-tag";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./dialog";
import { formatJsonWithSpaces } from "@/lib/json-sensitive-utils";
import { JsonSmartDiff } from "./json-smart-diff";

export function JsonCodeDiffViewer({
  oldData,
  newData,
  title = { old: "Previous Version", new: "Current Version" },
  className,
}: JsonDiffViewerProps) {
  // * Format the JSON data for display with spaces after colons
  const oldJson = useMemo(() => {
    if (!oldData) return [];
    const formatted = formatJsonWithSpaces(oldData);
    return formatted ? formatted.split("\n") : [];
  }, [oldData]);

  const newJson = useMemo(() => {
    if (!newData) return [];
    const formatted = formatJsonWithSpaces(newData);
    return formatted ? formatted.split("\n") : [];
  }, [newData]);

  // * Prepare data for virtualized lists
  const oldLines = useMemo(
    () =>
      oldJson.map((line, index) => ({
        line,
        lineNumber: index + 1,
        type: isLineRemoved(line, newJson)
          ? ("removed" as JsonViewerType)
          : ("unchanged" as JsonViewerType),
      })),
    [oldJson, newJson],
  );

  const newLines = useMemo(
    () =>
      newJson.map((line, index) => ({
        line,
        lineNumber: index + 1,
        type: isLineAdded(line, oldJson)
          ? "added"
          : ("unchanged" as JsonViewerType),
      })),
    [newJson, oldJson],
  );

  // * Check if diffs are large (to decide whether to use virtualization)
  const isLargeDiff = oldLines.length > 500 || newLines.length > 500;

  return (
    <JsonCodeDiffViewerInner className={className}>
      {/* Old Data Panel */}
      <JsonDiffPanel
        title={title.old}
        lines={oldLines}
        isLargeDiff={isLargeDiff}
      />

      {/* New Data Panel */}
      <JsonDiffPanel
        title={title.new}
        lines={newLines}
        isLargeDiff={isLargeDiff}
      />
    </JsonCodeDiffViewerInner>
  );
}

function JsonCodeDiffViewerInner({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("grid grid-cols-1 md:grid-cols-2 gap-4", className)}>
      {children}
    </div>
  );
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
      <DialogContent className="max-w-7xl">
        <DialogHeader>
          <DialogTitle>
            Detailed Change Comparison <BetaTag />
          </DialogTitle>
          <DialogDescription>
            Side-by-side view of all modified values in this record
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="p-4 overflow-hidden">
          <JsonSmartDiff
            oldData={fromData}
            newData={toData}
            title={{ old: "Previous Version", new: "Current Version" }}
          />
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
