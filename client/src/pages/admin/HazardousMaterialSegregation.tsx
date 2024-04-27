import AdminLayout from "@/components/admin-page/layout";
import { Checkbox } from "@/components/common/fields/checkbox";
import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { HazardousMaterialEditDialog } from "@/components/hazmat-seg-rules-edit-dialog";
import { HazmatSegRulesDialog } from "@/components/hazmat-seg-rules-table-dialog";
import { Badge } from "@/components/ui/badge";
import { segregationTypeChoices } from "@/lib/choices";
import { type HazardousMaterialSegregationRule } from "@/types/shipment";
import { type FilterConfig } from "@/types/tables";
import { type ColumnDef } from "@tanstack/react-table";

const readableSegType = (type: string) => {
  switch (type) {
    case "NotAllowed":
      return <Badge variant="inactive">Not Allowed</Badge>;
    default:
      return <Badge variant="active">Allowed With Conditions</Badge>;
  }
};

const columns: ColumnDef<HazardousMaterialSegregationRule>[] = [
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
    accessorKey: "classA",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Class A" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "classB",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Class B" />
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "segregationType",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Segregation Type" />
    ),
    cell: ({ row }) => readableSegType(row.original.segregationType),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
];

const filters: FilterConfig<HazardousMaterialSegregationRule>[] = [
  {
    columnName: "segregationType",
    title: "Segregation Type",
    options: segregationTypeChoices,
  },
];

export default function HazardousMaterialSegregation() {
  return (
    <AdminLayout>
      <DataTable
        queryKey="hazardous-material-segregation-table-data"
        columns={columns}
        link="/hazardous-material-segregations/"
        name="Hazmat Seg. Rules"
        exportModelName="hazardous_material_segregations"
        filterColumn="classA"
        tableFacetedFilters={filters}
        TableSheet={HazmatSegRulesDialog}
        TableEditSheet={HazardousMaterialEditDialog}
        addPermissionName="create_hazardousmaterialsegregation"
      />
    </AdminLayout>
  );
}
