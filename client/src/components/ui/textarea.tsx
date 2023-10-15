import * as React from "react";

import { cn } from "@/lib/utils";
import { Label } from "./label";
import { AlertTriangle } from "lucide-react";

export interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        className={cn(
          "flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Textarea.displayName = "Textarea";

export { Textarea };

type ExtendedTextareaProps = TextareaProps & {
  error?: string;
  description?: string;
  label?: string;
  withAsterisk?: boolean;
};

const TextareaField = React.forwardRef<
  HTMLTextAreaElement,
  ExtendedTextareaProps
>(
  (
    { error, className, description, label, withAsterisk = false, ...props },
    ref,
  ) => {
    return (
      <>
        {label && (
          <Label
            className={cn("text-sm font-medium", withAsterisk && "required")}
          >
            {label}
          </Label>
        )}
        <div className="relative">
          <Textarea
            ref={ref}
            className={cn(
              "pr-10",
              error &&
                "ring-2 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
              className,
            )}
            {...props}
          />
          {error && (
            <div className="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
              <AlertTriangle className="h-5 w-5 text-red-500" />
            </div>
          )}
        </div>
        {description && (
          <p className="text-xs text-foreground/70">{description}</p>
        )}
        {error && <p className="mt-2 text-sm text-red-500">{error}</p>}
      </>
    );
  },
);

TextareaField.displayName = "TextareaField";

export { TextareaField };
