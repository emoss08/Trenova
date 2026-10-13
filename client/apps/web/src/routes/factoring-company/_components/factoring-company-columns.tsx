import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { statusChoices } from "@/lib/choices";
import type { FactoringCompany } from "@/types/factoring-company";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<FactoringCompany>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        label: t("Status"),
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
      cell: ({ row }) => row.original.code,
      size: 120,
      minSize: 90,
      maxSize: 160,
      meta: {
        label: t("Code"),
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
      size: 260,
      minSize: 180,
      maxSize: 360,
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "nonRecourse",
      header: t("Recourse"),
      cell: ({ row }) => (row.original.nonRecourse ? t("Non-recourse") : t("Recourse")),
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Recourse"),
        apiField: "nonRecourse",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "city",
      header: t("City"),
      cell: ({ row }) => row.original.city,
      size: 180,
      minSize: 120,
      maxSize: 240,
      meta: {
        label: t("City"),
        apiField: "city",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "email",
      header: t("Email"),
      cell: ({ row }) => row.original.email || "—",
      size: 240,
      minSize: 160,
      maxSize: 320,
      meta: {
        label: t("Email"),
        apiField: "email",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "phoneNumber",
      header: t("Phone"),
      cell: ({ row }) => row.original.phoneNumber || "—",
      size: 160,
      minSize: 120,
      maxSize: 200,
      meta: {
        label: t("Phone"),
        apiField: "phoneNumber",
        filterable: false,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
