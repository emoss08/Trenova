import { Separator } from "@/components/ui/separator";
import type { ExtraAction } from "@/types/data-table";
import type { Table } from "@tanstack/react-table";
import {
  DataTableCreateButton,
  DataTableViewOptions,
} from "./data-table-view-options";

export function DataTableFilter<TData>({
  table,
  name,
  exportModelName,
  extraActions,
  setModalType,
}: {
  table: Table<TData>;
  name: string;
  exportModelName: string;
  setModalType: (modalType: "create" | "edit") => void;
  extraActions?: ExtraAction[];
}) {
  return (
    <div className="flex items-center gap-2">
      <DataTableViewOptions table={table} />
      <Separator className="h-6 w-px bg-border" orientation="vertical" />
      <DataTableCreateButton
        name={name}
        exportModelName={exportModelName}
        extraActions={extraActions}
        onCreateClick={() => setModalType("create")}
      />
    </div>
  );
}
