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
