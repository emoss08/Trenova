/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
import * as React from "react";

import { cn } from "@/lib/utils";
import { AlertTriangle } from "lucide-react";
import { Label } from "./label";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus:ring-1 focus:ring-inset focus:ring-foreground disabled:cursor-not-allowed disabled:opacity-50 sm:text-sm sm:leading-6",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Input.displayName = "Input";

export { Input };

export type ExtendedInputProps = Omit<InputProps, "name"> & {
  description?: string;
  label?: string;
  ref?: React.ForwardedRef<HTMLInputElement>;
};

export function InputField<T extends FieldValues>({
  ...props
}: ExtendedInputProps & UseControllerProps<T>) {
  const { field, fieldState } = useController(props);

  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium",
            props.rules?.required && "required",
          )}
          htmlFor={props.id}
        >
          {props.label}
        </Label>
      )}
      <div className="relative">
        <Input
          {...field}
          className={cn(
            "pr-10",
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
          )}
          {...props}
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

export function FileField<T extends FieldValues>({
  ...props
}: ExtendedInputProps & UseControllerProps<T>) {
  const { field, fieldState } = useController(props);

  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium",
            props.rules?.required && "required",
          )}
        >
          {props.label}
        </Label>
      )}
      <div className="relative">
        <Input
          type="file"
          className={cn(
            "pr-10",
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
            props.className,
          )}
          // value={field.value}
          onChange={(e) => {
            const value = e.target.files;

            console.log("Field Value", value);
            if (value) {
              field.onChange(value);
            }
          }}
          {...props}
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
