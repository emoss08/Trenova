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
import { ExtendedInputProps, Input } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { cn } from "@/lib/utils";
import * as React from "react";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { FieldDescription } from "./components";
import { FieldErrorMessage } from "./error-message";

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

  const { label, rules } = props;

  return (
    <>
      <span className="space-x-1">
        {label && <Label className="text-sm font-medium">{label}</Label>}
        {rules?.required && <span className="text-red-500">*</span>}
      </span>
      <div className="relative">
        <Input
          type="number"
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
        {fieldState.invalid && (
          <FieldErrorMessage formError={fieldState.error?.message} />
        )}
        {props.description && !fieldState.invalid && (
          <FieldDescription description={props.description} />
        )}
      </div>
    </>
  );
}
