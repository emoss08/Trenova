import * as React from "react";

import { cn } from "@/lib/utils";

export type InputProps = React.ComponentProps<"input"> & {
  isInvalid?: boolean;
  hasWarning?: boolean;
  icon?: React.ReactNode;
  sideText?: string;
  rightElement?: React.ReactNode;
};

function Input({
  className,
  type,
  isInvalid,
  hasWarning,
  icon,
  sideText,
  readOnly,
  rightElement,
  ...props
}: InputProps) {
  return (
    <div className="relative w-full">
      {icon && (
        <div
          className="pointer-events-none absolute inset-y-0 left-0 z-10 flex items-center pl-2"
          aria-hidden="true"
        >
          {icon}
        </div>
      )}
      <div className="relative flex rounded-md">
        <input
          type={type}
          readOnly={readOnly}
          className={cn(
            "border-muted-foreground/20 bg-primary/5 flex h-7 w-full rounded-md border px-2 py-1 text-xs",
            "file:border-0 file:bg-transparent file:text-sm file:font-medium",
            "placeholder:text-muted-foreground placeholder:truncate",
            "disabled:cursor-not-allowed disabled:opacity-50",
            "read-only:cursor-default read-only:text-muted-foreground",
            "focus-visible:border-foreground focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-foreground/20",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            // Read only state
            readOnly && "cursor-not-allowed opacity-60 pointer-events-none",
            // Invalid state
            isInvalid &&
              "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
            // Warning state (only if not invalid)
            !isInvalid &&
              hasWarning &&
              "border-yellow-500 bg-yellow-500/10 ring-0 ring-yellow-500 placeholder:text-yellow-600 focus:outline-hidden focus-visible:border-yellow-600 focus-visible:ring-4 focus-visible:ring-yellow-400/20",
            icon && "pl-7",
            (sideText || rightElement) && "pr-14",
            className,
          )}
          {...props}
        />
        {sideText && (
          <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3 text-xs text-muted-foreground">
            {sideText}
          </div>
        )}
        {rightElement && (
          <div className="absolute inset-y-0 right-0 flex items-center pr-1">
            {rightElement}
          </div>
        )}
      </div>
    </div>
  );
}

export { Input };
