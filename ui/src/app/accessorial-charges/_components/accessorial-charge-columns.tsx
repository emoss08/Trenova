/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<AccessorialChargeSchema>[] {
  const columnHelper = createColumnHelper<AccessorialChargeSchema>();
  const commonColumns = createCommonColumns<AccessorialChargeSchema>();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => <p>{row.original.code}</p>,
    }),
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
