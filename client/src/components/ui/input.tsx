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

import { AlertTriangle } from "lucide-react";
import { cn } from "@/lib/utils";

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
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

interface ExtendedInputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: string;
  // You can add other custom props if needed
}

const InputField = React.forwardRef<HTMLInputElement, ExtendedInputProps>(
  ({ error, className, ...props }, ref) => {
    return (
      <div className="relative">
        <Input
          ref={ref}
          className={cn("pr-10", error && "border-red-600", className)}
          {...props}
        />
        {error && (
          <>
            <div className="absolute top-0 right-0 mt-2 mr-3 text-red-600">
              <AlertTriangle size={20} />
            </div>
            <p className="mt-2 px-1 text-xs text-red-600">{error}</p>
          </>
        )}
      </div>
    );
  },
);

InputField.displayName = "InputField";

export { InputField };

const PasswordField = React.forwardRef<HTMLInputElement, ExtendedInputProps>(
  ({ error, className, ...props }, ref) => {
    return (
      <div className="relative">
        <Input
          ref={ref}
          className={cn("pr-10", error && "border-red-600", className)}
          {...props}
        />
        {error && (
          <>
            <div className="absolute top-0 right-0 mt-2 mr-3 text-red-600">
              <AlertTriangle size={20} />
            </div>
            <p className="mt-2 px-1 text-xs text-red-600">{error}</p>
          </>
        )}
      </div>
    );
  },
);

PasswordField.displayName = "InputField";

export { PasswordField };
