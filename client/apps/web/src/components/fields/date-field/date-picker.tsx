import { Button } from "@/components/ui/button";
import { generateDateOnly, generateDateOnlyString } from "@/lib/date";
import { CalendarIcon } from "lucide-react";
import type { DatePickerProps } from "./date-field";
import { DatePickerPopover } from "./date-picker-popover";
import { DateSuggestionInput, type Suggestion } from "./date-suggestion-input";

export type { Suggestion };

const defaultSuggestions = ["t", "t+1", "t+2", "t+3", "t+5", "t+7"];

export function AutoCompleteDatePicker({
  date,
  setDate,
  isInvalid,
  clearable,
  disabled,
  readOnly,
  ...props
}: DatePickerProps) {
  return (
    <DateSuggestionInput
      {...props}
      value={date}
      onValueChange={setDate}
      formatValue={generateDateOnlyString}
      parseInput={generateDateOnly}
      defaultSuggestions={defaultSuggestions}
      isInvalid={isInvalid}
      clearable={clearable}
      disabled={disabled}
      readOnly={readOnly}
      picker={
        <DatePickerPopover date={date} setDate={setDate}>
          <Button
            type="button"
            size="icon"
            variant="ghost"
            disabled={disabled || readOnly}
            className="absolute top-1/2 right-2 size-5 -translate-y-1/2 text-muted-foreground [&>svg]:size-3"
          >
            <span className="sr-only">Open date picker</span>
            <CalendarIcon className="size-4" />
          </Button>
        </DatePickerPopover>
      }
    />
  );
}
