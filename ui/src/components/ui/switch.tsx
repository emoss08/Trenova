/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as SwitchPrimitive from "@radix-ui/react-switch";
import * as React from "react";

import { cn } from "@/lib/utils";

type SwitchProps = React.ComponentProps<typeof SwitchPrimitive.Root> & {
  thumbClassName?: string;
  size?: "xs" | "sm" | "default" | "lg";
};

function Switch({
  className,
  thumbClassName,
  size = "default",
  ...props
}: SwitchProps) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "peer cursor-pointer data-[state=checked]:bg-blue-700 data-[state=unchecked]:bg-muted-foreground/40",
        "focus-visible:border-blue-600 focus-visible:ring-blue-600/20",
        "inline-flex shrink-0 items-center rounded-full border-2 border-transparent shadow-xs transition-all outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50",
        // Size variations
        size === "xs" && "h-3.5 w-6",
        size === "sm" && "h-4 w-7",
        size === "default" && "h-5 w-9",
        size === "lg" && "h-6 w-11",
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className={cn(
          "bg-white pointer-events-none block rounded-full ring-0 shadow-lg transition-transform",
          // Size variations for thumb
          size === "xs" &&
            "size-2.5 data-[state=checked]:translate-x-2.5 data-[state=unchecked]:translate-x-0",
          size === "sm" &&
            "size-3 data-[state=checked]:translate-x-3 data-[state=unchecked]:translate-x-0",
          size === "default" &&
            "size-4 data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0",
          size === "lg" &&
            "size-5 data-[state=checked]:translate-x-5 data-[state=unchecked]:translate-x-0",
          thumbClassName,
        )}
      />
    </SwitchPrimitive.Root>
  );
}

export { Switch };

