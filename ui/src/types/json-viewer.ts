/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

export type JsonViewerType = "added" | "removed" | "unchanged";

export type ActionType = "added" | "removed" | "unchanged";

export type JsonDiffViewerProps = {
  oldData: any;
  newData: any;
  title?: { old: string; new: string };
  className?: string;
};

export type DiffLineProps = {
  line: string;
  lineNumber: number;
  type: ActionType;
};

export type VirtualizedDiffViewerProps = {
  lines: Array<{
    line: string;
    lineNumber: number;
    type: ActionType;
  }>;
};

export type JsonViewerProps = {
  data: any;
  className?: string;
  initiallyExpanded?: boolean;
};

export type CollapsibleNodeProps = {
  name: string | number | null;
  value: any;
  isRoot?: boolean;
  initiallyExpanded?: boolean;
  withComma?: boolean;
};

export type NewLine = {
  line: string;
  lineNumber: number;
  type: JsonViewerType;
};
