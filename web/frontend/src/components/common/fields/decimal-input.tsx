/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
      const selectionStart = e.currentTarget.selectionStart;
      const selectionEnd = e.currentTarget.selectionEnd;

      // Allow normal behavior if text is highlighted (selection range is not zero)
      if (selectionStart !== selectionEnd || (e.ctrlKey && e.key === "a")) {
        return;
      }

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
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500 bg-red-500 bg-opacity-20",
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
