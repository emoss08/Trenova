import { cn } from "@/lib/utils";
import type { NumberFieldProps } from "@/types/fields";
import { ark } from "@ark-ui/react/factory";
import { NumberInput as ArkNumberInput } from "@ark-ui/react/number-input";
import { ChevronDownIcon, ChevronUpIcon } from "@radix-ui/react-icons";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper } from "../fields/field-components";

function NumberInputRoot({ className, ...props }: ArkNumberInput.RootProps) {
  return (
    <ArkNumberInput.Root
      data-slot="number-input-root"
      className={cn("flex flex-col", className)}
      {...props}
    />
  );
}

function NumberInputControl({
  className,
  ...props
}: ArkNumberInput.ControlProps) {
  return (
    <ArkNumberInput.Control
      data-slot="number-input-control"
      className={cn(
        "relative bg-muted rounded-md border border-muted-foreground/20 h-7 w-full px-2 py-1 text-xs",
        "placeholder:text-muted-foreground",
        "focus-visible:border-blue-600 focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-blue-600/20",
        "focus-visible:outline-none focus-visible:ring-1 [&[data-focus]]:ring-4 [&[data-focus]]:border-blue-600 [&[data-focus]]:ring-blue-600/20",
        "[&[data-invalid]]:border-red-500 [&[data-invalid]]:ring-4 [&[data-invalid]]:ring-red-500/20 [&[data-invalid]]:bg-red-500/20",
        "transition-[border-color,box-shadow] duration-200 ease-in-out",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

function NumberInputField({ className, ...props }: ArkNumberInput.InputProps) {
  return (
    <ArkNumberInput.Input
      data-slot="number-input-input"
      className={cn(
        "border-transparent border-none bg-transparent outline-none w-full",
        className,
      )}
      {...props}
    />
  );
}

function NumberInputIncrementTrigger({
  className,
  ...props
}: ArkNumberInput.IncrementTriggerProps) {
  return (
    <ArkNumberInput.IncrementTrigger
      data-slot="number-input-increment-trigger"
      className={cn(
        "absolute right-0 top-0 h-1/2 w-7 inline-flex items-center justify-center",
        "bg-background border-l border-border",
        className,
      )}
      {...props}
    />
  );
}

function NumberInputDecrementTrigger({
  className,
  ...props
}: ArkNumberInput.DecrementTriggerProps) {
  return (
    <ArkNumberInput.DecrementTrigger
      data-slot="number-input-decrement-trigger"
      className={cn(
        "absolute right-0 bottom-0 h-1/2 w-7 inline-flex items-center justify-center",
        "bg-background border-l border-t border-border",
        className,
      )}
      {...props}
    />
  );
}

export function NumberField<T extends FieldValues>({
  name,
  control,
  description,
  label,
  className,
  formattedOptions,
  inputMode = "numeric",
  placeholder = "Enter Valid Number",
  sideText,
  rules,
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
          <NumberInputRoot
            {...props}
            {...field}
            formatOptions={formattedOptions}
            className={cn(className)}
            inputMode={inputMode}
            allowMouseWheel
            defaultValue={field.value}
            invalid={fieldState.invalid}
            onValueChange={(details) => {
              field.onChange(details.value);
            }}
          >
            <NumberInputControl>
              <NumberInputField
                id={inputId}
                placeholder={placeholder}
                aria-label={label}
                value={field.value}
                aria-describedby={cn(
                  description && descriptionId,
                  fieldState.error && errorId,
                )}
              />
              {sideText && (
                <div className="pointer-events-none absolute inset-y-0 right-6 flex items-center pr-3 text-xs text-muted-foreground">
                  {sideText}
                </div>
              )}
              <NumberInputIncrementTrigger asChild>
                <ark.button className="hover:bg-muted focus:bg-muted cursor-pointer rounded-tr-md">
                  <ChevronUpIcon className="size-3" />
                </ark.button>
              </NumberInputIncrementTrigger>
              <NumberInputDecrementTrigger asChild>
                <ark.button className="hover:bg-muted focus:bg-muted cursor-pointer rounded-br-md">
                  <ChevronDownIcon className="size-3" />
                </ark.button>
              </NumberInputDecrementTrigger>
            </NumberInputControl>
          </NumberInputRoot>
        </FieldWrapper>
      )}
    />
  );
}
