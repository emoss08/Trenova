/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { PackingGroupBadge } from "@/components/status-badge";
import { type HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import {
  HazardousClassChoiceProps,
  mapToHazardousClassChoice,
} from "@/types/hazardous-material";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HazardousMaterialSchema>[] {
  const columnHelper = createColumnHelper<HazardousMaterialSchema>();
  const commonColumns = createCommonColumns<HazardousMaterialSchema>();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => row.original.code,
    }),
    {
      accessorKey: "class",
      header: "Class",
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.class as HazardousClassChoiceProps,
        ),
    },
    commonColumns.description,
    {
      accessorKey: "packingGroup",
      header: "Packing Group",
      cell: ({ row }) => (
        <PackingGroupBadge group={row.original.packingGroup} />
      ),
    },
    commonColumns.createdAt,
  ];
}
