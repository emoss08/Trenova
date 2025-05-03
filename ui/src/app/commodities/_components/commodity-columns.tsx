import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { HazmatBadge } from "@/components/status-badge";
import { type CommoditySchema } from "@/lib/schemas/commodity-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<CommoditySchema>[] {
  const columnHelper = createColumnHelper<CommoditySchema>();
  const commonColumns = createCommonColumns();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => <p>{row.original.name}</p>,
    }),
    commonColumns.description,
    columnHelper.display({
      id: "temperatureRange",
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
    }),
    columnHelper.display({
      id: "isHazmat",
      header: "Is Hazmat",
      cell: ({ row }) => (
        <HazmatBadge isHazmat={!!row.original.hazardousMaterialId} />
      ),
    }),
    commonColumns.createdAt,
  ];
}
