import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { generateDateOnly } from "@trenova/shared/lib/chrono";
import { generateDateOnlyString, userWallClockNow } from "@trenova/shared/lib/date";
import { CalendarIcon } from "@trenova/shared/components/icons";
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
  timezone,
  ...props
}: DatePickerProps) {
  const t = useT();
  const parseInput = (text: string) => generateDateOnly(text, userWallClockNow(timezone));

  return (
    <DateSuggestionInput
      {...props}
      value={date}
      onValueChange={setDate}
      formatValue={generateDateOnlyString}
      parseInput={parseInput}
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
            className="text-muted-foreground absolute top-1/2 right-2 size-5 -translate-y-1/2 [&>svg]:size-3"
          >
            <span className="sr-only">{t("Open date picker")}</span>
            <CalendarIcon className="size-4" />
          </Button>
        </DatePickerPopover>
      }
    />
  );
}
