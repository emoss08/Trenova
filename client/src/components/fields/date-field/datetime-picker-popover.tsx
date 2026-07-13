import { useCallback, useState } from "react";

import { Calendar } from "@/components/ui/calendar";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useMediaQuery } from "@/hooks/use-media-query";
import { TimePicker } from "../time-picker/time-picker";

interface DateTimePickerPopoverProps {
  children: React.ReactElement;
  dateTime: Date | undefined;
  setDateTime: (date: Date | undefined) => void;
}

export function DateTimePickerPopover({
  children,
  dateTime,
  setDateTime,
}: DateTimePickerPopoverProps) {
  const [isOpen, setIsOpen] = useState(false);

  const isDesktop = useMediaQuery("(min-width: 640px)");

  const onSelectDay = useCallback(
    (day: Date | undefined) => {
      if (!day) return;
      const next = new Date(day);
      if (dateTime) {
        next.setHours(
          dateTime.getHours(),
          dateTime.getMinutes(),
          dateTime.getSeconds(),
          dateTime.getMilliseconds(),
        );
      }
      setDateTime(next);
    },
    [dateTime, setDateTime],
  );

  if (!isDesktop) {
    return (
      <Drawer open={isOpen} onOpenChange={setIsOpen} shouldScaleBackground>
        <DrawerTrigger asChild>{children}</DrawerTrigger>
        <DrawerContent>
          <DrawerHeader className="sr-only text-left">
            <DrawerTitle>Date Time Picker</DrawerTitle>
            <DrawerDescription>Select date and time</DrawerDescription>
          </DrawerHeader>
          <div className="flex flex-col py-5">
            <Calendar
              mode="single"
              selected={dateTime}
              defaultMonth={dateTime}
              onSelect={onSelectDay}
              className="self-center"
            />
            <div className="border-t border-border p-3">
              <TimePicker date={dateTime} setDate={setDateTime} />
            </div>
          </div>
        </DrawerContent>
      </Drawer>
    );
  }

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger render={children} />
      <PopoverContent align="center" side="right" className="w-auto p-0">
        <Calendar mode="single" selected={dateTime} defaultMonth={dateTime} onSelect={onSelectDay} />
        <div className="border-t border-border p-3">
          <TimePicker date={dateTime} setDate={setDateTime} />
        </div>
      </PopoverContent>
    </Popover>
  );
}
