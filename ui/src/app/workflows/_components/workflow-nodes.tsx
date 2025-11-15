/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Handle, Position, type NodeProps } from "@xyflow/react";
import {
  Zap,
  Play,
  GitBranch,
  Repeat,
  Clock,
  Square,
} from "lucide-react";
import { memo } from "react";

const nodeIcons = {
  trigger: Zap,
  action: Play,
  condition: GitBranch,
  loop: Repeat,
  delay: Clock,
  end: Square,
};

const nodeColors = {
  trigger: {
    bg: "bg-green-100 dark:bg-green-950",
    border: "border-green-500",
    text: "text-green-700 dark:text-green-300",
  },
  action: {
    bg: "bg-blue-100 dark:bg-blue-950",
    border: "border-blue-500",
    text: "text-blue-700 dark:text-blue-300",
  },
  condition: {
    bg: "bg-yellow-100 dark:bg-yellow-950",
    border: "border-yellow-500",
    text: "text-yellow-700 dark:text-yellow-300",
  },
  loop: {
    bg: "bg-purple-100 dark:bg-purple-950",
    border: "border-purple-500",
    text: "text-purple-700 dark:text-purple-300",
  },
  delay: {
    bg: "bg-orange-100 dark:bg-orange-950",
    border: "border-orange-500",
    text: "text-orange-700 dark:text-orange-300",
  },
  end: {
    bg: "bg-red-100 dark:bg-red-950",
    border: "border-red-500",
    text: "text-red-700 dark:text-red-300",
  },
};

export const WorkflowNode = memo(({ data, selected }: NodeProps) => {
  const nodeType = data.nodeType || "action";
  const Icon = nodeIcons[nodeType as keyof typeof nodeIcons] || Play;
  const colors = nodeColors[nodeType as keyof typeof nodeColors] || nodeColors.action;

  return (
    <div
      className={`rounded-lg border-2 px-4 py-3 shadow-md transition-all ${
        colors.bg
      } ${colors.border} ${selected ? "ring-2 ring-primary ring-offset-2" : ""}`}
      style={{ minWidth: 150 }}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-gray-500"
      />

      <div className="flex items-center gap-2">
        <Icon className={`size-4 ${colors.text}`} />
        <div className={`font-medium text-sm ${colors.text}`}>
          {data.label || nodeType}
        </div>
      </div>

      {data.config?.description && (
        <div className="mt-1 text-muted-foreground text-xs">
          {data.config.description}
        </div>
      )}

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-gray-500"
      />
    </div>
  );
});

WorkflowNode.displayName = "WorkflowNode";
