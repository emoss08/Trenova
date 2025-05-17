import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { type ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<ServiceTypeSchema>[] {
  const columnHelper = createColumnHelper<ServiceTypeSchema>();
  const commonColumns = createCommonColumns<ServiceTypeSchema>();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => {
        const { color, code } = row.original;
        return <DataTableColorColumn text={code} color={color} />;
      },
    }),
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
