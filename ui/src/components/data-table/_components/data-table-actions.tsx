"use no memo";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { usePermissions } from "@/hooks/use-permission";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import type { Resource } from "@/types/audit-entry";
import type { ExtraAction } from "@/types/data-table";
import type { LiveModeTableConfig } from "@/types/live-mode";
import { Action } from "@/types/roles-permissions";
import { faCirclePlay, faCircleStop } from "@fortawesome/pro-solid-svg-icons";
import React from "react";
import { DataTableExport } from "./_actions/data-table-export";
import {
  DataTableCreateButton,
  DataTableViewOptions,
} from "./_view-options/data-table-view-options";

export default function DataTableActions({
  name,
  exportModelName,
  extraActions,
  handleCreateClick,
  resource,
  liveModeConfig,
  liveModeEnabled,
  onLiveModeToggle,
  filterState,
}: {
  name: string;
  resource: Resource;
  exportModelName: string;
  handleCreateClick: () => void;
  liveModeEnabled: boolean;
  onLiveModeToggle: (enabled: boolean) => void;
  filterState: FilterStateSchema;
  extraActions?: ExtraAction[];
  liveModeConfig?: LiveModeTableConfig;
}) {
  const { can } = usePermissions();

  return (
    <DataTableActionsInner>
      <DataTableViewOptions resource={resource} />
      <DataTableExport
        resource={resource}
        filterState={filterState}
        resourceName={name}
      />
      {liveModeConfig && (
        <Button
          variant={liveModeEnabled ? "red" : "outline"}
          onClick={() => onLiveModeToggle(!liveModeEnabled)}
        >
          {liveModeEnabled ? (
            <Icon icon={faCircleStop} />
          ) : (
            <Icon icon={faCirclePlay} />
          )}
          Live Mode
        </Button>
      )}
      <div className="h-6 w-px bg-border" />
      <DataTableCreateButton
        name={name}
        exportModelName={exportModelName}
        extraActions={extraActions}
        isDisabled={!can(resource, Action.Create)}
        onCreateClick={handleCreateClick}
      />
    </DataTableActionsInner>
  );
}

export function DataTableActionsInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex flex-col gap-2 lg:flex-row">{children}</div>;
}
