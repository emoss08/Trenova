/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyImage } from "@/components/ui/image";
import { ShipmentCommentSchema } from "@/lib/schemas/shipment-comment-schema";
import { cn } from "@/lib/utils";
import { formatDistanceToNow } from "date-fns";
import { useCallback, useMemo } from "react";
import { COMMENT_TYPES } from "../utils";
import { UserHoverCard } from "./user-hover-card";

export function CommentContent({
  shipmentComment,
  isLast,
}: {
  shipmentComment: ShipmentCommentSchema;
  isLast: boolean;
}) {
  const timeAgo = shipmentComment.createdAt
    ? formatDistanceToNow(new Date(shipmentComment.createdAt * 1000), {
        addSuffix: true,
      })
    : "Unknown time";

  const usernameToUserIdMap = useMemo(() => {
    const map = new Map<string, string>();
    if (shipmentComment.mentionedUsers) {
      shipmentComment.mentionedUsers.forEach((mention) => {
        const username = mention.mentionedUser?.username;
        const userId = mention.mentionedUserId;
        if (username && userId) {
          map.set(username.toLowerCase(), userId);
        }
      });
    }
    return map;
  }, [shipmentComment.mentionedUsers]);

  const renderContent = useCallback(
    (text: string) => {
      let slashCommandElement = null;
      let remainingText = text;

      for (const type of COMMENT_TYPES) {
        const regex = new RegExp(`^/${type.label}(\\s|$)`, "i");
        const match = text.match(regex);
        if (match) {
          slashCommandElement = (
            <span
              className={cn(
                "inline-flex items-center px-2 py-0.5 rounded-sm font-medium text-xs mr-1",
                type.className,
              )}
            >
              <span>{type.label}</span>
            </span>
          );
          remainingText = text.slice(match[0].length);
          break;
        }
      }

      const mentionRegex = /@(\w+)/g;
      const parts = remainingText.split(mentionRegex);

      const renderedParts = parts.map((part, idx) => {
        if (idx % 2 === 0) {
          return <span key={idx}>{part}</span>;
        } else {
          const userId = usernameToUserIdMap.get(part.toLowerCase());

          return <UserHoverCard key={idx} userId={userId} username={part} />;
        }
      });

      return (
        <>
          {slashCommandElement}
          {renderedParts}
        </>
      );
    },
    [usernameToUserIdMap],
  );
  return (
    <div
      className={cn(
        "group relative flex gap-3 py-4",
        !isLast && "border-b border-border/50",
      )}
    >
      <div className="flex-shrink-0">
        <LazyImage
          src={`https://avatar.vercel.sh/${shipmentComment.user?.username || "anonymous"}.svg`}
          alt={shipmentComment.user?.name || "User"}
          className="size-6 rounded-full"
        />
      </div>
      <div className="flex-1 space-y-1">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">
            {shipmentComment.user?.name || "Unknown User"}
          </span>
          <span className="text-xs text-muted-foreground">{timeAgo}</span>
        </div>

        <div className="text-sm text-foreground">
          <p className="break-words">
            {renderContent(shipmentComment.comment)}
          </p>
        </div>
      </div>
    </div>
  );
}
