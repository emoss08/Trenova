/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import { nanoid } from "nanoid";
import { UserHoverCard } from "./user-hover-card";
import { COMMENT_TYPES } from "./utils";

interface CommentJsonRendererProps {
  content: Record<string, any>;
  className?: string;
}

export function CommentJsonRenderer({
  content,
  className,
}: CommentJsonRendererProps) {
  const renderNode = (node: any): React.ReactNode => {
    switch (node.type) {
      case "text":
        return <span key={nanoid()}>{node.text}</span>;

      case "mention":
        return (
          <UserHoverCard
            key={nanoid()}
            userId={node.attrs.id}
            username={node.attrs.label}
          />
        );

      case "commentType": {
        const commentType = COMMENT_TYPES.find(
          (t) => t.value === node.attrs.type,
        );
        if (!commentType) return null;

        return (
          <span
            key={nanoid()}
            className={cn(
              "inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium mr-0.5",
              commentType.className,
            )}
          >
            <Icon icon={commentType.icon} className="size-3" />
            <span>{commentType.label}</span>
          </span>
        );
      }

      case "paragraph":
        return (
          <p key={nanoid()} className="m-0">
            {node.content?.map(renderNode)}
          </p>
        );

      case "doc":
        return <>{node.content?.map(renderNode)}</>;

      default:
        console.warn("Unknown node type:", node.type);
        return null;
    }
  };

  return (
    <div className={cn("break-words", className)}>{renderNode(content)}</div>
  );
}
