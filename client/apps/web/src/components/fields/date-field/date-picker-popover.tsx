import { useT } from "@trenova/shared/i18n/use-t";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@trenova/shared/components/ui/drawer";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useMediaQuery } from "@/hooks/use-media-query";
import { useCallback, useState } from "react";

interface DatePickerPopoverProps {
  children: React.ReactElement;
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
}

export function DatePickerPopover({ children, date, setDate }: DatePickerPopoverProps) {
  const t = useT();

  const [isOpen, setIsOpen] = useState(false);

  const isDesktop = useMediaQuery("(min-width: 640px)");

  const onSelect = useCallback(
    (selected: Date | undefined) => {
      if (!selected) return;
      setDate(selected);
      setIsOpen(false);
    },
    [setDate],
  );

  if (!isDesktop) {
    return (
      <Drawer open={isOpen} onOpenChange={setIsOpen} shouldScaleBackground>
        <DrawerTrigger asChild>{children}</DrawerTrigger>
        <DrawerContent>
          <DrawerHeader className="sr-only text-left">
            <DrawerTitle>{t("Date Picker")}</DrawerTitle>
            <DrawerDescription>{t("Select date")}</DrawerDescription>
          </DrawerHeader>
          <div className="flex flex-col py-5">
            <Calendar
              mode="single"
              selected={date}
              defaultMonth={date}
              onSelect={onSelect}
              className="self-center"
            />
          </div>
        </DrawerContent>
      </Drawer>
    );
  }

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger render={children} />
      <PopoverContent align="center" side="bottom" className="w-auto p-0">
        <Calendar mode="single" selected={date} defaultMonth={date} onSelect={onSelect} />
      </PopoverContent>
    </Popover>
  );
}
