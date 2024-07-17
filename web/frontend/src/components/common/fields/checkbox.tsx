/**
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



import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import * as React from "react";

import { cn } from "@/lib/utils";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  Controller,
  FieldValues,
  useController,
  UseControllerProps,
} from "react-hook-form";
import { ErrorMessage } from "./error-message";

const Checkbox = React.forwardRef<
  React.ElementRef<typeof CheckboxPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>
>(({ className, ...props }, ref) => (
  <CheckboxPrimitive.Root
    ref={ref}
    className={cn(
      "peer h-4 w-4 shrink-0 rounded-sm border border-primary shadow focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground",
      className,
    )}
    {...props}
  >
    <CheckboxPrimitive.Indicator
      className={cn("flex items-center justify-center text-current")}
    >
      <FontAwesomeIcon icon={faCheck} className="size-3 font-bold" />
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
));
Checkbox.displayName = CheckboxPrimitive.Root.displayName;

export { Checkbox };

type CheckboxInputProps = CheckboxPrimitive.CheckboxProps &
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root> & {
    formError?: string;
    description?: string;
    label?: string;
  };

export function CheckboxInput<T extends FieldValues>({
  ...props
}: CheckboxInputProps & UseControllerProps<T>) {
  const { fieldState } = useController(props);

  const { label, description, id, className } = props;

  return (
    <label
      htmlFor={id}
      className={cn("items-top flex cursor-pointer space-x-2", className)}
    >
      <Controller
        name={props.name}
        control={props.control}
        render={({ field }) => (
          <Checkbox
            {...field}
            onCheckedChange={(e) => {
              field.onChange(e);
            }}
            checked={field.value as boolean}
            id={id} // Ensure the checkbox has the same id as the label's htmlFor
          />
        )}
      />
      <div className="grid gap-1.5 leading-none">
        {label && (
          <span className="select-none text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
            {label}
          </span>
        )}
        {description && (
          <p className="select-none text-wrap text-sm text-muted-foreground">
            {description}
          </p>
        )}
        {fieldState.invalid && (
          <ErrorMessage formError={fieldState.error?.message} />
        )}
      </div>
    </label>
  );
}
