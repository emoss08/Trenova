"use client";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { toUnixTimeStamp } from "@/lib/date";
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
      setDate(toUnixTimeStamp(newDate));
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
            className="w-[280px] justify-start text-left font-normal data-[empty=true]:text-muted-foreground"
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
