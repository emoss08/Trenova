/* eslint-disable react-refresh/only-export-components */
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { type BadgeAttrProps } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { accessorialChargeMethodChoices, statusChoices } from "@/lib/choices";
import { formatCurrency } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { AccessorialCharge, RateUnit } from "@/types/accessorial-charge";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback } from "react";

const rateUnitAbbreviations: Record<RateUnit, string> = {
  Mile: "/mi",
  Hour: "/hr",
  Day: "/day",
  Stop: "/stop",
};

function formatAmount(row: AccessorialCharge): string {
  const { method, amount, rateUnit } = row;

  if (method === "Percentage") {
    return `${amount}%`;
  }

  const formatted = formatCurrency(amount);

  if (method === "PerUnit" && rateUnit) {
    return `${formatted}${rateUnitAbbreviations[rateUnit]}`;
  }

  return formatted;
}

function MethodBadge({ method }: { method: AccessorialCharge["method"] }) {
  const methodAttributes: Record<AccessorialCharge["method"], BadgeAttrProps> =
    {
      Flat: {
        variant: "active",
        text: "Flat",
      },
      PerUnit: {
        variant: "indigo",
        text: "Per Unit",
      },
      Percentage: {
        variant: "warning",
        text: "Percentage",
      },
    };
  return (
    <Badge variant={methodAttributes[method].variant} className="max-h-6">
      {methodAttributes[method].text}
    </Badge>
  );
}

function AccessorialChargeStatusCell({ row }: { row: AccessorialCharge }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: AccessorialCharge["status"]) => {
      if (!row.id) return;
      await apiService.accessorialChargeService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["accessorial-charge-list"],
      });
    },
    [row.id, queryClient],
  );

  return (
    <EditableStatusBadge
      status={row.status}
      options={statusChoices}
      onStatusChange={handleStatusChange}
    />
  );
}

export function getColumns(): ColumnDef<AccessorialCharge>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        return <AccessorialChargeStatusCell row={row.original} />;
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
      cell: ({ row }) => {
        const code = row.original.code;
        return <p>{code}</p>;
      },
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "method",
      header: "Method",
      cell: ({ row }) => {
        const method = row.original.method;
        return <MethodBadge method={method} />;
      },
      meta: {
        apiField: "method",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: accessorialChargeMethodChoices,
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
      size: 400,
      minSize: 400,
      maxSize: 500,
      meta: {
        apiField: "description",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "amount",
      header: "Rate",
      cell: ({ row }) => {
        return <p>{formatAmount(row.original)}</p>;
      },
      meta: {
        apiField: "amount",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return (
          <HoverCardTimestamp
            className="shrink-0"
            timestamp={row.original.createdAt}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        label: "Created At",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
