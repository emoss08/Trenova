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
import {
  FieldValues,
  useController,
  UseControllerProps,
} from "react-hook-form";
import { Label } from "@/components/common/fields/label";
import { cn } from "@/lib/utils";
import { AlertTriangle } from "lucide-react";
import { ExtendedInputProps, Input } from "@/components/common/fields/input";

const controlKeys = ["Backspace", "Delete", "ArrowLeft", "ArrowRight", "Tab"];

type DecimalFieldProps = ExtendedInputProps & {
  precision?: number;
};

function isKeyAllowed(
  e: React.KeyboardEvent<HTMLInputElement>,
  field: any,
  precision: number,
) {
  if (controlKeys.includes(e.key)) {
    return true;
  }

  const isNumericOrDot = /[0-9.]/.test(e.key);
  const isSingleDot = e.key === "." && !field.value.includes(".");
  const isWithinPrecision =
    field.value.includes(".") && field.value.split(".")[1].length < precision;

  return (
    isNumericOrDot &&
    ((e.key === "." && isSingleDot) ||
      (e.key !== "." && (isWithinPrecision || !field.value.includes("."))))
  );
}

export function DecimalField<T extends FieldValues>({
  precision = 2,
  ...props
}: DecimalFieldProps & UseControllerProps<T>) {
  const { field, fieldState } = useController(props);

  const handleKeyDown = React.useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (!isKeyAllowed(e, field, precision)) {
        e.preventDefault();
      }
    },
    [field, precision],
  );

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
          type="text"
          className={cn(
            "pr-10",
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
          )}
          onKeyDown={handleKeyDown}
          {...field}
          {...props}
          aria-label={props.label}
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
