import { Calendar } from "@/components/ui/calendar";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Tooltip, TooltipTrigger } from "@/components/ui/tooltip";
import { useMediaQuery } from "@/hooks/use-media-query";
import { generateDateOnlyString } from "@/lib/date";
import { useCallback, useState } from "react";

interface DatePickerPopoverProps {
  children: React.ReactElement;
  onOpen: () => void;
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
  setInputValue: (value: string) => void;
}

export function DatePickerPopover({
  children,
  onOpen,
  date,
  setDate,
  setInputValue,
}: DatePickerPopoverProps) {
  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const isDesktop = useMediaQuery("(min-width: 640px)");

  const onSelect = useCallback(
    (date: Date | undefined) => {
      if (date) {
        setDate(date);
        setInputValue(generateDateOnlyString(date));
        setIsPopoverOpen(false);
      }
    },
    [setDate, setInputValue],
  );

  if (!isDesktop) {
    return (
      <Drawer
        open={isDrawerOpen}
        onOpenChange={(value) => {
          onOpen();
          setIsDrawerOpen(value);
        }}
        shouldScaleBackground
      >
        <DrawerTrigger asChild>{children}</DrawerTrigger>
        <DrawerContent>
          <DrawerHeader className="sr-only text-left">
            <DrawerTitle>Date Time Picker</DrawerTitle>
            <DrawerDescription>Select date and time</DrawerDescription>
          </DrawerHeader>
          <div className="flex flex-col py-5">
            <Calendar
              mode="single"
              selected={date}
              onSelect={onSelect}
              className="self-center"
            />
          </div>
        </DrawerContent>
      </Drawer>
    );
  }

  return (
    <Popover
      open={isPopoverOpen}
      onOpenChange={(value) => {
        onOpen();
        setIsPopoverOpen(value);
      }}
    >
      <Tooltip>
        <TooltipTrigger render={<PopoverTrigger render={children} />} />
      </Tooltip>
      <PopoverContent align="center" side="bottom" className="w-auto p-0">
        <Calendar mode="single" selected={date} onSelect={onSelect} />
      </PopoverContent>
    </Popover>
  );
}
