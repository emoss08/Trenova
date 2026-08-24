import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { RateMatrix } from "@trenova/shared/types/rate";

export function getColumns(): ColumnDef<RateMatrix>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const choice = statusChoices.find((option) => option.value === row.original.status);

        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.status
        );
      },
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: "Status",
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
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.code}</span>,
      size: 120,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: "Code",
        apiField: "code",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      size: 240,
      minSize: 200,
      maxSize: 320,
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
      },
    },
    {
      // Without this the list is a set of grids of anonymous numbers. The same
      // grid is a per-mile tariff or a flat table depending on which formula
      // template prices it, and nothing else on the row says which.
      accessorKey: "formulaTemplateName",
      header: "Rates are",
      cell: ({ row }) => <span className="text-sm">{row.original.formulaTemplateName || "—"}</span>,
      size: 170,
      minSize: 140,
      maxSize: 210,
      meta: {
        label: "Rates are",
        apiField: "formulaTemplateId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "currency",
      header: "Currency",
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.currency}</span>,
      size: 100,
      minSize: 90,
      maxSize: 120,
      meta: {
        label: "Currency",
        apiField: "currency",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => <DataTableDescription description={row.original.description} />,
      size: 300,
      minSize: 200,
      maxSize: 420,
      meta: {
        label: "Description",
        apiField: "description",
        filterable: true,
        sortable: false,
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 150,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: "Created",
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
