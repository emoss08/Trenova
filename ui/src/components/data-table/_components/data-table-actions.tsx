"use no memo";
import { Separator } from "@/components/ui/separator";
import { usePermissions } from "@/hooks/use-permissions";
import type { Resource } from "@/types/audit-entry";
import type { ExtraAction } from "@/types/data-table";
import { Action } from "@/types/roles-permissions";
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
}: {
  name: string;
  resource: Resource;
  exportModelName: string;
  handleCreateClick: () => void;
  extraActions?: ExtraAction[];
}) {
  const { can } = usePermissions();

  return (
    <DataTableActionsInner>
      <DataTableViewOptions resource={resource} />
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
