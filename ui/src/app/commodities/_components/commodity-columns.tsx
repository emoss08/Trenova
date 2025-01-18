import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { type CommoditySchema } from "@/lib/schemas/commodity-schema";
import { type ColumnDef } from "@tanstack/react-table";

function HazmatBadge({ isHazmat }: { isHazmat: boolean }) {
  return (
    <Badge variant={isHazmat ? "active" : "inactive"} className="max-h-6">
      {isHazmat ? "Yes" : "No"}
    </Badge>
  );
}

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
      accessorKey: "isHazmat",
      header: "Is Hazmat",
      cell: ({ row }) => <HazmatBadge isHazmat={row.original.isHazardous} />,
    },
  ];
}
