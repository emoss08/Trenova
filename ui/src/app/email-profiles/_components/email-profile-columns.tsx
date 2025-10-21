import { BooleanBadge, StatusBadge } from "@/components/status-badge";
import { providerTypeChoices, statusChoices } from "@/lib/choices";
import type { EmailProfileSchema } from "@/lib/schemas/email-profile-schema";
import { type ColumnDef } from "@tanstack/react-table";

export const getProviderIcon = (
  providerType: EmailProfileSchema["providerType"],
) => {
  return providerTypeChoices.find((choice) => choice.value === providerType)
    ?.icon;
};

export function getColumns(): ColumnDef<EmailProfileSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <StatusBadge status={status} />;
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
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => {
        const name = row.original.name;
        return <p>{name}</p>;
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
      accessorKey: "host",
      header: "Host",
      cell: ({ row }) => {
        const host = row.original.host;
        if (!host) return <p>N/A</p>;

        // Show first 5 characters clearly, blur the rest
        const visibleLength = Math.min(5, host.length);
        const visiblePart = host.slice(0, visibleLength);
        const blurredPart = host.slice(visibleLength);

        return (
          <p className="flex items-center">
            <span>{visiblePart}</span>
            <span className="relative inline-flex items-center justify-center px-1 py-0.5 ml-0.5 rounded-md bg-foreground/10 bg-clip-padding backdrop-filter backdrop-blur-lg border border-border select-none">
              <span className="blur-sm opacity-60">{blurredPart}</span>
            </span>
          </p>
        );
      },
      minSize: 100,
      meta: {
        apiField: "host",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "providerType",
      header: "Provider Type",
      cell: ({ row }) => {
        const providerType = row.original.providerType;
        return (
          <div className="flex items-center gap-1 [&_svg]:size-4 [&_svg]:shrink-0">
            {getProviderIcon(providerType)}
            <p>{providerType}</p>
          </div>
        );
      },
      meta: {
        apiField: "providerType",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "isDefault",
      header: "Is Default Profile?",
      cell: ({ row }) => {
        const isDefault = row.original.isDefault;
        return <BooleanBadge value={isDefault} />;
      },
      meta: {
        apiField: "isDefault",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
  ];
}
