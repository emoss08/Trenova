import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { toDate, toUnixTimeStamp } from "@/lib/date";
import { cn } from "@/lib/utils";
import {
  AutoCompleteDateFieldProps,
  DateFieldProps,
  DoubleClickEditDateProps,
} from "@/types/fields";
import { CalendarIcon } from "@radix-ui/react-icons";
import { format } from "date-fns";
import { useCallback, useState } from "react";
import { Controller, FieldValues } from "react-hook-form";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";
import { AutoCompleteDatePicker } from "./date-field/date-picker";
import { FieldWrapper } from "./field-components";

const styles = {
  base: "w-full h-8 text-sm justify-start text-left font-normal border border-muted-foreground/20 bg-muted rounded-md",
  invalid:
    "border-red-500 bg-red-500/20 text-red-500 hover:text-red-500 hover:bg-red-500/20 data-[state=open]:border-red-600 data-[state=open]:outline-none data-[state=open]:ring-4 data-[state=open]:ring-red-400/20",
  open: "text-sm data-[state=open]:border-blue-600 data-[state=open]:outline-none data-[state=open]:ring-4 data-[state=open]:ring-blue-600/20",
  focusVisible:
    "focus-visible:border-blue-600 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-blue-600/20",
  hover:
    "transition-[border-color,box-shadow] duration-200 ease-in-out hover:bg-none",
  disabled: "text-muted-foreground hover:text-muted-foreground",
};

export function DateField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  placeholder,
}: DateFieldProps<T>) {
  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const dateValue = toDate(field.value);

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <Popover>
              <PopoverTrigger asChild>
                <Button
                  variant="outline"
                  className={cn(
                    styles.base,
                    styles.focusVisible,
                    styles.hover,
                    !dateValue && styles.disabled,
                    fieldState.invalid && styles.invalid,
                  )}
                >
                  <CalendarIcon className="mr-0.5" />
                  {dateValue ? (
                    format(dateValue, "PPP")
                  ) : (
                    <span>{placeholder || "Pick a date"}</span>
                  )}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-[--radix-popover-trigger-width] p-0">
                <Calendar
                  mode="single"
                  selected={dateValue}
                  onSelect={(date) => {
                    field.onChange(toUnixTimeStamp(date));
                  }}
                  initialFocus
                />
              </PopoverContent>
            </Popover>
          </FieldWrapper>
        );
      }}
    />
  );
}

export function DoubleClickEditDate<T extends FieldValues>({
  name,
  control,
  rules,
  placeholder,
}: DoubleClickEditDateProps<T>) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const dateValue = toDate(field.value);
        const error = fieldState.error?.message;

        const handleSelect = useCallback(
          (date: Date | undefined) => {
            field.onChange(toUnixTimeStamp(date));
            setIsOpen(false);
          },
          [field],
        );

        return (
          <Popover open={isOpen} onOpenChange={setIsOpen}>
            <PopoverTrigger>
              <TooltipProvider>
                <Tooltip delayDuration={0}>
                  <TooltipTrigger asChild>
                    <span className="flex flex-col text-left text-xs">
                      <div
                        className={cn(
                          "cursor-text",
                          fieldState.invalid && "text-red-500",
                        )}
                      >
                        {dateValue ? (
                          format(dateValue, "PPP")
                        ) : (
                          <span>{placeholder || "Pick a date"}</span>
                        )}
                      </div>
                      {isOpen ? (
                        <span
                          onClick={() => setIsOpen(false)}
                          className="cursor-pointer select-none text-xs text-muted-foreground"
                        >
                          Cancel
                        </span>
                      ) : (
                        <span className="cursor-pointer select-none text-xs text-muted-foreground">
                          Click to edit
                        </span>
                      )}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent
                    className={cn(
                      error &&
                        "bg-red-50 text-red-500 dark:bg-red-900 dark:text-red-50",
                    )}
                  >
                    <p>{error ? error : placeholder}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0">
              <Calendar
                mode="single"
                selected={dateValue}
                onSelect={handleSelect}
                initialFocus
              />
            </PopoverContent>
          </Popover>
        );
      }}
    />
  );
}

export function AutoCompleteDateField<T extends FieldValues>({
  name,
  control,
  rules,
  className,
  label,
  description,
  placeholder,
  ...props
}: AutoCompleteDateFieldProps<T>) {
  const inputId = `input-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const dateValue = field.value ? toDate(field.value) : undefined;

        const handleChange = useCallback(
          (date: Date | undefined) => {
            console.info("Date before format", date);
            const formattedDate = toUnixTimeStamp(date);
            console.log("formattedDate", formattedDate);
            field.onChange(formattedDate);
          },
          [field],
        );

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <AutoCompleteDatePicker
              {...props}
              {...field}
              name={name}
              id={inputId}
              aria-label={label}
              date={dateValue}
              placeholder={placeholder}
              setDate={handleChange}
              onBlur={field.onBlur}
              className={className}
              isInvalid={fieldState.invalid}
              autoComplete="off"
              aria-describedby={cn(
                description && descriptionId,
                fieldState.error && errorId,
              )}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}
