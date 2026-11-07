import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { Badge } from "@trenova/shared/components/ui/badge";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import type { AuditEntryRow } from "@/lib/graphql/audit-log-table";
import { Resource } from "@trenova/shared/types/permission";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import {
  AUDIT_CHANNEL_LABELS,
  auditOperationFilterOptions,
  channelLabel,
  operationLabel,
  resourceLabel,
  resourceSourceLabel,
} from "./audit-log-formatters";

const auditResourceFilterOptions = Object.values(Resource).map((value) => ({
  value,
  label: resourceSourceLabel(value),
}));

const AUDIT_CATEGORY_LABELS = defineLabels({
  User: "User",
  System: "System",
  Security: "Security",
} as const);

function optionsOf<T extends Record<string, string>>(labels: T) {
  return Object.entries(labels).map(([value, label]) => ({ value, label }));
}

export function getColumns(t: TranslateFn): ColumnDef<AuditEntryRow>[] {
  return [
    {
      accessorKey: "resourceId",
      header: t("Resource ID"),
      size: 220,
      minSize: 180,
      maxSize: 260,
      meta: {
        label: t("Resource ID"),
        apiField: "resourceId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "entityRef",
      header: t("Record"),
      cell: ({ row }) => row.original.entityRef || "—",
      size: 180,
      minSize: 140,
      maxSize: 240,
      meta: {
        label: t("Record"),
        apiField: "entityRef",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "comment",
      header: t("Description"),
      cell: ({ row }) => row.original.comment || "—",
      size: 360,
      minSize: 300,
      maxSize: 480,
      meta: {
        label: t("Description"),
        apiField: "comment",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "resource",
      header: t("Resource"),
      cell: ({ row }) => resourceLabel(row.original.resource),
      size: 170,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: t("Resource"),
        apiField: "resource",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: auditResourceFilterOptions,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "operation",
      header: t("Action"),
      cell: ({ row }) => operationLabel(row.original.operation),
      size: 150,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Action"),
        apiField: "operation",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: auditOperationFilterOptions,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "timestamp",
      header: t("Timestamp"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.timestamp} />,
      size: 200,
      minSize: 170,
      maxSize: 240,
      meta: {
        label: t("Timestamp"),
        apiField: "timestamp",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      id: "actor",
      header: t("Who"),
      cell: ({ row }) => {
        const { user, actorName, principalType } = row.original;
        const name = actorName || user?.name || t("Unknown");

        return (
          <div className="flex items-center gap-3">
            <ResolvedUserAvatar
              userId={user?.id}
              name={name}
              profilePicUrl={user?.profilePicUrl}
              thumbnailUrl={user?.thumbnailUrl}
              className="bg-muted size-8 rounded-md"
              imageClassName="rounded-md bg-muted"
              fallbackClassName="text-xs"
              alt={name}
            />
            <div className="flex flex-col">
              <span className="font-medium">{name}</span>
              {user && user.emailAddress === null ? (
                <WithheldValue kind="user" className="self-start" />
              ) : (
                <span className="text-muted-foreground text-xs">
                  {user?.emailAddress || principalType}
                </span>
              )}
            </div>
          </div>
        );
      },
      size: 260,
      minSize: 220,
      maxSize: 320,
      meta: {
        label: t("Who"),
        apiField: "principalId",
        filterable: true,
        sortable: false,
        filterType: "record",
        filterRecord: "USER",
      },
    },
    {
      accessorKey: "channel",
      header: t("Channel"),
      cell: ({ row }) => channelLabel(row.original.channel),
      size: 160,
      minSize: 130,
      maxSize: 200,
      meta: {
        label: t("Channel"),
        apiField: "channel",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: optionsOf(AUDIT_CHANNEL_LABELS),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "category",
      header: t("Category"),
      cell: ({ row }) =>
        row.original.category === "Security" ? (
          <Badge variant="warning">{AUDIT_CATEGORY_LABELS.Security}</Badge>
        ) : (
          AUDIT_CATEGORY_LABELS[row.original.category]
        ),
      size: 120,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: t("Category"),
        apiField: "category",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: optionsOf(AUDIT_CATEGORY_LABELS),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "correlationId",
      header: t("Request ID"),
      cell: ({ row }) => row.original.correlationId || "—",
      size: 200,
      minSize: 160,
      maxSize: 260,
      meta: {
        label: t("Request ID"),
        apiField: "correlationId",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
  ];
}
