import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { Checkbox } from "@/components/ui/checkbox";
import { type HazmatSegregationRuleSchema } from "@/lib/schemas/hazmat-segregation-rule-schema";
import {
  HazardousClassChoiceProps,
  mapToHazardousClassChoice,
} from "@/types/hazardous-material";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<HazmatSegregationRuleSchema>[] {
  const columnHelper = createColumnHelper<HazmatSegregationRuleSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    {
      id: "select",
      header: ({ table }) => {
        const isAllSelected = table.getIsAllPageRowsSelected();
        const isSomeSelected = table.getIsSomePageRowsSelected();

        return (
          <Checkbox
            data-slot="select-all"
            checked={isAllSelected || (isSomeSelected && "indeterminate")}
            onCheckedChange={(checked) =>
              table.toggleAllPageRowsSelected(!!checked)
            }
            aria-label="Select all"
          />
        );
      },
      cell: ({ row }) => (
        <Checkbox
          data-slot="select-row"
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(!!checked)}
          aria-label="Select row"
        />
      ),
      size: 50,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
    },
    createEntityColumn(columnHelper, "name", {
      accessorKey: "name",
      getHeaderText: "Name",
      getId: (hazmatSegregationRule) => hazmatSegregationRule.id,
      getDisplayText: (hazmatSegregationRule) => hazmatSegregationRule.name,
    }),
    {
      accessorKey: "classA",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Class A" />
      ),
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.classA as HazardousClassChoiceProps,
        ),
    },
    {
      accessorKey: "classB",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Class B" />
      ),
      cell: ({ row }) =>
        mapToHazardousClassChoice(
          row.original.classB as HazardousClassChoiceProps,
        ),
    },
    {
      accessorKey: "description",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description ?? ""} />
      ),
    },
    commonColumns.createdAt,
  ];
}
