import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { DocumentClassDialog } from "@/components/document-class-table-dialog";
import { DocumentClassEditDialog } from "@/components/document-class-table-edit-dialog";
import { DocumentClassification } from "@/types/billing";
import { type ColumnDef } from "@tanstack/react-table";

const columns: ColumnDef<DocumentClassification>[] = [
  {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        className="translate-y-[2px]"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        className="translate-y-[2px]"
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => <StatusBadge status={row.original.status} />,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "code",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Code" />
    ),
    cell: ({ row }) => {
      if (row.original.color) {
        return (
          <div className="text-foreground flex items-center space-x-2 text-sm font-medium">
            <div
              className={"mx-2 size-2 rounded-xl"}
              style={{ backgroundColor: row.original.color }}
            />
            {row.original.code}
          </div>
        );
      } else {
        return row.original.code;
      }
    },
  },
  {
    accessorKey: "description",
    header: "Description",
  },
];

export default function DocumentClassificationPage() {
  return (
    <DataTable
      queryKey="document-classification-table-data"
      columns={columns}
      link="/document-classifications/"
      name="Document Class."
      exportModelName="document_classifications"
      filterColumn="name"
      TableSheet={DocumentClassDialog}
      TableEditSheet={DocumentClassEditDialog}
      addPermissionName="documentclassification.add"
    />
  );
}
