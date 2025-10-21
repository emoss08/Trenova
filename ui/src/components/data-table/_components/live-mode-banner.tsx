/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { faRefresh, faTimes } from "@fortawesome/pro-regular-svg-icons";

interface LiveModeBannerProps {
  show: boolean;
  newItemsCount: number;
  connected: boolean;
  onRefresh: () => void;
  onDismiss: () => void;
}

export function LiveModeBanner({
  show,
  newItemsCount,
  connected,
  onRefresh,
  onDismiss,
}: LiveModeBannerProps) {
  if (!show) return null;

  return (
    <div className="bg-blue-500/10 border border-blue-600/50 p-3 rounded-md flex justify-between items-center mb-4 w-full transition-all duration-200">
      <div className="flex items-center gap-3 text-blue-600">
        <div className="flex items-center gap-2">
          <div
            className={`w-2 h-2 rounded-full ${connected ? "bg-green-500" : "bg-red-500"}`}
          />
          <span className="text-sm font-medium">
            {newItemsCount} new item{newItemsCount !== 1 ? "s" : ""} available
          </span>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={onRefresh}
          className="text-blue-600 hover:text-blue-700 hover:bg-blue-500/20"
        >
          <Icon icon={faRefresh} className="w-4 h-4 mr-1" />
          Refresh
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={onDismiss}
          className="text-blue-600 hover:text-blue-700 hover:bg-blue-500/20"
        >
          <Icon icon={faTimes} className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}
