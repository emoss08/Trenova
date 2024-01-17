/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */
import { cn } from "@/lib/utils";
import { AlertTriangle } from "lucide-react";
import * as React from "react";
import {
  FieldValues,
  useController,
  UseControllerProps,
} from "react-hook-form";
import { ErrorMessage } from "./input";
import { Label } from "./label";

export interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        className={cn(
          "flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:ring-1 focus-visible:outline-none focus:ring-inset focus:ring-foreground disabled:cursor-not-allowed disabled:opacity-50 read-only:cursor-not-allowed read-only:opacity-50",
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
        {fieldState.invalid && (
          <>
            <div className="pointer-events-none absolute inset-y-0 right-0 mr-3 mt-3">
              <AlertTriangle size={15} className="text-red-500" />
            </div>
            <ErrorMessage formError={fieldState.error?.message} />
          </>
        )}
        {props.description && !fieldState.invalid && (
          <p className="text-foreground/70 text-xs">{props.description}</p>
        )}
      </div>
    </>
  );
}
