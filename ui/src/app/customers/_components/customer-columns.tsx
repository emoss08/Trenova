import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<CustomerSchema>[] {
  const columnHelper = createColumnHelper<CustomerSchema>();
  const commonColumns = createCommonColumns();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => <p>{row.original.code}</p>,
    }),
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => <p>{row.original.name}</p>,
    }),
    commonColumns.createdAt,
  ];
}
