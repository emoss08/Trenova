import {
  createCommonColumns,
  createEntityRefColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import { formatLocation } from "@/lib/utils";
import { Location } from "@/types/location";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<LocationSchema>[] {
  const columnHelper = createColumnHelper<LocationSchema>();
  const commonColumns = createCommonColumns<LocationSchema>();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => {
        const { name } = row.original;
        return <p>{name}</p>;
      },
    }),
    createEntityRefColumn<Location, "locationCategory">(
      columnHelper,
      "locationCategory",
      {
        basePath: "/dispatch/configurations/location-categories",
        getHeaderText: "Location Category",
        getId: (locationCategory) => locationCategory.id ?? undefined,
        getDisplayText: (locationCategory) => locationCategory.name,
        color: {
          getColor: (locationCategory) => locationCategory.color,
        },
      },
    ),
    commonColumns.description,
    {
      id: "addressLine",
      header: "Address Line",
      cell: ({ row }) => {
        return <p>{formatLocation(row.original)}</p>;
      },
    },

    commonColumns.createdAt,
  ];
}
