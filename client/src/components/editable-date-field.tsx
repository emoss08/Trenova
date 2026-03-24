"use no memo";
import { generateDateOnlyString, toDate, toUnixTimeStamp } from "@/lib/date";
import { cn } from "@/lib/utils";
import { ChevronDownIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { Badge } from "./ui/badge";
import { Calendar } from "./ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { Spinner } from "./ui/spinner";

type EditableDateFieldProps = {
  date?: number | null;
  onDateChange: (newDate: number) => Promise<void>;
  disabled?: boolean;
  className?: string;
};

export function EditableDateField({
  date,
  onDateChange,
  disabled = false,
  className,
}: EditableDateFieldProps) {
  const [open, setOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleDateChange = useCallback(
    async (newDate: number) => {
      setOpen(false);
      setIsLoading(true);
      await onDateChange(newDate);
      setIsLoading(false);
    },
    [onDateChange],
  );

  const formattedDate = toDate(date ?? undefined);

  const displayDate = formattedDate
    ? generateDateOnlyString(formattedDate)
    : "No date";

  const variant = date ? "active" : "inactive";

  const handleDateSelect = useCallback(
    (newDate: Date | undefined) => {
      void handleDateChange(toUnixTimeStamp(newDate) ?? 0);
    },
    [handleDateChange],
  );

  if (disabled) {
    return (
      <Badge variant="outline" className={className}>
        {displayDate}
      </Badge>
    );
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Badge
            variant={variant}
            className={cn("cursor-pointer capitalize", className)}
            render={<button type="button" disabled={isLoading} />}
          >
            {displayDate}
            {isLoading ? (
              <Spinner className="size-3" />
            ) : (
              <ChevronDownIcon className="size-3" />
            )}
          </Badge>
        }
      />
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          selected={formattedDate}
          onSelect={handleDateSelect}
        />
      </PopoverContent>
    </Popover>
  );
}
