import * as React from "react";

import { cn } from "@/lib/utils";
import { Label } from "./label";
import { AlertTriangle } from "lucide-react";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";

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
  description?: string;
  label?: string;
  ref?: React.ForwardedRef<HTMLTextAreaElement>;
};

export function TextareaField<T extends FieldValues>({
  ...props
}: ExtendedTextareaProps & UseControllerProps<T>) {
  const { field, fieldState } = useController(props);
  const { label, id, className } = props;
  return (
    <>
      {label && (
        <Label
          className={cn(
            "text-sm font-medium",
            props.rules?.required && "required",
          )}
          htmlFor={id}
        >
          {label}
        </Label>
      )}
      <div className="relative">
        <Textarea
          className={cn(
            "pr-10",
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
            className,
          )}
          {...props}
          {...field}
        />
        {fieldState.error?.message && (
          <>
            <div className="pointer-events-none absolute inset-y-0 top-0 right-0 mt-3 mr-3">
              <AlertTriangle size={15} className="text-red-500" />
            </div>
            <p className="text-xs text-red-600">{fieldState.error?.message}</p>
          </>
        )}
        {props.description && !fieldState.error?.message && (
          <p className="text-xs text-foreground/70">{props.description}</p>
        )}
      </div>
    </>
  );
}
