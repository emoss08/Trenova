/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";
import { faFile, faImage } from "@fortawesome/pro-regular-svg-icons";
import React from "react";
import { Icon } from "../ui/icons";

export const DocumentUploadSkeleton = React.memo(
  function DocumentUploadSkeleton({
    isHovering = false,
  }: {
    isHovering?: boolean;
  }) {
    // Memoize class names
    const leftDocClassNames = cn(
      "absolute z-10 bg-foreground/20 rounded-md h-7 w-24 p-1 transform duration-400",
      isHovering
        ? "translate-x-[-24px] translate-y-[-8px] rotate-[-5deg]"
        : "translate-x-[-12px] translate-y-[-14px] rotate-0",
    );

    const rightDocClassNames = cn(
      "absolute z-10 bg-foreground/20 rounded-md h-7 w-24 p-1 transform duration-400",
      isHovering
        ? "translate-x-[24px] translate-y-[8px] rotate-[5deg]"
        : "translate-x-[12px] translate-y-[18px] rotate-0",
    );

    return (
      <div className="flex items-center justify-center relative h-20 w-24 bg-background dark:bg-background/50 rounded-md size-full">
        {/* Left document */}
        <div className={leftDocClassNames}>
          <div className="flex items-center gap-x-1">
            <div className="flex items-center justify-center bg-blue-500 rounded-sm size-5 p-1">
              <Icon icon={faFile} className="size-4 text-white" />
            </div>
            <div className="flex flex-col gap-0.5 size-full">
              <div className="w-full h-1.5 bg-muted-foreground rounded-md" />
              <div className="w-10 h-1.5 bg-muted-foreground/50 rounded-md" />
            </div>
          </div>
        </div>

        {/* Right document */}
        <div className={rightDocClassNames}>
          <div className="flex items-center gap-x-1">
            <div className="flex items-center justify-center bg-pink-500 rounded-sm size-5 p-1">
              <Icon icon={faImage} className="size-4 text-white" />
            </div>
            <div className="flex flex-col gap-0.5 size-full">
              <div className="w-full h-1.5 bg-muted-foreground rounded-md" />
              <div className="w-10 h-1.5 bg-muted-foreground/50 rounded-md" />
            </div>
          </div>
        </div>
      </div>
    );
  },
);
