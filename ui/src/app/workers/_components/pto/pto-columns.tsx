/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { PTOStatusBadge, PTOTypeBadge } from "@/components/status-badge";
import { ptoStatusChoices, ptoTypeChoices } from "@/lib/choices";
import { type WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<WorkerPTOSchema>[] {
  const commonColumns = createCommonColumns<WorkerPTOSchema>();

  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <PTOStatusBadge status={status} />;
      },
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ptoStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "worker.firstName",
      header: "First Name",
      cell: (info) => {
        return <p>{info.getValue() as string}</p>;
      },
      meta: {
        apiField: "worker.firstName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "worker.lastName",
      header: "Last Name",
      cell: (info) => {
        return <p>{info.getValue() as string}</p>;
      },
      meta: {
        apiField: "worker.lastName",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => {
        const type = row.original.type;
        return <PTOTypeBadge type={type} />;
      },
      meta: {
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: ptoTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    commonColumns.createdAt,
  ];
}
