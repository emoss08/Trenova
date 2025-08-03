/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import {
  useCurrentPageFavorite,
  useToggleCurrentPageFavorite,
} from "@/hooks/use-favorites";
import { faStar } from "@fortawesome/pro-regular-svg-icons";
import { faStar as faStarSolid } from "@fortawesome/pro-solid-svg-icons";
import { NotificationCenter } from "./notification-center/notification-center";
import { Icon } from "./ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./ui/tooltip";

export function NavActions() {
  const { data: favoriteData, isLoading } = useCurrentPageFavorite();
  const { toggle, isPending } = useToggleCurrentPageFavorite();

  const isFavorite = favoriteData?.isFavorite ?? false;

  return (
    <TooltipProvider>
      <div className="flex items-center gap-1 text-sm">
        <NotificationCenter />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              onClick={toggle}
              disabled={isLoading || isPending}
            >
              <Icon
                icon={isFavorite ? faStarSolid : faStar}
                className={
                  isFavorite ? "text-yellow-500" : "text-muted-foreground"
                }
              />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            <p>{isFavorite ? "Remove from favorites" : "Add to favorites"}</p>
          </TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  );
}
