import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { LazyImage } from "./image";

export function UserHoverCard({
  userId,
  username,
}: {
  userId?: string;
  username: string;
}) {
  const [hoveredUserId, setHoveredUserId] = useState<string | null>(null);

  const { data: hoveredUserData } = useQuery({
    ...queries.user.detail(hoveredUserId || ""),
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
      <HoverCardTrigger
        render={
          <span className="cursor-pointer font-medium text-blue-600 hover:underline dark:text-blue-400">
            @{username}
          </span>
        }
      />
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
