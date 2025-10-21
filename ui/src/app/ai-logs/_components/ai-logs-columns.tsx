import { DataTableColumnHeaderWithTooltip } from "@/components/data-table/_components/data-table-column-header";
import {
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { UserAvatar } from "@/components/nav-user";
import { AILogSchema } from "@/lib/schemas/ai-log-schema";
import { ModelTypes, OperationTypes } from "@/types/ai-logs";
import { type ColumnDef } from "@tanstack/react-table";
import { ModelBadge, OperationBadge } from "./column-components";

export function getColumns(): ColumnDef<AILogSchema>[] {
  return [
    {
      id: "operation",
      accessorKey: "operation",
      header: "Operation",
      cell: ({ row }) => {
        const { operation } = row.original;
        return <OperationBadge operation={operation as OperationTypes} />;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "operation",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "model",
      header: "Model",
      cell: ({ row }) => (
        <ModelBadge model={row.original.model as ModelTypes} />
      ),
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "model",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "object",
      accessorKey: "object",
      header: "Object",
      cell: ({ row }) => <p>{row.original.object}</p>,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "object",
        label: "Object",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "response",
      header: "Response",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.response}
          truncateLength={100}
        />
      ),
      size: 400,
      minSize: 400,
      maxSize: 500,
      meta: {
        apiField: "response",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "prompt",
      header: "Prompt",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.prompt}
          truncateLength={100}
        />
      ),
      size: 400,
      minSize: 400,
      maxSize: 500,
      meta: {
        apiField: "prompt",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "serviceTier",
      header: "Service Tier",
      cell: ({ row }) => <p>{row.original.serviceTier}</p>,
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "serviceTier",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "promptTokens",
      header: "Prompt Tokens",
      cell: ({ row }) => <p>{row.original.promptTokens}</p>,
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "promptTokens",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "completionTokens",
      header: "Completion Tokens",
      cell: ({ row }) => <p>{row.original.completionTokens}</p>,
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "completionTokens",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "totalTokens",
      header: "Total Tokens",
      cell: ({ row }) => <p>{row.original.totalTokens}</p>,
      size: 100,
      minSize: 75,
      maxSize: 180,
      meta: {
        apiField: "totalTokens",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "user",
      header: ({ column }) => (
        <DataTableColumnHeaderWithTooltip
          column={column}
          title="User"
          tooltipContent="The user that performed the action."
        />
      ),
      cell: ({ row }) => {
        const { user } = row.original;
        if (!user) {
          return <p>Unknown</p>;
        }

        return <UserAvatar user={user} />;
      },
      size: 300,
      minSize: 300,
      maxSize: 350,
      meta: {
        apiField: "user.name",
        label: "User Name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "timestamp",
      header: "Timestamp",
      cell: ({ row }) => {
        return (
          <HoverCardTimestamp
            className="shrink-0"
            timestamp={row.original.timestamp}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "timestamp",
        label: "Timestamp",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
