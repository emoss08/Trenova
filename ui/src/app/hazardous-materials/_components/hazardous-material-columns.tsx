import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { PackingGroupBadge, StatusBadge } from "@/components/status-badge";
import { type HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import {
  HazardousClassChoiceProps,
  mapToHazardousClassChoice,
} from "@/types/hazardous-material";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HazardousMaterialSchema>[] {
  const columnHelper = createColumnHelper<HazardousMaterialSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    commonColumns.selection,
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
    },
    {
      accessorKey: "code",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Code" />
      ),
    },
    {
      accessorKey: "class",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Class" />
      ),
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.class as HazardousClassChoiceProps,
        ),
    },
    {
      accessorKey: "description",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} />
      ),
    },
    {
      accessorKey: "packingGroup",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Packing Group" />
      ),
      cell: ({ row }) => (
        <PackingGroupBadge group={row.original.packingGroup} />
      ),
    },
  ];
}
