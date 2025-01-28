import * as React from "react";
import {
  DayPicker,
  type DropdownNavProps,
  type DropdownProps,
  useDayPicker,
  useNavigation,
} from "react-day-picker";

import { cn } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import { ChevronLeftIcon, ChevronRightIcon } from "@radix-ui/react-icons";
import { format, setMonth } from "date-fns";
import { Select, SelectContent, SelectItem, SelectTrigger } from "./select";

export type CalendarProps = React.ComponentProps<typeof DayPicker>;

interface SelectItem {
  label: string;
  value: string;
}

const CalendarDropdown = ({ name, value }: DropdownProps) => {
  const { fromDate, fromMonth, fromYear, toMonth, toDate, toYear } =
    useDayPicker();
  const { goToMonth, currentMonth } = useNavigation();

  const getSelectItems = (): SelectItem[] => {
    if (name === "months") {
      return Array.from({ length: 12 }, (_, i) => ({
        label: format(setMonth(new Date(), i), "MMMM"),
        value: i.toString(),
      }));
    }

    const earliestYear =
      fromYear || fromMonth?.getFullYear() || fromDate?.getFullYear();
    const latestYear =
      toYear || toMonth?.getFullYear() || toDate?.getFullYear();

    if (!earliestYear || !latestYear) return [];

    return Array.from({ length: latestYear - earliestYear + 1 }, (_, i) => {
      const year = (earliestYear + i).toString();
      return { label: year, value: year };
    });
  };

  const handleValueChange = (newValue: string) => {
    const newDate = new Date(currentMonth);
    if (name === "months") {
      newDate.setMonth(parseInt(newValue));
    } else {
      newDate.setFullYear(parseInt(newValue));
    }
    goToMonth(newDate);
  };

  const getDisplayValue = () => {
    return name === "months"
      ? format(currentMonth, "MMMM")
      : currentMonth.getFullYear().toString();
  };

  const selectItems = getSelectItems();
  if (!selectItems.length) return null;

  return (
    <Select onValueChange={handleValueChange} value={value?.toString()}>
      <SelectTrigger
        className={cn(
          "h-7 px-2 gap-1 text-xs",
          name === "months" ? "w-full" : "w-fit",
        )}
      >
        {getDisplayValue()}
      </SelectTrigger>
      <SelectContent className="max-h-[min(26rem,var(--radix-select-content-available-height))]">
        {selectItems.map((item) => (
          <SelectItem key={item.value} value={item.value}>
            {item.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};

function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  ...props
}: CalendarProps) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn("w-fit rounded-lg border border-border p-2", className)}
      captionLayout="dropdown"
      fromDate={new Date(new Date().getFullYear() - 100, 0, 1)}
      toDate={new Date(new Date().getFullYear() + 20, 11, 31)}
      classNames={{
        months: "flex flex-col sm:flex-row space-y-4 sm:space-x-4 sm:space-y-0",
        month: "w-full",
        caption: "relative mb-1 flex h-9 items-center justify-center z-20 mx-0",
        caption_label: "text-sm font-medium hidden",
        nav: "space-x-1 flex items-center",
        nav_button: cn(
          buttonVariants({ variant: "outline" }),
          "h-7 w-7 bg-transparent p-0 opacity-50 hover:opacity-100",
        ),
        nav_button_previous: "absolute left-1",
        nav_button_next: "absolute right-1",
        table: "w-full border-collapse space-y-1",
        head_row: "flex",
        head_cell:
          "text-muted-foreground rounded-md w-8 font-normal text-[0.8rem]",
        row: "flex w-full mt-2",
        cell: cn(
          "relative p-0 text-center text-sm focus-within:relative focus-within:z-20 [&:has([aria-selected])]:bg-accent [&:has([aria-selected].day-outside)]:bg-accent/50 [&:has([aria-selected].day-range-end)]:rounded-r-md",
          props.mode === "range"
            ? "[&:has(>.day-range-end)]:rounded-r-md [&:has(>.day-range-start)]:rounded-l-md first:[&:has([aria-selected])]:rounded-l-md last:[&:has([aria-selected])]:rounded-r-md"
            : "[&:has([aria-selected])]:rounded-md",
        ),
        day: cn(
          buttonVariants({ variant: "ghost" }),
          "h-8 w-8 p-0 font-normal aria-selected:opacity-100",
        ),
        day_range_start: "day-range-start",
        day_range_end: "day-range-end",
        day_selected:
          "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
        day_today: "bg-muted text-accent-foreground",
        day_outside:
          "day-outside text-muted-foreground aria-selected:bg-accent/50 aria-selected:text-muted-foreground",
        day_disabled: "text-muted-foreground opacity-50",
        day_range_middle:
          "aria-selected:bg-accent aria-selected:text-accent-foreground",
        day_hidden: "invisible",
        caption_dropdowns: "flex w-full items-center gap-2",
        ...classNames,
      }}
      hideNavigation
      components={{
        IconLeft: ({ className, ...props }) => (
          <ChevronLeftIcon className={cn("h-4 w-4", className)} {...props} />
        ),
        IconRight: ({ className, ...props }) => (
          <ChevronRightIcon className={cn("h-4 w-4", className)} {...props} />
        ),
        Dropdown: CalendarDropdown,
        DropdownNav: (props: DropdownNavProps) => {
          return (
            <div className="flex w-full items-center gap-2">
              {props.children}
            </div>
          );
        },
      }}
      {...props}
    />
  );
}
Calendar.displayName = "Calendar";

export { Calendar };
