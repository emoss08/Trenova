import { cn } from "@/lib/utils";
import type { NumberFieldProps } from "@/types/fields";
import { ChevronDown, ChevronUp } from "lucide-react";
import * as React from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper } from "../fields/field-components";

export function NumberField<T extends FieldValues>({
  name,
  control,
  description,
  label,
  className,
  placeholder = "Enter Valid Number",
  sideText,
  rules,
  tabIndex,
  ...props
}: NumberFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
        >
          <div className="relative w-full">
            <input
              // Controlled value: keep input empty string when undefined/null
              value={field.value ?? ""}
              onChange={(e) => {
                const raw = e.target.value;
                // Allow the UI to hold an empty string without snapping back
                if (raw === "") {
                  field.onChange("");
                  return;
                }
                // Store numeric value during typing when possible
                const next = Number(raw);
                field.onChange(Number.isNaN(next) ? raw : next);
              }}
              onBlur={(e) => {
                // On blur, coerce to number or clear to undefined
                const raw = e.target.value;
                if (raw === "") {
                  field.onChange(undefined);
                } else {
                  const next = Number(raw);
                  if (!Number.isNaN(next)) field.onChange(next);
                }
                field.onBlur();
              }}
              name={field.name}
              ref={field.ref as React.Ref<HTMLInputElement>}
              tabIndex={tabIndex}
              placeholder={placeholder}
              id={inputId}
              disabled={props.disabled}
              readOnly={props.readOnly}
              aria-label={props["aria-label"] || label}
              aria-describedby={cn(
                description && descriptionId,
                fieldState.error && errorId,
                props["aria-describedby"],
              )}
              className={cn(
                "border-muted-foreground/20 bg-primary/5 flex h-7 w-full rounded-md border px-2 py-1 text-xs",
                "file:border-0 file:bg-transparent file:text-sm file:font-medium",
                "placeholder:text-muted-foreground",
                "disabled:cursor-not-allowed disabled:opacity-50",
                "read-only:cursor-default read-only:text-muted-foreground",
                "focus-visible:border-foreground focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-foreground/20",
                "transition-[border-color,box-shadow] duration-200 ease-in-out",
                props.readOnly &&
                  "cursor-not-allowed opacity-60 pointer-events-none",
                fieldState.invalid &&
                  "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
                sideText ? "pr-24" : "pr-12",
                className,
              )}
              {...(props.max !== undefined ? { max: props.max } : {})}
              {...(props.min !== undefined ? { min: props.min } : {})}
              {...(props.step !== undefined ? { step: props.step } : {})}
            />

            <div className="absolute right-[1px] top-[1px] bottom-[1px] flex items-center gap-1 pr-0">
              {sideText && (
                <div className="pointer-events-none mr-2 select-none text-xs text-muted-foreground">
                  {sideText}
                </div>
              )}
              <div className="flex h-full flex-col items-stretch rounded-r-md border-l border-muted-foreground/20 bg-transparent">
                <button
                  type="button"
                  aria-label="Increment"
                  className="inline-flex w-7 flex-1 items-center justify-center border-b border-muted-foreground/20 text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground disabled:opacity-50"
                  disabled={props.disabled || props.readOnly}
                  onClick={() => {
                    const step = props.step ? Number(props.step) : 1;
                    const current =
                      typeof field.value === "number"
                        ? field.value
                        : Number(field.value || 0);
                    let next = current + step;
                    if (props.max !== undefined) {
                      const max = Number(props.max as number);
                      if (!Number.isNaN(max)) next = Math.min(next, max);
                    }
                    field.onChange(next);
                  }}
                >
                  <ChevronUp className="h-3 w-3" />
                </button>
                <button
                  type="button"
                  aria-label="Decrement"
                  className="inline-flex w-7 flex-1 items-center justify-center text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground disabled:opacity-50"
                  disabled={props.disabled || props.readOnly}
                  onClick={() => {
                    const step = props.step ? Number(props.step) : 1;
                    const current =
                      typeof field.value === "number"
                        ? field.value
                        : Number(field.value || 0);
                    let next = current - step;
                    if (props.min !== undefined) {
                      const min = Number(props.min as number);
                      if (!Number.isNaN(min)) next = Math.max(next, min);
                    }
                    field.onChange(next);
                  }}
                >
                  <ChevronDown className="h-3 w-3" />
                </button>
              </div>
            </div>
          </div>
        </FieldWrapper>
      )}
    />
  );
}
