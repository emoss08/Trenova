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
  TableViewSource,
} from "@/types/table-configuration";
import { useQueryClient } from "@tanstack/react-query";
import {
  Building2Icon,
  CheckIcon,
  CopyIcon,
  GlobeIcon,
  LockIcon,
  MoreHorizontalIcon,
  SaveIcon,
  StarIcon,
  TrashIcon,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

export function DataTableConfigItem({
  config,
  isOwn,
  isActive,
  isViewDirty,
  canManageOrgDefaults,
  currentConfig,
  onApplyConfig,
  onViewPersisted,
  onViewDeleted,
  setOpen,
}: {
  config: TableConfiguration;
  isOwn: boolean;
  isActive?: boolean;
  isViewDirty?: boolean;
  canManageOrgDefaults?: boolean;
  currentConfig?: TableConfig;
  onApplyConfig: (config: TableConfig, source?: TableViewSource) => void;
  onViewPersisted?: (config: TableConfiguration) => void;
  onViewDeleted?: (id: string) => void;
  setOpen: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [dropdownOpen, setDropdownOpen] = useState(false);

  const refetchConfigurations = async () => {
    await queryClient.refetchQueries({
      queryKey: ["tableConfiguration"],
    });
  };

  const { mutateAsync: deleteConfig, isPending: isDeletingConfig } = useApiMutation({
    mutationFn: (id: string) => apiService.tableConfigurationService.delete(id),
    resourceName: "Table Configuration",
    onSuccess: () => {
      onViewDeleted?.(config.id);
      toast.success("View deleted", {
        description: `"${config.name}" has been deleted.`,
      });
    },
    onSettled: refetchConfigurations,
  });

  const { mutateAsync: setDefaultConfig, isPending: isSettingDefaultConfig } = useApiMutation({
    mutationFn: (id: string) => apiService.tableConfigurationService.setDefault(id),
    resourceName: "Table Configuration",
    onSuccess: (updated: TableConfiguration) => {
      onApplyConfig(updated.tableConfig, { id: updated.id, name: updated.name });
      toast.success("Default view updated", {
        description: `"${updated.name}" is now your default view and has been applied.`,
      });
    },
    onSettled: refetchConfigurations,
  });

  const { mutateAsync: setOrgDefaultConfig, isPending: isSettingOrgDefault } = useApiMutation({
    mutationFn: (enabled: boolean) =>
      apiService.tableConfigurationService.setOrgDefault(config.id, enabled),
    resourceName: "Table Configuration",
    onSuccess: (updated: TableConfiguration) => {
      toast.success(
        updated.isOrgDefault ? "Organization default set" : "Organization default removed",
        {
          description: updated.isOrgDefault
            ? `"${updated.name}" is now the default view for everyone in your organization.`
            : `"${updated.name}" is no longer the organization default.`,
        },
      );
    },
    onSettled: refetchConfigurations,
  });

  const { mutateAsync: updateConfig, isPending: isUpdatingConfig } = useApiMutation({
    mutationFn: (tableConfig: TableConfig) =>
      apiService.tableConfigurationService.update(config.id, {
        name: config.name,
        description: config.description,
        resource: config.resource,
        tableConfig,
        visibility: config.visibility,
        isDefault: config.isDefault,
      }),
    resourceName: "Table Configuration",
    onSuccess: (updated: TableConfiguration) => {
      onViewPersisted?.(updated);
      toast.success("View updated", {
        description: `"${updated.name}" now matches the current table state.`,
      });
    },
    onSettled: refetchConfigurations,
  });

  const { mutateAsync: duplicateConfig, isPending: isDuplicating } = useApiMutation({
    mutationFn: () => apiService.tableConfigurationService.duplicate(config),
    resourceName: "Table Configuration",
    onSuccess: (created: TableConfiguration) => {
      toast.success("View duplicated", {
        description: `"${created.name}" has been added to your views.`,
      });
    },
    onSettled: refetchConfigurations,
  });

  const handleApply = () => {
    onApplyConfig(config.tableConfig, { id: config.id, name: config.name });
    setOpen(false);
  };

  const withStopPropagation =
    (fn: () => unknown) => (e: React.MouseEvent<HTMLElement, MouseEvent>) => {
      e.stopPropagation();
      void fn();
    };

  return (
    <div
      className={cn(
        "group flex w-full items-center justify-between gap-2 rounded-md p-1 hover:bg-accent",
        isActive && "bg-accent/50",
      )}
    >
      <button
        type="button"
        onClick={handleApply}
        title={config.description || `Apply "${config.name}"`}
        className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 text-left"
      >
        {config.visibility === "Private" ? (
          <LockIcon className="size-3.5 shrink-0 text-muted-foreground" />
        ) : (
          <GlobeIcon className="size-3.5 shrink-0 text-muted-foreground" />
        )}
        <span className="flex min-w-0 flex-col">
          <span className="truncate">{config.name}</span>
          {!isOwn && config.user?.name && (
            <span className="truncate text-[10px] text-muted-foreground">
              by {config.user.name}
            </span>
          )}
        </span>
        {config.isDefault && isOwn && (
          <span className="flex shrink-0 items-center gap-0.5 rounded-sm bg-muted px-1 py-px text-[10px] font-medium text-muted-foreground">
            <StarIcon className="size-2.5" />
            Default
          </span>
        )}
        {config.isOrgDefault && (
          <span className="flex shrink-0 items-center gap-0.5 rounded-sm bg-muted px-1 py-px text-[10px] font-medium text-muted-foreground">
            <Building2Icon className="size-2.5" />
            Org default
          </span>
        )}
      </button>
      <div className="flex shrink-0 items-center gap-1">
        {isActive && (
          <span
            title={isViewDirty ? "Applied, with unsaved changes" : "Currently applied"}
            className="flex items-center"
          >
            {isViewDirty ? (
              <span className="size-1.5 rounded-full bg-amber-500" />
            ) : (
              <CheckIcon className="size-3.5 text-primary" />
            )}
          </span>
        )}
        <DropdownMenu open={dropdownOpen} onOpenChange={setDropdownOpen}>
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
                title={`${config.name} view options`}
                aria-label={`${config.name} view options`}
                aria-expanded={dropdownOpen}
              >
                <MoreHorizontalIcon className="size-4 text-muted-foreground" />
                <span className="sr-only">Open menu</span>
              </Button>
            }
          />
          <DropdownMenuContent align="end" side="inline-start" className="min-w-62">
            <DropdownMenuGroup>
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title="Apply"
                onClick={handleApply}
                description="Apply this view to the table"
                startContent={<CheckIcon className="size-4" />}
              />
              {isOwn && currentConfig && (
                <DropdownMenuItem
                  title="Save current state to view"
                  disabled={isUpdatingConfig}
                  onClick={withStopPropagation(() => updateConfig(currentConfig))}
                  description="Overwrite this view with the current filters, sorting, and columns"
                  startContent={<SaveIcon className="size-4" />}
                />
              )}
              <DropdownMenuItem
                title="Duplicate"
                disabled={isDuplicating}
                onClick={withStopPropagation(() => duplicateConfig(undefined))}
                description="Create your own private copy of this view"
                startContent={<CopyIcon className="size-4" />}
              />
              {isOwn && (
                <DropdownMenuItem
                  title="Set as default"
                  disabled={isSettingDefaultConfig || config.isDefault}
                  onClick={withStopPropagation(async () => {
                    await setDefaultConfig(config.id);
                    setOpen(false);
                  })}
                  description="Apply this view automatically when the table loads"
                  startContent={<StarIcon className="size-4" />}
                />
              )}
              {canManageOrgDefaults && config.visibility === "Public" && (
                <DropdownMenuItem
                  title={config.isOrgDefault ? "Remove org default" : "Set as org default"}
                  disabled={isSettingOrgDefault}
                  onClick={withStopPropagation(() => setOrgDefaultConfig(!config.isOrgDefault))}
                  description="The org default applies for everyone without a personal default"
                  startContent={<Building2Icon className="size-4" />}
                />
              )}
              {isOwn && (
                <DropdownMenuItem
                  title="Delete"
                  color="danger"
                  disabled={isDeletingConfig}
                  onClick={withStopPropagation(() => deleteConfig(config.id))}
                  description="Delete this view"
                  startContent={<TrashIcon className="size-4" />}
                />
              )}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
