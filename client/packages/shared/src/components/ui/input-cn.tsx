import { Input as InputPrimitive } from "@base-ui/react/input";
import * as React from "react";

import { cn } from "@trenova/shared/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        "ui-field h-7 w-full min-w-0 px-2 py-0.5 text-base outline-none file:inline-flex file:h-5.5 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground ui-focus-ring disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-60 aria-invalid:border-danger aria-invalid:[--ring:var(--ring-danger)] md:text-sm",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
