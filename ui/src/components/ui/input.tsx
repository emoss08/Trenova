import * as React from "react";

import { cn } from "@/lib/utils";

export type InputProps = React.ComponentProps<"input"> & {
  isInvalid?: boolean;
  icon?: React.ReactNode;
};

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, isInvalid, icon, ...props }, ref) => {
    return (
      <div className="relative">
        {icon && (
          <div
            className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3"
            aria-hidden="true"
          >
            {icon}
          </div>
        )}
        <input
          type={type}
          className={cn(
            "border-muted-foreground/20 bg-muted flex h-8 w-full rounded-md border px-2 py-1 text-sm",
            "file:border-0 file:bg-transparent file:text-sm file:font-medium",
            "placeholder:text-muted-foreground",
            "disabled:cursor-not-allowed disabled:opacity-50",
            "focus-visible:border-blue-600 focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-blue-600/20",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            // Invalid state
            isInvalid &&
              "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
            icon && "pl-10",
            className,
          )}
          ref={ref}
          {...props}
        />
      </div>
    );
  },
);

Input.displayName = "Input";

export { Input };

