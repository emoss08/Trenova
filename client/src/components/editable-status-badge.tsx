"use no memo";
import { cn } from "@/lib/utils";
import type { SelectOption } from "@/types/fields";
import { CheckCheckIcon, CheckIcon, ChevronDownIcon, ClockIcon, XIcon } from "lucide-react";
import type React from "react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { StatusBadge } from "./status-badge";
import { Badge, type BadgeVariant } from "./ui/badge";
import { Command, CommandGroup, CommandList, SelectCommandItem } from "./ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { Spinner } from "./ui/spinner";

const STATUS_VARIANTS: Record<string, BadgeVariant> = {
  active: "active",
  inactive: "inactive",
  draft: "secondary",
  pending: "outline",
  completed: "default",
  cancelled: "inactive",
  processing: "secondary",
  inreview: "warning",
  paused: "warning",
  outstanding: "warning",
  posted: "purple",
  paid: "active",
  voided: "inactive",
  open: "info",
  closed: "secondary",
};

const STATUS_ICONS: Record<string, React.ReactNode> = {
  active: <CheckCheckIcon className="size-3" />,
  inactive: <XIcon className="size-3" />,
  draft: <ClockIcon className="size-3" />,
  pending: <ClockIcon className="size-3" />,
  completed: <CheckIcon className="size-3" />,
  cancelled: <XIcon className="size-3" />,
  processing: <ClockIcon className="size-3" />,
  inreview: <ClockIcon className="size-3" />,
};

type EditableStatusBadgeProps<T extends string> = {
  status: T;
  options: SelectOption[];
  onStatusChange: (newStatus: T) => Promise<void>;
  variants?: Partial<Record<T, BadgeVariant>>;
  disabled?: boolean;
  disabledReason?: string;
  className?: string;
};

export function EditableStatusBadge<T extends string>({
  status,
  options,
  onStatusChange,
  variants,
  disabled = false,
  disabledReason,
  className,
}: EditableStatusBadgeProps<T>) {
  const [open, setOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleStatusChange = useCallback(
    async (newStatus: T) => {
      if (newStatus === status) {
        setOpen(false);
        return;
      }

      setOpen(false);
      setIsLoading(true);

      await onStatusChange(newStatus)
        .catch(() => {
          toast.error("Failed to update status");
        })
        .finally(() => {
          setIsLoading(false);
        });
    },
    [status, onStatusChange],
  );

  const normalizedStatus = status.toLowerCase();
  const variant = variants?.[status] ?? STATUS_VARIANTS[normalizedStatus] ?? "outline";
  const icon = STATUS_ICONS[normalizedStatus];
  const label = options.find((option) => option.value === status)?.label ?? status;

  if (disabled) {
    if (variants?.[status]) {
      return (
        <Badge variant={variant} className={cn("max-h-5", className)} title={disabledReason}>
          {label}
        </Badge>
      );
    }
    return <StatusBadge status={status} className={className} />;
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
            {icon}
            {label}
            {isLoading ? <Spinner className="size-3" /> : <ChevronDownIcon className="size-3" />}
          </Badge>
        }
      />
      <PopoverContent className="w-30 p-0" align="start">
        <Command>
          <CommandList>
            <CommandGroup>
              {options.map((option) => (
                <SelectCommandItem
                  key={option.value}
                  value={option.value}
                  onSelect={(currentValue) => handleStatusChange(currentValue as T)}
                  className="text-xs"
                  label={option.label}
                  color={option.color}
                  description={option.description}
                  icon={option.icon}
                  disabled={option.disabled}
                />
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
