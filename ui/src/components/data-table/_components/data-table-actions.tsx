"use no memo";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Skeleton } from "@/components/ui/skeleton";
import { usePermissions } from "@/hooks/use-permission";
import type { Resource } from "@/types/audit-entry";
import type { ExtraAction } from "@/types/data-table";
import type { LiveModeTableConfig } from "@/types/live-mode";
import { Action } from "@/types/roles-permissions";
import { faCirclePlay, faCircleStop } from "@fortawesome/pro-solid-svg-icons";
import React from "react";
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
}: {
  name: string;
  resource: Resource;
  exportModelName: string;
  handleCreateClick: () => void;
  liveModeEnabled: boolean;
  onLiveModeToggle: (enabled: boolean) => void;
  extraActions?: ExtraAction[];
  liveModeConfig?: LiveModeTableConfig;
}) {
  const { can } = usePermissions();

  return (
    <DataTableActionsInner>
      <DataTableViewOptions resource={resource} />
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
  return <div className="flex items-center gap-2">{children}</div>;
}

export function DataTableActionsSkeleton() {
  return (
    <div className="flex items-center gap-2">
      <Skeleton className="h-7 w-44 rounded-md" />
      <Skeleton className="h-7 w-26 rounded-md" />
      <div className="h-6 w-px bg-border" />
      <Skeleton className="h-7 w-18 rounded-md" />
    </div>
  );
}
