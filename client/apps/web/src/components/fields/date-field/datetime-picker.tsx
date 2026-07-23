import { Button } from "@/components/ui/button";
import { generateDateTime, generateDateTimeString } from "@/lib/date";
import { CalendarIcon } from "lucide-react";
import { DateSuggestionInput } from "./date-suggestion-input";
import { DateTimePickerPopover } from "./datetime-picker-popover";

const defaultSuggestions = ["t 0800", "t 1200", "t 1700", "t+1 0800", "t+1 1200", "t+1 1700"];

const formatDateTime = (date: Date) => generateDateTimeString(date);

export interface DateTimePickerProps
  extends Omit<React.ComponentProps<"input">, "value" | "defaultValue" | "onChange"> {
  dateTime: Date | undefined;
  setDateTime: (date: Date | undefined) => void;
  isInvalid?: boolean;
  clearable?: boolean;
}

export function DateTimePicker({
  dateTime,
  setDateTime,
  isInvalid,
  clearable,
  disabled,
  readOnly,
  ...props
}: DateTimePickerProps) {
  return (
    <DateSuggestionInput
      {...props}
      value={dateTime}
      onValueChange={setDateTime}
      formatValue={formatDateTime}
      parseInput={generateDateTime}
      defaultSuggestions={defaultSuggestions}
      isInvalid={isInvalid}
      clearable={clearable}
      disabled={disabled}
      readOnly={readOnly}
      picker={
        <DateTimePickerPopover dateTime={dateTime} setDateTime={setDateTime}>
          <Button
            type="button"
            size="icon"
            variant="ghost"
            disabled={disabled || readOnly}
            className="absolute top-1/2 right-2 size-5 -translate-y-1/2 text-muted-foreground [&>svg]:size-3"
          >
            <span className="sr-only">Open date time picker</span>
            <CalendarIcon className="size-4" />
          </Button>
        </DateTimePickerPopover>
      }
    />
  );
}
