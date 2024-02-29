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

import { Input } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { faPaintBrush } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import * as React from "react";
import { useInteractOutside } from "react-aria";
import { HexColorPicker } from "react-colorful";
import { ColorInputBaseProps } from "react-colorful/dist/types";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { FieldErrorMessage } from "./error-message";

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
    onInteractOutside: () => {
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
          <FieldErrorMessage formError={fieldState.error?.message} />
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

export function GradientPicker({
  background,
  setBackground,
  className,
}: {
  background: string;
  setBackground: (background: string) => void;
  className?: string;
}) {
  // TODO(WOLFRED): Add more colors
  const solids = [
    "#E2E2E2",
    "#ff75c3",
    "#ffa647",
    "#ffe83f",
    "#9fff5b",
    "#70e2ff",
    "#cd93ff",
    "#09203f",
  ];

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant={"outline"}
          className={cn(
            "w-full justify-start text-left font-normal truncate",
            !background && "text-muted-foreground",
            className,
          )}
        >
          <div className="flex w-full items-center gap-2">
            {background ? (
              <div
                className="size-4 rounded !bg-cover !bg-center transition-all"
                style={{ background }}
              ></div>
            ) : (
              <FontAwesomeIcon icon={faPaintBrush} className="size-4" />
            )}
            <div className="flex-1 truncate">
              {background ? background : "Pick a color"}
            </div>
          </div>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-64">
        <div className="mt-0 flex flex-wrap gap-1">
          {solids.map((s) => (
            <div
              key={s}
              style={{ background: s }}
              className="size-6 cursor-pointer rounded-md active:scale-105"
              onClick={() => setBackground(s)}
            />
          ))}
        </div>
        <Input
          id="custom"
          value={background}
          className="col-span-2 mt-4 h-8"
          onChange={(e) => setBackground(e.currentTarget.value)}
        />
      </PopoverContent>
    </Popover>
  );
}
