"use no memo";
import { cn } from "@/lib/utils";
import type { SelectOption } from "@/types/fields";
import type { EquipmentStatus } from "@/types/helpers";
import {
  CheckCheckIcon,
  CheckIcon,
  ChevronDownIcon,
  ClockIcon,
  XIcon,
} from "lucide-react";
import type React from "react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { StatusBadge } from "./status-badge";
import { Badge, type BadgeVariant } from "./ui/badge";
import {
  Command,
  CommandGroup,
  CommandList,
  SelectCommandItem,
} from "./ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { Spinner } from "./ui/spinner";

const EQUIPMENT_STATUS_VARIANTS: Record<EquipmentStatus, BadgeVariant> = {
  Available: "active",
  AtMaintenance: "purple",
  OutOfService: "inactive",
  Sold: "warning",
};

const EQUIPMENT_STATUS_LABELS: Record<EquipmentStatus, string> = {
  Available: "Available",
  AtMaintenance: "At Maintenance",
  OutOfService: "Out of Service",
  Sold: "Sold",
};

const EQUIPMENT_STATUS_ICONS: Record<EquipmentStatus, React.ReactNode> = {
  Available: <CheckCheckIcon className="size-3" />,
  AtMaintenance: <ClockIcon className="size-3" />,
  OutOfService: <XIcon className="size-3" />,
  Sold: <CheckIcon className="size-3" />,
};

type EditableEquipmentStatusBadgeProps = {
  status: EquipmentStatus;
  options: SelectOption[];
  onStatusChange: (newStatus: EquipmentStatus) => Promise<void>;
  disabled?: boolean;
  className?: string;
};

export function EditableEquipmentStatusBadge({
  status,
  options,
  onStatusChange,
  disabled = false,
  className,
}: EditableEquipmentStatusBadgeProps) {
  const [open, setOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleStatusChange = useCallback(
    async (newStatus: EquipmentStatus) => {
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

  const variant = EQUIPMENT_STATUS_VARIANTS[status] || "outline";
  const icon = EQUIPMENT_STATUS_ICONS[status];
  const label = EQUIPMENT_STATUS_LABELS[status];

  if (disabled) {
    return <StatusBadge status={status} className={className} />;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Badge
            variant={variant}
            className={cn(
              "w-fit shrink-0 cursor-pointer text-left capitalize",
              className,
            )}
            render={<button type="button" disabled={isLoading} />}
          >
            {icon}
            {label}
            {isLoading ? (
              <Spinner className="size-3" />
            ) : (
              <ChevronDownIcon className="size-3" />
            )}
          </Badge>
        }
      />
      <PopoverContent className="w-32 p-0" align="start">
        <Command>
          <CommandList>
            <CommandGroup>
              {options.map((option) => (
                <SelectCommandItem
                  key={option.value}
                  value={option.value}
                  onSelect={(currentValue) =>
                    handleStatusChange(currentValue as EquipmentStatus)
                  }
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
