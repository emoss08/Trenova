/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { LazyImage } from "@/components/ui/image";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

export function UserHoverCard({
  userId,
  username,
}: {
  userId?: string;
  username: string;
}) {
  const [hoveredUserId, setHoveredUserId] = useState<string | null>(null);

  const { data: hoveredUserData } = useQuery({
    ...queries.user.getUserById(hoveredUserId || ""),
    enabled: !!hoveredUserId,
  });

  return (
    <HoverCard
      onOpenChange={(open) => {
        if (open && userId) {
          setHoveredUserId(userId);
        } else {
          setHoveredUserId(null);
        }
      }}
    >
      <HoverCardTrigger asChild>
        <span className="text-blue-600 dark:text-blue-400 font-medium cursor-pointer hover:underline">
          @{username}
        </span>
      </HoverCardTrigger>
      <HoverCardContent className="w-64">
        {userId && hoveredUserId === userId ? (
          <div className="space-y-2">
            <div className="flex gap-2">
              <LazyImage
                src={`https://avatar.vercel.sh/${hoveredUserData?.username || username}.svg`}
                alt={hoveredUserData?.name || username}
                className="size-8 rounded-full"
              />
              <div className="flex flex-col text-xs">
                <h4 className="font-semibold">
                  {hoveredUserData?.name || "Loading..."}
                </h4>
                <p className="text-blue-600 dark:text-blue-400">
                  @{hoveredUserData?.username || username}
                </p>
              </div>
            </div>
            {hoveredUserData?.emailAddress && (
              <div className="text-xs text-muted-foreground">
                {hoveredUserData.emailAddress}
              </div>
            )}
          </div>
        ) : (
          <div className="text-sm text-muted-foreground">User not found</div>
        )}
      </HoverCardContent>
    </HoverCard>
  );
}
