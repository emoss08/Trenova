"use no memo";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  TableConfig,
  TableConfiguration,
} from "@/types/table-configuration";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  GlobeIcon,
  LockIcon,
  MoreHorizontalIcon,
  PencilIcon,
  TrashIcon,
} from "lucide-react";
import React, { useCallback, useState } from "react";
import { toast } from "sonner";

export function DataTableConfigItem({
  config,
  onApplyConfig,
  setOpen,
}: {
  config: TableConfiguration;
  onApplyConfig: (config: TableConfig) => void;
  setOpen: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const { mutateAsync: deleteConfig, isPending: isDeletingConfig } =
    useApiMutation({
      mutationFn: (id: string) => apiService.tableConfigurationService.delete(id),
      resourceName: "Table Configuration",
      onSuccess: () => {
        toast.success("Table Configuration deleted", {
          description: "The table view has been deleted.",
        });
      },
      onSettled: async () => {
        await queryClient.refetchQueries({
          queryKey: ["tableConfiguration"],
        });
      },
    });

  const { mutateAsync: setDefaultConfig, isPending: isSettingDefaultConfig } =
    useApiMutation({
      mutationFn: (id: string) => apiService.tableConfigurationService.setDefault(id),
      resourceName: "Table Configuration",
      onSuccess: () => {
        toast.success("Table Configuration set as default", {
          description: "The table view has been set as the default.",
        });
      },
      onSettled: async () => {
        await queryClient.refetchQueries({
          queryKey: ["tableConfiguration"],
        });
      },
    });

  const handleApply = (config: TableConfiguration) => {
    onApplyConfig(config.tableConfig);
    setOpen(false);
  };

  const handleSetDefault = useCallback(
    async (
      e: React.MouseEvent<HTMLElement, MouseEvent>,
      id: TableConfiguration["id"],
    ) => {
      e.stopPropagation();
      await setDefaultConfig(id);
    },
    [setDefaultConfig],
  );

  const handleDelete = useCallback(
    async (
      e: React.MouseEvent<HTMLElement, MouseEvent>,
      id: TableConfiguration["id"],
    ) => {
      e.stopPropagation();
      await deleteConfig(id);
    },
    [deleteConfig],
  );

  return (
    <div
      key={config.id}
      className="group flex w-full items-center justify-between gap-2 rounded-md p-1 hover:bg-accent"
    >
      <div className="flex min-w-0 items-center gap-2">
        {config.visibility === "Private" ? (
          <LockIcon className="size-3.5 shrink-0 text-muted-foreground" />
        ) : (
          <GlobeIcon className="size-3.5 shrink-0 text-muted-foreground" />
        )}
        <span className="truncate">{config.name}</span>
        {config.isDefault && (
          <CheckIcon className="size-3.5 shrink-0 text-primary" />
        )}
      </div>
      <div className="flex shrink-0 items-center gap-0.5">
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon-xs"
                className={cn(
                  "cursor-pointer opacity-0 transition-opacity group-hover:opacity-100",
                  dropdownOpen && "opacity-100",
                )}
                type="button"
                title={`${config.name} configuration options`}
                aria-label={`${config.name} configuration options`}
                aria-expanded={dropdownOpen}
                onClick={() => setDropdownOpen(!dropdownOpen)}
              >
                <MoreHorizontalIcon className="size-4 text-muted-foreground" />
                <span className="sr-only">Open menu</span>
              </Button>
            }
          />
          <DropdownMenuContent
            align="end"
            side="inline-start"
            className="min-w-62"
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title="Apply"
                onClick={() => handleApply(config)}
                description="Apply this configuration to the current view"
                startContent={<PencilIcon className="size-4" />}
              />
              <DropdownMenuItem
                title="Set as default"
                disabled={isSettingDefaultConfig}
                onClick={(e) => handleSetDefault(e, config.id)}
                description="Set this configuration as the default for this resource"
                startContent={<CheckIcon className="size-4" />}
              />
              <DropdownMenuItem
                title="Delete"
                color="danger"
                disabled={isDeletingConfig}
                onClick={(e) => handleDelete(e, config.id)}
                description="Delete this configuration"
                startContent={<TrashIcon className="size-4" />}
              />
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
