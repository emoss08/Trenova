import * as React from "react";

import { cn } from "@/lib/utils";

export type TextareaProps = React.ComponentProps<"textarea"> & {
  isInvalid?: boolean;
};

function Textarea({ className, isInvalid, ...props }: TextareaProps) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "flex min-h-[60px] w-full rounded-md border border-muted-foreground/20 bg-muted px-3 py-2 text-base",
        "shadow-xs placeholder:text-muted-foreground focus-visible:outline-hidden",
        "focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 md:text-sm",
        "focus-visible:border-blue-600 focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-blue-600/20",
        "transition-[border-color,box-shadow] duration-200 ease-in-out",
        isInvalid &&
          "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
        className,
      )}
      {...props}
    />
  );
}

export { Textarea };

