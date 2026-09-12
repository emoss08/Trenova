import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { EditableStatusBadge } from "@/components/editable-status-badge";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { accountCategoryChoices, statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { AccountTypeRow } from "@/lib/graphql/account-type-table";
import type { AccountType } from "@/types/account-type";
import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { useCallback } from "react";

function AccountTypeStatusCell({ row }: { row: AccountTypeRow }) {
  const queryClient = useQueryClient();

  const handleStatusChange = useCallback(
    async (newStatus: AccountType["status"]) => {
      if (!row.id) return;
      await apiService.accountTypeService.patch(row.id, {
        status: newStatus,
      });

      await queryClient.invalidateQueries({
        queryKey: ["account-type-list"],
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

export function getColumns(t: TranslateFn): ColumnDef<AccountTypeRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <AccountTypeStatusCell row={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
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
      header: t("Code"),
      cell: ({ row }) => {
        const { code, color } = row.original;
        return <DataTableColorColumn text={code} color={color ?? undefined} />;
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
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => row.original.name,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "category",
      header: t("Category"),
      cell: ({ row }) => {
        const choice = accountCategoryChoices.find((c) => c.value === row.original.category);
        if (!choice) return row.original.category;
        return <DataTableColorColumn text={t(choice.label)} color={choice.color} />;
      },
      meta: {
        apiField: "category",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: accountCategoryChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={100} />
      ),
      size: 400,
      minSize: 300,
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
