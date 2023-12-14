import * as React from "react";

import { cn } from "@/lib/utils";

export const Form = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => {
  return (
    <div
      className={cn("flex-1 overflow-visible", className)}
      ref={ref}
      {...props}
    />
  );
});

export const FormGroup = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => {
  return (
    <div
      ref={ref}
      className={cn(
        "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 my-4",
        className,
      )}
      {...props}
    />
  );
});

export const FormControl = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => {
  return (
    <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
      <div ref={ref} className={cn("min-h-[4em]", className)} {...props} />
    </div>
  );
});
