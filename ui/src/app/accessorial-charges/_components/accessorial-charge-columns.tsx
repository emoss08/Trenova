import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<AccessorialChargeSchema>[] {
  const columnHelper = createColumnHelper<AccessorialChargeSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    commonColumns.status,
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => <p>{row.original.code}</p>,
    }),
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
