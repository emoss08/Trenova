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

export type ExtendedInputProps = InputProps & {
  formError?: string;
  description?: string;
  label?: string;
  withAsterisk?: boolean;
};

const InputField = React.forwardRef<HTMLInputElement, ExtendedInputProps>(
  (
    { formError, className, description, label, withAsterisk, ...props },
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
          <Input
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
              <div className="pointer-events-none absolute inset-y-0 top-0 right-0 mt-2 mr-3">
                <AlertTriangle size={20} className="text-red-500" />
              </div>
              <p className="text-xs text-red-600">{formError}</p>
            </>
          )}
          {description && !formError && (
            <p className="text-xs text-foreground/70">{description}</p>
          )}
        </div>
      </>
    );
  },
);

InputField.displayName = "InputField";

export { InputField };

const PasswordField = React.forwardRef<HTMLInputElement, ExtendedInputProps>(
  ({ formError, className, label, withAsterisk, ...props }, ref) => {
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
          <Input
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
              <div className="absolute top-0 right-0 mt-2 mr-3 text-red-600">
                <AlertTriangle size={20} />
              </div>
              <p className="mt-2 px-1 text-xs text-red-600">{formError}</p>
            </>
          )}
        </div>
      </>
    );
  },
);

PasswordField.displayName = "InputField";

export { PasswordField };

const FileField = React.forwardRef<
  HTMLInputElement,
  Omit<ExtendedInputProps, "placeholder">
>(
  (
    { formError, className, label, description, withAsterisk, ...props },
    ref,
  ) => {
    return (
      <div className="relative">
        {label && (
          <Label
            className={cn("text-sm font-medium", withAsterisk && "required")}
          >
            {label}
          </Label>
        )}
        <Input
          ref={ref}
          type="file"
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
            <div className="pointer-events-none absolute inset-y-0 top-0 right-0 mt-2 mr-3">
              <AlertTriangle size={20} className="text-red-500" />
            </div>
            <p className="mt-2 px-1 text-xs text-red-600">{formError}</p>
          </>
        )}
        {description && !formError && (
          <p className="text-xs text-foreground/70">{description}</p>
        )}
      </div>
    );
  },
);

FileField.displayName = "FileField";

export { FileField };
