import { EntityRefCell } from "@/components/data-table/_components/data-table-column-helpers";
import {
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { statusChoices } from "@/lib/choices";
import { LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import type { LocationSchema } from "@/lib/schemas/location-schema";
import { formatLocation } from "@/lib/utils";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<LocationSchema>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
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
      id: "code",
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => {
        const { code } = row.original;
        return <p>{code}</p>;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => {
        const { name } = row.original;
        return <p>{name}</p>;
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "category",
      accessorKey: "locationCategory",
      header: "Location Category",
      cell: ({ row }) => {
        const { locationCategory } = row.original;
        if (!locationCategory) {
          return <p className="text-muted-foreground">-</p>;
        }
        return (
          <EntityRefCell<LocationCategorySchema, LocationSchema>
            entity={locationCategory}
            config={{
              basePath: "/dispatch/configurations/location-categories",
              getId: (locationCategory) => locationCategory.id ?? undefined,
              getDisplayText: (locationCategory) => locationCategory.name,
              color: {
                getColor: (locationCategory) => locationCategory.color,
              },
              getHeaderText: "Location Category",
            }}
            parent={row.original}
          />
        );
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "locationCategory.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "addressLine",
      header: "Address Line",
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "addressLine1",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      cell: ({ row }) => {
        return <p>{formatLocation(row.original)}</p>;
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
      size: 100,
      minSize: 100,
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
