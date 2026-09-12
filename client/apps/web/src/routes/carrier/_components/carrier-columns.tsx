import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import {
  CarrierComplianceStatusBadge,
  CarrierSafetyRatingBadge,
} from "@trenova/shared/components/status-badge";
import {
  carrierComplianceStatusChoices,
  carrierSafetyRatingChoices,
  carrierStatusChoices,
  carrierTypeChoices,
  findChoice,
} from "@/lib/choices";
import { apiService } from "@/services/api";
import type { CarrierRow } from "@/lib/graphql/carrier-table";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";

function CarrierStatusCell({ row }: { row: CarrierRow }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: Carrier["status"]) => {
      if (!row.id) return;
      await apiService.carrierService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["carrier-list"],
      });
    },
    [row.id, queryClient],
  );

  return (
    <EditableStatusBadge
      status={row.status}
      options={carrierStatusChoices}
      variants={{ DoNotUse: "inactive" }}
      onStatusChange={handleStatusChange}
    />
  );
}

export function getColumns(t: TranslateFn): ColumnDef<CarrierRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <CarrierStatusCell row={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: carrierStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => <span className="font-medium">{row.original.code}</span>,
      size: 120,
      minSize: 80,
      maxSize: 150,
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => {
        const { name, dbaName } = row.original;
        return (
          <div className="flex flex-col">
            <span>{name}</span>
            {dbaName ? (
              <span className="text-2xs text-muted-foreground">{t("DBA: {0}", dbaName)}</span>
            ) : null}
          </div>
        );
      },
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "carrierType",
      header: t("Type"),
      cell: ({ row }) => (
        <span>
          {findChoice(carrierTypeChoices, row.original.carrierType)?.label ??
            row.original.carrierType}
        </span>
      ),
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: {
        apiField: "carrierType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: carrierTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "dotNumber",
      header: t("DOT #"),
      cell: ({ row }) => <span>{row.original.dotNumber || "-"}</span>,
      size: 120,
      minSize: 90,
      maxSize: 150,
      meta: {
        apiField: "dotNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "mcNumber",
      header: t("MC #"),
      cell: ({ row }) => <span>{row.original.mcNumber || "-"}</span>,
      size: 120,
      minSize: 90,
      maxSize: 150,
      meta: {
        apiField: "mcNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "scac",
      header: "SCAC",
      cell: ({ row }) => <span>{row.original.scac || "-"}</span>,
      size: 100,
      minSize: 80,
      maxSize: 120,
      meta: {
        apiField: "scac",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "complianceStatus",
      header: t("Compliance"),
      cell: ({ row }) => <CarrierComplianceStatusBadge status={row.original.complianceStatus} />,
      size: 140,
      minSize: 110,
      maxSize: 160,
      meta: {
        apiField: "complianceStatus",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: carrierComplianceStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "safetyRating",
      header: t("Safety Rating"),
      cell: ({ row }) => <CarrierSafetyRatingBadge status={row.original.safetyRating} />,
      size: 140,
      minSize: 110,
      maxSize: 170,
      meta: {
        apiField: "safetyRating",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: carrierSafetyRatingChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
