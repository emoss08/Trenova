"use no memo";
import { Separator } from "@/components/ui/separator";
import type { ExtraAction } from "@/types/data-table";
import type { Table } from "@tanstack/react-table";
import {
  DataTableCreateButton,
  DataTableViewOptions,
} from "./data-table-view-options";

export function DataTableActions<TData>({
  table,
  name,
  exportModelName,
  extraActions,
  handleCreateClick,
}: {
  table: Table<TData>;
  name: string;
  exportModelName: string;
  handleCreateClick: () => void;
  extraActions?: ExtraAction[];
}) {
  return (
    <DataTableActionsInner>
      <DataTableViewOptions table={table} />
      <Separator className="h-6 w-px bg-border" orientation="vertical" />
      <DataTableCreateButton
        name={name}
        exportModelName={exportModelName}
        extraActions={extraActions}
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
