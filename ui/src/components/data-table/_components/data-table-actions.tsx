"use no memo";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Separator } from "@/components/ui/separator";
import { usePermissions } from "@/hooks/use-permissions";
import type { Resource } from "@/types/audit-entry";
import type { ExtraAction } from "@/types/data-table";
import { Action } from "@/types/roles-permissions";
import { faCirclePlay, faCircleStop } from "@fortawesome/pro-solid-svg-icons";
import React from "react";
import {
  DataTableCreateButton,
  DataTableViewOptions,
} from "./data-table-view-options";

export default function DataTableActions({
  name,
  exportModelName,
  extraActions,
  handleCreateClick,
  resource,
  liveModeEnabled,
  onLiveModeToggle,
}: {
  name: string;
  resource: Resource;
  exportModelName: string;
  handleCreateClick: () => void;
  extraActions?: ExtraAction[];
  liveModeEnabled: boolean;
  onLiveModeToggle: (enabled: boolean) => void;
}) {
  const { can } = usePermissions();

  return (
    <DataTableActionsInner>
      <DataTableViewOptions resource={resource} />
      <Button
        variant={liveModeEnabled ? "green" : "outline"}
        onClick={() => onLiveModeToggle(!liveModeEnabled)}
      >
        {liveModeEnabled ? (
          <Icon icon={faCircleStop} />
        ) : (
          <Icon icon={faCirclePlay} />
        )}
        Live Mode
      </Button>
      {can(resource, Action.Create) ? (
        <>
          <Separator className="h-6 w-px bg-border" orientation="vertical" />
          <DataTableCreateButton
            name={name}
            exportModelName={exportModelName}
            extraActions={extraActions}
            onCreateClick={handleCreateClick}
          />
        </>
      ) : (
        <p className="text-xs text-destructive">
          You do not have permissions to create {resource}s
        </p>
      )}
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
