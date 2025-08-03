/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn } from "@/lib/utils";
import { faSpinnerThird } from "@fortawesome/pro-regular-svg-icons";
import { Suspense } from "react";
import { Icon } from "./icons";

export type ComponentLoaderProps = {
  className?: string;
  message?: string;
  description?: string;
};

export function ComponentLoader({
  className,
  message,
  description,
}: ComponentLoaderProps) {
  return (
    <div
      className={cn("flex flex-col items-center justify-center p-2", className)}
    >
      <Icon
        icon={faSpinnerThird}
        size="1x"
        className="text-primary motion-safe:animate-spin"
      />
      <p className="text-foreground mt-2 text-sm">
        {message ?? "Loading data..."}
      </p>
      <p className="text-muted-foreground mt-2 text-sm">
        {description ?? "If this takes too long, please refresh the page."}
      </p>
    </div>
  );
}

export function SuspenseLoader({
  children,
  componentLoaderProps,
}: {
  children: React.ReactNode;
  componentLoaderProps?: ComponentLoaderProps;
}) {
  return (
    <Suspense fallback={<ComponentLoader {...componentLoaderProps} />}>
      {children}
    </Suspense>
  );
}
