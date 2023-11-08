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

import {
  FieldValues,
  useController,
  UseControllerProps,
} from "react-hook-form";
import { ColorInputBaseProps } from "react-colorful/dist/types";
import * as React from "react";
import useClickOutside, { cn } from "@/lib/utils";
import { Label } from "@/components/common/fields/label";
import { HexColorInput, HexColorPicker } from "react-colorful";
import { AlertTriangle } from "lucide-react";

type ColorFieldProps<T extends FieldValues> = {
  label?: string;
  description?: string;
} & UseControllerProps<T> &
  ColorInputBaseProps;

export function ColorField<T extends FieldValues>({
  ...props
}: ColorFieldProps<T>) {
  const [showPicker, setShowPicker] = React.useState<boolean>(false);
  const popoverRef = React.useRef<HTMLDivElement>(null);
  const { field, fieldState } = useController(props);

  const close = React.useCallback(() => setShowPicker(false), []);
  useClickOutside(popoverRef, close);

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
        <HexColorInput
          {...field}
          className={cn(
            "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus:ring-1 focus:ring-inset focus:ring-foreground disabled:cursor-not-allowed disabled:opacity-50 sm:text-sm sm:leading-6",
            fieldState.invalid &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
            props.className,
          )}
          {...props}
        />
        <div className="absolute inset-y-0 right-10 my-2 h-6 w-[1px] bg-gray-300" />
        <div
          className="absolute right-0 top-0 my-2.5 mx-2 h-5 w-5 rounded-xl"
          style={{ backgroundColor: props.color }}
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
      {showPicker && (
        <div ref={popoverRef} className="absolute z-1000 w-auto">
          <HexColorPicker color={props.color} onChange={props.onChange} />
        </div>
      )}
    </div>
  );
}
