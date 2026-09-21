import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { fieldTypeChoices } from "@/lib/choices";
import type { CustomFieldDefinitionRow } from "@/lib/graphql/custom-field-definition-table";
import type { FieldType } from "@/types/custom-field";
import type { ColumnDef } from "@trenova/shared/types/data-table";

const fieldTypeBadgeVariants: Record<FieldType, BadgeVariant> = {
  text: "info",
  number: "info",
  date: "info",
  boolean: "warning",
  select: "warning",
  multiSelect: "info",
};

export function getColumns(t: TranslateFn): ColumnDef<CustomFieldDefinitionRow>[] {
  return [
    {
      accessorKey: "label",
      header: t("Label"),
      cell: ({ row }) => {
        const { color, label } = row.original;
        return <DataTableColorColumn text={label} color={color ?? undefined} />;
      },
      meta: {
        label: t("Label"),
        apiField: "label",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "name",
      header: t("Name"),
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
      accessorKey: "resourceType",
      header: t("Resource type"),
      cell: ({ row }) => (
        <Badge variant="neutral" appearance="outline" className="capitalize">
          {row.original.resourceType}
        </Badge>
      ),
      meta: {
        label: t("Resource type"),
        apiField: "resourceType",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "fieldType",
      header: t("Field type"),
      cell: ({ row }) => {
        const fieldType = row.original.fieldType;
        const choice = fieldTypeChoices.find((c) => c.value === fieldType);
        const variant = fieldTypeBadgeVariants[fieldType] || "default";
        return <Badge variant={variant}>{choice?.label || fieldType}</Badge>;
      },
      meta: {
        label: t("Field type"),
        apiField: "fieldType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: fieldTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "isRequired",
      header: t("Required"),
      cell: ({ row }) => (
        <Badge variant={row.original.isRequired ? "success" : "danger"}>
          {row.original.isRequired ? t("Yes") : t("No")}
        </Badge>
      ),
      size: 100,
    },
    {
      accessorKey: "isActive",
      header: t("Active"),
      cell: ({ row }) => (
        <Badge variant={row.original.isActive ? "success" : "danger"}>
          {row.original.isActive ? t("Active") : t("Inactive")}
        </Badge>
      ),
      size: 100,
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: {
        label: t("Created At"),
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
    },
  ];
}
