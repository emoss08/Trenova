import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HazmatBadge, StatusBadge } from "@/components/status-badge";
import { Checkbox } from "@/components/ui/checkbox";
import { type CommoditySchema } from "@/lib/schemas/commodity-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<CommoditySchema>[] {
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
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
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
      id: "temperatureRange",
      accessorFn: (row) => {
        return `${row.minTemperature}°F - ${row.maxTemperature}°F`;
      },
      header: "Temperature Range",
      cell: ({ row }) => {
        if (row.original?.minTemperature && row.original?.maxTemperature) {
          return (
            <span>
              {row.original.minTemperature}&deg;F -{" "}
              {row.original.maxTemperature}&deg;F
            </span>
          );
        }

        return "No Temperature Range";
      },
    },
    {
      id: "isHazmat",
      accessorKey: "hazardousMaterialId",
      header: "Is Hazmat",
      cell: ({ row }) => (
        <HazmatBadge isHazmat={!!row.original.hazardousMaterialId} />
      ),
    },
  ];
}
