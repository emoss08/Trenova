import * as SwitchPrimitive from "@radix-ui/react-switch";
import * as React from "react";

import { cn } from "@/lib/utils";

type SwitchProps = React.ComponentProps<typeof SwitchPrimitive.Root> & {
  thumbClassName?: string;
  size?: "xs" | "sm" | "default" | "lg";
  readOnly?: boolean;
};

function Switch({
  className,
  thumbClassName,
  size = "default",
  readOnly,
  ...props
}: SwitchProps) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "peer data-[state=checked]:bg-primary data-[state=unchecked]:bg-input focus-visible:border-ring focus-visible:ring-ring/50",
        "dark:data-[state=unchecked]:bg-input/80 inline-flex h-[1.15rem] w-8 shrink-0 items-center rounded-full border border-transparent",
        "shadow-xs transition-all outline-none focus-visible:ring-[3px] disabled:cursor-not-allowed disabled:opacity-50",
        readOnly && "cursor-default opacity-60 pointer-events-none",
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
          "bg-background dark:data-[state=unchecked]:bg-foreground dark:data-[state=checked]:bg-primary-foreground pointer-events-none block",
          "size-3 rounded-full ring-0 transition-transform data-[state=checked]:translate-x-[calc(100%-2px)] data-[state=unchecked]:translate-x-0",
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
