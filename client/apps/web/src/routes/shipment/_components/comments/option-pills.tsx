import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import type { GenericSelectOption } from "@trenova/shared/types/fields";
import { CheckIcon } from "lucide-react";
import { useState, type ReactNode } from "react";

export function CommentOptionPill<T extends string>({
  label,
  icon,
  value,
  options,
  onChange,
}: {
  label: string;
  icon: ReactNode;
  value: T;
  options: ReadonlyArray<GenericSelectOption<T>>;
  onChange: (value: T) => void;
}) {
  const [popoverOpen, setPopoverOpen] = useState(false);
  const [tooltipOpen, setTooltipOpen] = useState(false);
  const selected = options.find((o) => o.value === value);

  return (
    <Popover open={popoverOpen} onOpenChange={setPopoverOpen}>
      <Tooltip open={popoverOpen ? false : tooltipOpen} onOpenChange={setTooltipOpen}>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={
                <Button variant="ghostInvert" size="xs" className="size-6">
                  {icon}
                </Button>
              }
            />
          }
        />
        <TooltipContent side="top">
          {label}: {selected?.label ?? value}
        </TooltipContent>
      </Tooltip>
      <PopoverContent align="start" className="max-h-60 w-44 gap-1 overflow-y-auto p-1 dark">
        <div className="px-2 py-1 text-2xs font-medium text-muted-foreground">{label}</div>
        {options.map((option) => (
          <button
            key={String(option.value)}
            type="button"
            className={cn(
              "flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs hover:bg-accent",
              option.value === value && "bg-accent",
            )}
            onClick={() => {
              onChange(option.value as T);
              setPopoverOpen(false);
            }}
          >
            {option.color && (
              <span
                className="size-2 shrink-0 rounded-full"
                style={{ backgroundColor: option.color }}
              />
            )}
            <span className="truncate">{option.label}</span>
            {option.value === value && <CheckIcon className="ml-auto size-3 shrink-0" />}
          </button>
        ))}
      </PopoverContent>
    </Popover>
  );
}
