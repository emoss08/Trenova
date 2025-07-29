/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { LazyImage } from "@/components/ui/image";
import { ShipmentCommentSchema } from "@/lib/schemas/shipment-comment-schema";
import { cn } from "@/lib/utils";
import { formatDistanceToNow } from "date-fns";
import { Trash2Icon } from "lucide-react";

export function CommentRow({
  index,
  shipmentComment,
  isLast,
  onDelete,
}: {
  index: number;
  shipmentComment: ShipmentCommentSchema;
  isLast: boolean;
  onDelete?: (index: number) => void;
}) {
  const timeAgo = shipmentComment.createdAt
    ? formatDistanceToNow(new Date(shipmentComment.createdAt * 1000), {
        addSuffix: true,
      })
    : "Unknown time";

  // Function to render content with highlighted mentions
  const renderContentWithMentions = (text: string) => {
    const mentionRegex = /@(\w+)/g;
    const parts = text.split(mentionRegex);

    return parts.map((part, idx) => {
      // Even indices are regular text, odd indices are usernames
      if (idx % 2 === 0) {
        return <span key={idx}>{part}</span>;
      } else {
        return (
          <span
            key={idx}
            className="text-blue-600 dark:text-blue-400 font-medium cursor-pointer hover:underline"
          >
            @{part}
          </span>
        );
      }
    });
  };

  return (
    <div
      className={cn(
        "group relative flex gap-3 py-4",
        !isLast && "border-b border-border/50",
      )}
    >
      {/* User Avatar */}
      <div className="flex-shrink-0">
        <LazyImage
          src={`https://avatar.vercel.sh/${shipmentComment.user?.username || "anonymous"}.svg`}
          alt={shipmentComment.user?.name || "User"}
          className="size-8 rounded-full"
        />
      </div>

      {/* Comment Content */}
      <div className="flex-1 space-y-1">
        {/* User info and timestamp */}
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">
            {shipmentComment.user?.name || "Unknown User"}
          </span>
          {shipmentComment.isHighPriority && (
            <span className="inline-flex items-center rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-700 ring-1 ring-inset ring-red-600/20 dark:bg-red-400/10 dark:text-red-400 dark:ring-red-400/20">
              High Priority
            </span>
          )}
          <span className="text-xs text-muted-foreground">{timeAgo}</span>
        </div>

        {/* Comment text */}
        <div className="text-sm text-foreground">
          <p className="break-words">
            {renderContentWithMentions(shipmentComment.comment)}
          </p>
        </div>

        {/* Mentioned users */}
        {shipmentComment.mentionedUsers &&
          shipmentComment.mentionedUsers.length > 0 && (
            <div className="flex items-center gap-1 pt-1">
              <span className="text-xs text-muted-foreground">Mentioned:</span>
              <div className="flex gap-1">
                {shipmentComment.mentionedUsers.map((mention) => (
                  <span
                    key={mention.id}
                    className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700 ring-1 ring-inset ring-blue-700/10 dark:bg-blue-400/10 dark:text-blue-400 dark:ring-blue-400/30"
                  >
                    @{mention.mentionedUser?.username || "unknown"}
                  </span>
                ))}
              </div>
            </div>
          )}
      </div>

      {/* Delete button - only show if onDelete is provided */}
      {onDelete && (
        <div className="absolute right-0 top-4 opacity-0 transition-opacity group-hover:opacity-100">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-muted-foreground hover:text-destructive"
            onClick={() => onDelete(index)}
          >
            <Trash2Icon className="h-4 w-4" />
            <span className="sr-only">Delete comment</span>
          </Button>
        </div>
      )}
    </div>
  );
}
