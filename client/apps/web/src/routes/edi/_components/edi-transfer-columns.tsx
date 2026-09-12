import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { EDITransferStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ediTransferStatusChoices } from "@/lib/choices";
import type { EDITransferRow } from "@/lib/graphql/edi-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { LinkIcon } from "lucide-react";
import { Link } from "react-router";

export function getTransferColumns(
  direction: "inbound" | "outbound",
  t: TranslateFn,
): ColumnDef<EDITransferRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <EDITransferStatusBadge status={row.original.status} />,
      size: 160,
      meta: {
        label: t("Status"),
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
      header: t("Partner"),
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
        label: t("Partner"),
        apiField: direction === "inbound" ? "sourcePartnerId" : "targetPartnerId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "reference",
      header: t("Reference"),
      cell: ({ row }) => {
        const payload = row.original.tenderPayload;
        return (
          <div className="min-w-0">
            <div className="truncate font-medium">{payload.bol || t("Load tender")}</div>
            <div className="text-muted-foreground truncate text-xs">
              {payload.customerLabel || payload.serviceTypeLabel || t("No tender summary")}
            </div>
          </div>
        );
      },
      size: 280,
      meta: {
        label: t("Reference"),
        apiField: "tenderPayload.bol",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "submittedAt",
      header: t("Submitted"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.submittedAt} />,
      size: 180,
      meta: {
        label: t("Submitted"),
        apiField: "submittedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
    {
      accessorKey: "targetShipmentId",
      header: t("Target Shipment"),
      cell: ({ row }) =>
        row.original.targetShipmentId ? (
          <Link
            className="text-primary inline-flex items-center gap-1 underline-offset-4 hover:underline"
            to={`/shipment-management/shipments?item=${row.original.targetShipmentId}`}
          >
            <LinkIcon className="size-3.5" />
            {t("Open shipment")}
          </Link>
        ) : (
          <DataTablePlaceholder text={t("Pending")} />
        ),
      size: 180,
      meta: {
        label: t("Target Shipment"),
        apiField: "targetShipmentId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "mappingSummary",
      header: t("Mappings"),
      cell: ({ row }) => {
        const unresolvedCount = row.original.mappingSnapshot.filter(
          (mapping) => !mapping.resolved,
        ).length;
        const totalCount = row.original.mappingSnapshot.length;

        if (totalCount === 0) {
          return <DataTablePlaceholder text={t("No requirements")} />;
        }

        return (
          <div className="flex flex-wrap gap-1">
            <Badge variant={unresolvedCount > 0 ? "outline" : "active"}>
              {unresolvedCount > 0 ? t("{0} unresolved", unresolvedCount) : t("Resolved")}
            </Badge>
            <Badge variant="secondary">{t("{0} total", totalCount)}</Badge>
          </div>
        );
      },
      size: 220,
      meta: {
        label: t("Mappings"),
        apiField: "mappingSnapshot",
        filterable: false,
        sortable: false,
      },
    },
  ];
}
