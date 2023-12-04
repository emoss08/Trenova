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

import DatePicker, { ReactDatePickerProps } from "react-datepicker";

import { ExtendedInputProps, Input } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { cn } from "@/lib/utils";
import React from "react";

const DatePickerInput = React.forwardRef<HTMLInputElement, ExtendedInputProps>(
  ({ value, onClick }, ref) => (
    <Input value={value} onClick={onClick} ref={ref} />
  ),
);

export function DatepickerField({
  ...props
}: ReactDatePickerProps & {
  label: string;
  withAsterisk?: boolean;
  description?: string;
  formError?: string;
}) {
  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium bg-background border-input",
            props.withAsterisk && "required",
          )}
          htmlFor={props.id}
        >
          {props.label}
        </Label>
      )}
      <div className="relative">
        <DatePicker
          wrapperClassName="flex"
          selected={props.selected}
          onChange={props.onChange}
          customInput={<DatePickerInput />}
          calendarClassName="bg-background border border-input rounded-md shadow-sm text-foreground"
          monthClassName={() =>
            "text-foreground bg-muted-foreground bg-background"
          }
          dayClassName={() => "text-foreground hover:bg-accent bg-background"}
          weekDayClassName={() => "text-foreground bg-background select-none"}
        />
        {props.description && !props.formError && (
          <p className="text-xs text-foreground/70">{props.description}</p>
        )}
      </div>
    </>
  );
}
