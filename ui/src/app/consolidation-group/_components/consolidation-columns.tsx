/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { ConsolidationStatusBadge } from "@/components/status-badge";
import { consolidationStatusChoices } from "@/lib/choices";
import { type ConsolidationGroupSchema } from "@/lib/schemas/consolidation-schema";
import { type ColumnDef } from "@tanstack/react-table";
import { Package } from "lucide-react";

export function getColumns(): ColumnDef<ConsolidationGroupSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <ConsolidationStatusBadge status={status} />;
      },
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: consolidationStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "consolidationNumber",
      header: "Consol. Number",
      cell: ({ row }) => {
        const number = row.original.consolidationNumber;
        return <p>{number || "-"}</p>;
      },
      meta: {
        apiField: "consolidationNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      enableHiding: false,
    },
    {
      accessorKey: "totalShipments",
      header: "Shipments",
      cell: ({ row }) => {
        const totalShipments = row.original.shipments?.length || 0;
        return (
          <div className="flex items-center gap-2">
            <Package className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="font-medium">{totalShipments}</span>
          </div>
        );
      },
      meta: {
        apiField: "totalShipments",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      cell: ({ row }) => {
        return (
          <HoverCardTimestamp
            className="font-table tracking-tight"
            timestamp={row.original.createdAt}
          />
        );
      },
      meta: {
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "gte",
      },
    },
  ];
}
