"use no memo";
import { cn } from "@/lib/utils";
import type { SelectOption } from "@/types/fields";
import type { WorkerType } from "@/types/worker";
import { BriefcaseIcon, ChevronDownIcon, UserIcon } from "lucide-react";
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

const WORKER_TYPE_VARIANTS: Record<WorkerType, BadgeVariant> = {
  Employee: "active",
  Contractor: "purple",
};

const WORKER_TYPE_LABELS: Record<WorkerType, string> = {
  Employee: "Employee",
  Contractor: "Contractor",
};

const WORKER_TYPE_ICONS: Record<WorkerType, React.ReactNode> = {
  Employee: <UserIcon className="size-3" />,
  Contractor: <BriefcaseIcon className="size-3" />,
};

type EditableWorkerTypeBadgeProps = {
  workerType: WorkerType;
  options: SelectOption[];
  onWorkerTypeChange: (newType: WorkerType) => Promise<void>;
  disabled?: boolean;
  className?: string;
};

export function EditableWorkerTypeBadge({
  workerType,
  options,
  onWorkerTypeChange,
  disabled = false,
  className,
}: EditableWorkerTypeBadgeProps) {
  const [open, setOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleTypeChange = useCallback(
    async (newType: WorkerType) => {
      if (newType === workerType) {
        setOpen(false);
        return;
      }

      setOpen(false);
      setIsLoading(true);

      await onWorkerTypeChange(newType)
        .catch(() => {
          toast.error("Failed to update worker type");
        })
        .finally(() => {
          setIsLoading(false);
        });
    },
    [workerType, onWorkerTypeChange],
  );

  const variant = WORKER_TYPE_VARIANTS[workerType] || "outline";
  const icon = WORKER_TYPE_ICONS[workerType];
  const label = WORKER_TYPE_LABELS[workerType];

  if (disabled) {
    return <StatusBadge status={workerType} className={className} />;
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
                    handleTypeChange(currentValue as WorkerType)
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
