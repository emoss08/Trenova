/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import type { ReactNodeViewProps } from "@tiptap/react";
import { NodeViewWrapper } from "@tiptap/react";
import { COMMENT_TYPES } from "../utils";

export const CommentTypeComponent = ({
  node,
}: ReactNodeViewProps<HTMLSpanElement>) => {
  const type = node.attrs.type;
  const commentType = COMMENT_TYPES.find((t) => t.value === type);

  if (!commentType) {
    return null;
  }

  return (
    <NodeViewWrapper
      as="span"
      className={cn(
        "inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium",
        "select-none cursor-default",
        commentType.className,
      )}
      contentEditable={false}
    >
      <Icon icon={commentType.icon} className="size-3" />
      <span>{commentType.label}</span>
    </NodeViewWrapper>
  );
};
