import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { PackingGroupBadge, StatusBadge } from "@/components/status-badge";
import { Checkbox } from "@/components/ui/checkbox";
import { type HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import {
  HazardousClassChoiceProps,
  mapToHazardousClassChoice,
} from "@/types/hazardous-material";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HazardousMaterialSchema>[] {
  return [
    {
      accessorKey: "select",
      id: "select",
      header: ({ table }) => {
        return (
          <Checkbox
            checked={
              table.getIsAllPageRowsSelected() ||
              (table.getIsSomePageRowsSelected() && "indeterminate")
            }
            onCheckedChange={(checked) =>
              table.toggleAllPageRowsSelected(!!checked)
            }
            aria-label="Select all"
          />
        );
      },
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(!!checked)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
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
