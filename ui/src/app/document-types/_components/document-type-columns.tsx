import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { Checkbox } from "@/components/ui/checkbox";
import { DocumentTypeSchema } from "@/lib/schemas/document-type-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<DocumentTypeSchema>[] {
  const columnHelper = createColumnHelper<DocumentTypeSchema>();
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
    createEntityColumn(columnHelper, "code", {
      accessorKey: "code",
      getHeaderText: "Code",
      getId: (documentType) => documentType.id,
      getDisplayText: (documentType) => documentType.code,
      getColor: (documentType) => documentType.color,
    }),
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
      cell: ({ row }) => <p>{row.original.name}</p>,
    },
    {
      accessorKey: "description",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} />
      ),
    },
    commonColumns.createdAt,
  ];
}
