import {
  BooleanBadge,
  DataTableColorColumn,
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import {
  facilityTypeChoices,
  locationCategoryTypeChoices,
} from "@/lib/choices";
import { type LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import {
  FacilityType,
  mapToFacilityType,
  mapToLocationCategoryType,
} from "@/types/location-category";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<LocationCategorySchema>[] {
  return [
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => {
        const { color, name } = row.original;
        return <DataTableColorColumn text={name} color={color} />;
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
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => <p>{mapToLocationCategoryType(row.original.type)}</p>,
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: locationCategoryTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "facilityType",
      accessorKey: "facilityType",
      header: "Facility Type",
      cell: ({ row }) => (
        <p>
          {row.original.facilityType
            ? mapToFacilityType(row.original.facilityType as FacilityType)
            : ""}
        </p>
      ),
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        apiField: "facilityType",
        label: "Facility Type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: facilityTypeChoices,
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
      accessorKey: "hasSecureParking",
      header: "Has Secure Parking",
      cell: ({ row }) => <BooleanBadge value={row.original.hasSecureParking} />,
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "hasSecureParking",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "allowsOvernight",
      header: "Allows Overnight",
      cell: ({ row }) => <BooleanBadge value={row.original.allowsOvernight} />,
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "allowsOvernight",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "requiresAppointment",
      header: "Requires Appointment",
      cell: ({ row }) => (
        <BooleanBadge value={row.original.requiresAppointment} />
      ),
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "requiresAppointment",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "hasRestroom",
      header: "Has Restroom",
      cell: ({ row }) => <BooleanBadge value={row.original.hasRestroom} />,
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        apiField: "hasRestroom",
        filterable: true,
        sortable: true,
        filterType: "boolean",
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
