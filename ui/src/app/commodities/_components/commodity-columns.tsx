import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { HazmatBadge, StatusBadge } from "@/components/status-badge";
import { type CommoditySchema } from "@/lib/schemas/commodity-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<CommoditySchema>[] {
  const columnHelper = createColumnHelper<CommoditySchema>();
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
    createEntityColumn(columnHelper, "name", {
      accessorKey: "name",
      getHeaderText: "Name",
      getId: (commodity) => commodity.id,
      getDisplayText: (commodity) => commodity.name,
    }),
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
    commonColumns.createdAt,
  ];
}
