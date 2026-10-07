"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import type { TableConfig, TableConfiguration, TableViewSource } from "@/types/table-configuration";
import { useQueryClient } from "@tanstack/react-query";
import {
  Building07Icon,
  CheckIcon,
  Copy01Icon,
  DotsHorizontalIcon,
  Globe02Icon,
  Lock01Icon,
  Save01Icon,
  Star01Icon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
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
  const t = useT();

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
      toast.success(t("View deleted"), {
        description: t('"{0}" has been deleted.', config.name),
      });
    },
    onSettled: refetchConfigurations,
  });

  const { mutateAsync: setDefaultConfig, isPending: isSettingDefaultConfig } = useApiMutation({
    mutationFn: (id: string) => apiService.tableConfigurationService.setDefault(id),
    resourceName: "Table Configuration",
    onSuccess: (updated: TableConfiguration) => {
      onApplyConfig(updated.tableConfig, { id: updated.id, name: updated.name });
      toast.success(t("Default view updated"), {
        description: t('"{0}" is now your default view and has been applied.', updated.name),
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
        updated.isOrgDefault ? t("Organization default set") : t("Organization default removed"),
        {
          description: updated.isOrgDefault
            ? t('"{0}" is now the default view for everyone in your organization.', updated.name)
            : t('"{0}" is no longer the organization default.', updated.name),
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
      toast.success(t("View updated"), {
        description: t('"{0}" now matches the current table state.', updated.name),
      });
    },
    onSettled: refetchConfigurations,
  });

  const { mutateAsync: duplicateConfig, isPending: isDuplicating } = useApiMutation({
    mutationFn: () => apiService.tableConfigurationService.duplicate(config),
    resourceName: "Table Configuration",
    onSuccess: (created: TableConfiguration) => {
      toast.success(t("View duplicated"), {
        description: t('"{0}" has been added to your views.', created.name),
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
        "group hover:bg-accent flex w-full items-center justify-between gap-2 rounded-md p-1",
        isActive && "bg-accent/50",
      )}
    >
      <button
        type="button"
        onClick={handleApply}
        title={config.description || t('Apply "{0}"', config.name)}
        className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 text-left"
      >
        {config.visibility === "Private" ? (
          <Lock01Icon className="text-muted-foreground size-3.5 shrink-0" />
        ) : (
          <Globe02Icon className="text-muted-foreground size-3.5 shrink-0" />
        )}
        <span className="flex min-w-0 flex-col">
          <span className="truncate">{config.name}</span>
          {!isOwn && config.user?.name && (
            <span className="text-muted-foreground truncate text-2xs">
              {t("by {0}", config.user.name)}
            </span>
          )}
        </span>
        {config.isDefault && isOwn && (
          <span className="bg-muted text-muted-foreground flex shrink-0 items-center gap-0.5 rounded-sm px-1 py-px text-2xs font-medium">
            <Star01Icon className="size-2.5" />
            {t("Default")}
          </span>
        )}
        {config.isOrgDefault && (
          <span className="bg-muted text-muted-foreground flex shrink-0 items-center gap-0.5 rounded-sm px-1 py-px text-2xs font-medium">
            <Building07Icon className="size-2.5" />
            {t("Org default")}
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
              <span className="size-1.5 rounded-full bg-warning" />
            ) : (
              <CheckIcon className="text-primary size-3.5" />
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
                title={t("{0} view options", config.name)}
                aria-label={t("{0} view options", config.name)}
                aria-expanded={dropdownOpen}
              >
                <DotsHorizontalIcon className="text-muted-foreground size-4" />
                <span className="sr-only">{t("Open menu")}</span>
              </Button>
            }
          />
          <DropdownMenuContent align="end" side="inline-start" className="min-w-62">
            <DropdownMenuGroup>
              <DropdownMenuLabel>{t("Actions")}</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title={t("Apply")}
                onClick={handleApply}
                description={t("Apply this view to the table")}
                startContent={<CheckIcon className="size-4" />}
              />
              {isOwn && currentConfig && (
                <DropdownMenuItem
                  title={t("Save current state to view")}
                  disabled={isUpdatingConfig}
                  onClick={withStopPropagation(() => updateConfig(currentConfig))}
                  description={t(
                    "Overwrite this view with the current filters, sorting, and columns",
                  )}
                  startContent={<Save01Icon className="size-4" />}
                />
              )}
              <DropdownMenuItem
                title={t("Duplicate")}
                disabled={isDuplicating}
                onClick={withStopPropagation(() => duplicateConfig(undefined))}
                description={t("Create your own private copy of this view")}
                startContent={<Copy01Icon className="size-4" />}
              />
              {isOwn && (
                <DropdownMenuItem
                  title={t("Set as default")}
                  disabled={isSettingDefaultConfig || config.isDefault}
                  onClick={withStopPropagation(async () => {
                    await setDefaultConfig(config.id);
                    setOpen(false);
                  })}
                  description={t("Apply this view automatically when the table loads")}
                  startContent={<Star01Icon className="size-4" />}
                />
              )}
              {canManageOrgDefaults && config.visibility === "Public" && (
                <DropdownMenuItem
                  title={config.isOrgDefault ? "Remove org default" : "Set as org default"}
                  disabled={isSettingOrgDefault}
                  onClick={withStopPropagation(() => setOrgDefaultConfig(!config.isOrgDefault))}
                  description={t("The org default applies for everyone without a personal default")}
                  startContent={<Building07Icon className="size-4" />}
                />
              )}
              {isOwn && (
                <DropdownMenuItem
                  title={t("Delete")}
                  color="danger"
                  disabled={isDeletingConfig}
                  onClick={withStopPropagation(() => deleteConfig(config.id))}
                  description={t("Delete this view")}
                  startContent={<Trash01Icon className="size-4" />}
                />
              )}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
