/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

"use no memo";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Skeleton } from "@/components/ui/skeleton";
import { usePermissions } from "@/hooks/use-permissions";
import type { Resource } from "@/types/audit-entry";
import type { ExtraAction } from "@/types/data-table";
import type { LiveModeTableConfig } from "@/types/live-mode";
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

      {can(resource, Action.Create) ? (
        <>
          <div className="h-6 w-px bg-border" />
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
