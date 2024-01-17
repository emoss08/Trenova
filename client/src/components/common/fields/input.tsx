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
import * as React from "react";

import { cn } from "@/lib/utils";
import { AlertTriangle } from "lucide-react";
import {
  FieldValues,
  useController,
  UseControllerProps,
} from "react-hook-form";
import { Label } from "./label";

export function ErrorMessage({ formError }: { formError?: string }) {
  return (
    <div className="mt-2 inline-block rounded bg-red-50 px-2 py-1 text-xs  leading-tight text-red-500 dark:bg-red-300 dark:text-red-800 ">
      {formError ? formError : "An Error has occurred. Please try again."}
    </div>
  );
}

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus:ring-1 focus:ring-inset focus:ring-foreground disabled:cursor-not-allowed disabled:opacity-50 read-only:cursor-not-allowed read-only:opacity-50 sm:text-sm sm:leading-6",
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
  icon?: React.ReactNode;
};

export function InputField<T extends FieldValues>({
  icon,
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
        {icon && (
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
            {icon}
          </div>
        )}
        <Input
          {...field}
          className={cn(
            icon && "pl-10",
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
            props.className,
          )}
          {...props}
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
          onChange={(e) => {
            const value = e.target.files;
            if (value) {
              field.onChange(value);
            }
          }}
          {...props}
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

export function TimeField<T extends FieldValues>({
  ...props
}: ExtendedInputProps & UseControllerProps<T>) {
  // Forces time input to be in 24 hour format (ex: 17:00:00)
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
          type="time"
          step="1" // Include this to allow seconds in the format HH:MM:SS
          className={cn(
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
            props.className,
          )}
          {...field}
          {...props}
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
