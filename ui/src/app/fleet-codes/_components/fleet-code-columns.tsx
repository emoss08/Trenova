/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import type { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<FleetCodeSchema>[] {
  const columnHelper = createColumnHelper<FleetCodeSchema>();
  const commonColumns = createCommonColumns<FleetCodeSchema>();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => {
        const { color, name } = row.original;
        return <DataTableColorColumn text={name} color={color} />;
      },
    }),
    commonColumns.description,
    {
      id: "manager",
      header: "Manager",
      cell: ({ row }) => {
        const { manager } = row.original;
        if (!manager) return "-";
        return <p>{manager.name}</p>;
      },
    },
    commonColumns.createdAt,
  ];
}
