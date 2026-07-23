import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { EDITransferStatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { ediTransferStatusChoices } from "@/lib/choices";
import type { EDITransfer } from "@/types/edi";
import type { ColumnDef } from "@tanstack/react-table";
import { LinkIcon } from "lucide-react";
import { Link } from "react-router";

export function getTransferColumns(direction: "inbound" | "outbound"): ColumnDef<EDITransfer>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <EDITransferStatusBadge status={row.original.status} />,
      size: 160,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: [...ediTransferStatusChoices],
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "partner",
      header: "Partner",
      cell: ({ row }) => {
        const partner =
          direction === "inbound" ? row.original.sourcePartner : row.original.targetPartner;
        return partner?.name ? (
          <span className="font-medium">{partner.name}</span>
        ) : (
          <DataTablePlaceholder />
        );
      },
      size: 240,
      meta: {
        label: "Partner",
        apiField: direction === "inbound" ? "sourcePartnerId" : "targetPartnerId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "reference",
      header: "Reference",
      cell: ({ row }) => {
        const payload = row.original.tenderPayload;
        return (
          <div className="min-w-0">
            <div className="truncate font-medium">{payload.bol || "Load tender"}</div>
            <div className="truncate text-xs text-muted-foreground">
              {payload.customerLabel || payload.serviceTypeLabel || "No tender summary"}
            </div>
          </div>
        );
      },
      size: 280,
      meta: {
        label: "Reference",
        apiField: "tenderPayload.bol",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "submittedAt",
      header: "Submitted",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.submittedAt} />,
      size: 180,
      meta: {
        label: "Submitted",
        apiField: "submittedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "targetShipmentId",
      header: "Target Shipment",
      cell: ({ row }) =>
        row.original.targetShipmentId ? (
          <Link
            className="inline-flex items-center gap-1 text-primary underline-offset-4 hover:underline"
            to={`/shipment-management/shipments?item=${row.original.targetShipmentId}`}
          >
            <LinkIcon className="size-3.5" />
            Open shipment
          </Link>
        ) : (
          <DataTablePlaceholder text="Pending" />
        ),
      size: 180,
      meta: {
        label: "Target Shipment",
        apiField: "targetShipmentId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "mappingSummary",
      header: "Mappings",
      cell: ({ row }) => {
        const unresolvedCount = row.original.mappingSnapshot.filter(
          (mapping) => !mapping.resolved,
        ).length;
        const totalCount = row.original.mappingSnapshot.length;

        if (totalCount === 0) {
          return <DataTablePlaceholder text="No requirements" />;
        }

        return (
          <div className="flex flex-wrap gap-1">
            <Badge variant={unresolvedCount > 0 ? "outline" : "active"}>
              {unresolvedCount > 0 ? `${unresolvedCount} unresolved` : "Resolved"}
            </Badge>
            <Badge variant="secondary">{totalCount} total</Badge>
          </div>
        );
      },
      size: 220,
      meta: {
        label: "Mappings",
        apiField: "mappingSnapshot",
        filterable: false,
        sortable: false,
      },
    },
  ];
}
