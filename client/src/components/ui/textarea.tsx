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
          "flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:ring-1 focus-visible:outline-none focus:ring-inset focus:ring-foreground disabled:cursor-not-allowed disabled:opacity-50",
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
  formError?: string;
  description?: string;
  label?: string;
  withAsterisk?: boolean;
};

const TextareaField = React.forwardRef<
  HTMLTextAreaElement,
  ExtendedTextareaProps
>(
  (
    {
      formError,
      className,
      description,
      label,
      withAsterisk = false,
      ...props
    },
    ref,
  ) => {
    return (
      <>
        {label && (
          <Label
            className={cn("text-sm font-medium", withAsterisk && "required")}
            htmlFor={props.id}
          >
            {label}
          </Label>
        )}
        <div className="relative">
          <Textarea
            ref={ref}
            className={cn(
              "pr-10",
              formError &&
                "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
              className,
            )}
            {...props}
          />
          {formError && (
            <>
              <div className="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
                <AlertTriangle className="h-5 w-5 text-red-500" />
              </div>
              <p className="text-xs text-red-500">{formError}</p>
            </>
          )}
        </div>
        {description && !formError && (
          <p className="text-xs text-foreground/70">{description}</p>
        )}
      </>
    );
  },
);

TextareaField.displayName = "TextareaField";

export { TextareaField };
