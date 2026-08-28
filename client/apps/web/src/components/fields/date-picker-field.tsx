"use client";

import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { fromUserWallClock } from "@trenova/shared/lib/date";
import { format } from "date-fns";
import { Calendar as CalendarIcon } from "lucide-react";
import { useCallback } from "react";

type DatePickerFieldProps = {
  date?: Date;
  setDate: (newDate: number | undefined) => void;
};

export function DatePickerField({ date, setDate }: DatePickerFieldProps) {
  const handleDateSelect = useCallback(
    (newDate: Date | undefined) => {
      setDate(fromUserWallClock(newDate));
    },
    [setDate],
  );
  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            data-empty={!date}
            className="data-[empty=true]:text-muted-foreground w-[280px] justify-start text-left font-normal"
          >
            <CalendarIcon />
            {date ? format(date, "PPP") : <span>Pick a date</span>}
          </Button>
        }
      />
      <PopoverContent className="w-auto p-0">
        <Calendar mode="single" selected={date} onSelect={handleDateSelect} />
      </PopoverContent>
    </Popover>
  );
}
