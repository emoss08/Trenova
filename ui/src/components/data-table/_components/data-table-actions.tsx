"use no memo";
import { Separator } from "@/components/ui/separator";
import { usePermissions } from "@/hooks/use-permissions";
import type { Resource } from "@/types/audit-entry";
import type { ExtraAction } from "@/types/data-table";
import { Action } from "@/types/roles-permissions";
import type { Table } from "@tanstack/react-table";
import {
  DataTableCreateButton,
  DataTableViewOptions,
} from "./data-table-view-options";

export default function DataTableActions<TData>({
  table,
  name,
  exportModelName,
  extraActions,
  handleCreateClick,
  resource,
}: {
  table: Table<TData>;
  name: string;
  resource: Resource;
  exportModelName: string;
  handleCreateClick: () => void;
  extraActions?: ExtraAction[];
}) {
  const { can } = usePermissions();

  console.info("resource", resource);
  return (
    <DataTableActionsInner>
      <DataTableViewOptions table={table} />
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
