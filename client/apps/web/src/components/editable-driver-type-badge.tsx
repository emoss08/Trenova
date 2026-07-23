"use no memo";
import { cn } from "@/lib/utils";
import type { SelectOption } from "@/types/fields";
import type { DriverType } from "@/types/worker";
import {
  ChevronDownIcon,
  MapPinIcon,
  RouteIcon,
  TruckIcon,
  UsersIcon,
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

const DRIVER_TYPE_VARIANTS: Record<DriverType, BadgeVariant> = {
  Local: "info",
  Regional: "active",
  OTR: "warning",
  Team: "purple",
};

const DRIVER_TYPE_LABELS: Record<DriverType, string> = {
  Local: "Local",
  Regional: "Regional",
  OTR: "OTR",
  Team: "Team",
};

const DRIVER_TYPE_ICONS: Record<DriverType, React.ReactNode> = {
  Local: <MapPinIcon className="size-3" />,
  Regional: <RouteIcon className="size-3" />,
  OTR: <TruckIcon className="size-3" />,
  Team: <UsersIcon className="size-3" />,
};

type EditableDriverTypeBadgeProps = {
  driverType: DriverType;
  options: SelectOption[];
  onDriverTypeChange: (newType: DriverType) => Promise<void>;
  disabled?: boolean;
  className?: string;
};

export function EditableDriverTypeBadge({
  driverType,
  options,
  onDriverTypeChange,
  disabled = false,
  className,
}: EditableDriverTypeBadgeProps) {
  const [open, setOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleTypeChange = useCallback(
    async (newType: DriverType) => {
      if (newType === driverType) {
        setOpen(false);
        return;
      }

      setOpen(false);
      setIsLoading(true);

      await onDriverTypeChange(newType)
        .catch(() => {
          toast.error("Failed to update driver type");
        })
        .finally(() => {
          setIsLoading(false);
        });
    },
    [driverType, onDriverTypeChange],
  );

  const variant = DRIVER_TYPE_VARIANTS[driverType] || "outline";
  const icon = DRIVER_TYPE_ICONS[driverType];
  const label = DRIVER_TYPE_LABELS[driverType];

  if (disabled) {
    return <StatusBadge status={driverType} className={className} />;
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
                    handleTypeChange(currentValue as DriverType)
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
