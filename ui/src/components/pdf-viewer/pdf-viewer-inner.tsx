/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";
import type React from "react";

export function PDFViewerInner({
  className,
  children,
  setContainerRef,
}: {
  className: string;
  setContainerRef: React.Dispatch<React.SetStateAction<HTMLElement | null>>;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex flex-col md:flex-row h-full w-full bg-transparent rounded-lg",
        className,
      )}
      ref={setContainerRef}
    >
      <div className="flex-1 flex flex-col overflow-hidden">{children}</div>
    </div>
  );
}
