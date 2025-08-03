/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<ShipmentTypeSchema>[] {
  const columnHelper = createColumnHelper<ShipmentTypeSchema>();
  const commonColumns = createCommonColumns<ShipmentTypeSchema>();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => {
        const { color, code } = row.original;
        return <DataTableColorColumn text={code} color={color} />;
      },
    }),
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
