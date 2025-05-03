import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { DocumentTypeSchema } from "@/lib/schemas/document-type-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<DocumentTypeSchema>[] {
  const columnHelper = createColumnHelper<DocumentTypeSchema>();
  const commonColumns = createCommonColumns();

  return [
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => {
        const { color, code } = row.original;
        return <DataTableColorColumn text={code} color={color} />;
      },
    }),
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <p>{row.original.name}</p>,
    },
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
