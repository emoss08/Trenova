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

import { ErrorMessage, Input } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { cn } from "@/lib/utils";
import { AlertTriangle } from "lucide-react";
import * as React from "react";
import { useInteractOutside } from "react-aria";
import { HexColorPicker } from "react-colorful";
import { ColorInputBaseProps } from "react-colorful/dist/types";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";

export type ColorFieldProps<T extends FieldValues> = {
  label?: string;
  description?: string;
} & UseControllerProps<T> &
  Omit<ColorInputBaseProps, "onChange">;

export function ColorField<T extends FieldValues>({
  ...props
}: ColorFieldProps<T>) {
  const [showPicker, setShowPicker] = React.useState<boolean>(false);
  const popoverRef = React.useRef<HTMLDivElement>(null);
  const { field, fieldState } = useController(props);

  useInteractOutside({
    ref: popoverRef,
    onInteractOutside: (e) => {
      setShowPicker(false);
    },
  });

  // Handler for HexColorPicker
  const handleColorPickerChange = (newColor: string) => {
    field.onChange(newColor);
  };

  // Handler for Input
  const handleInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    field.onChange(event.target.value);
  };

  return (
    <div className="relative">
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
      <div className="relative w-full" onClick={() => setShowPicker(true)}>
        <Input
          {...field}
          className={cn(
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
            props.className,
          )}
          onChange={handleInputChange}
          {...props}
        />
        {field.value && (
          <>
            <div className="bg-border absolute inset-y-0 right-10 my-2 h-6 w-[1px]" />
            <div
              className="absolute right-0 top-0 mx-2 mt-2 size-5 rounded-xl"
              style={{ backgroundColor: field.value }}
            />
          </>
        )}
        {fieldState.invalid && (
          <>
            <div className="pointer-events-none absolute inset-y-0 right-0 mr-2.5 mt-2.5">
              <AlertTriangle size={15} className="text-red-500" />
            </div>
            <ErrorMessage formError={fieldState.error?.message} />
          </>
        )}
        {props.description && !fieldState.invalid && (
          <p className="text-foreground/70 text-xs">{props.description}</p>
        )}
      </div>
      {showPicker && (
        <div ref={popoverRef} className="z-100 absolute mt-2 w-auto">
          <HexColorPicker
            color={field.value}
            onChange={handleColorPickerChange}
          />
        </div>
      )}
    </div>
  );
}
