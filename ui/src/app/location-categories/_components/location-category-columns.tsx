/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { type LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import {
  FacilityType,
  mapToFacilityType,
  mapToLocationCategoryType,
} from "@/types/location-category";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<LocationCategorySchema>[] {
  const columnHelper = createColumnHelper<LocationCategorySchema>();
  const commonColumns = createCommonColumns<LocationCategorySchema>();

  return [
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => {
        const { color, name } = row.original;
        return <DataTableColorColumn text={name} color={color} />;
      },
    }),
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => <p>{mapToLocationCategoryType(row.original.type)}</p>,
    },
    {
      accessorKey: "facilityType",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Facility Type" />
      ),
      cell: ({ row }) => (
        <p>
          {row.original.facilityType
            ? mapToFacilityType(row.original.facilityType as FacilityType)
            : ""}
        </p>
      ),
    },
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
