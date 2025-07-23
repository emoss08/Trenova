/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { type EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<EquipmentManufacturerSchema>[] {
  const columnHelper = createColumnHelper<EquipmentManufacturerSchema>();
  const commonColumns = createCommonColumns<EquipmentManufacturerSchema>();

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
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
