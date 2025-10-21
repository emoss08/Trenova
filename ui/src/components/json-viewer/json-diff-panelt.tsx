/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import type { NewLine } from "@/types/json-viewer";
import { DiffLine } from "./json-diff-line";
import { VirtualizedDiffViewer } from "./json-viewer-diff-viewer";
import { JsonCodeDiffHeader } from "./json-viewer-header";

export function JsonDiffPanel({
  title,
  lines,
  isLargeDiff,
}: {
  title: string;
  lines: NewLine[];
  isLargeDiff: boolean;
}) {
  return (
    <JsonDiffPanelInner>
      <JsonCodeDiffHeader title={title} lines={lines.length} />
      {isLargeDiff ? (
        <VirtualizedDiffViewer lines={lines} />
      ) : (
        <JsonDiffLineInner>
          {lines.map(({ line, lineNumber, type }, index) => (
            <DiffLine
              key={`old-${index}`}
              line={line}
              lineNumber={lineNumber}
              type={type}
            />
          ))}
        </JsonDiffLineInner>
      )}
    </JsonDiffPanelInner>
  );
}

function JsonDiffPanelInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card">
      {children}
    </div>
  );
}

function JsonDiffLineInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="overflow-auto max-h-[calc(80vh-100px)]">{children}</div>
  );
}
