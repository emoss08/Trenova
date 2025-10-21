/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { PackingGroupBadge, StatusBadge } from "@/components/status-badge";
import {
  hazardousClassChoices,
  packingGroupChoices,
  statusChoices,
} from "@/lib/choices";
import { type HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import { Status } from "@/types/common";
import {
  HazardousClassChoiceProps,
  mapToHazardousClassChoice,
} from "@/types/hazardous-material";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HazardousMaterialSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status as Status} />;
      },
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => <p>{row.original.code}</p>,
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "class",
      header: "Class",
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.class as HazardousClassChoiceProps,
        ),
      meta: {
        apiField: "class",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: hazardousClassChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.description}
          truncateLength={100}
        />
      ),
      meta: {
        apiField: "description",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "packingGroup",
      header: "Packing Group",
      cell: ({ row }) => (
        <PackingGroupBadge group={row.original.packingGroup} />
      ),
      meta: {
        apiField: "packingGroup",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: packingGroupChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
    },
  ];
}
