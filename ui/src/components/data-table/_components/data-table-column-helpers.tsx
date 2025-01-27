import { Checkbox } from "@/components/ui/checkbox";
import { ColumnHelper } from "@tanstack/react-table";

export function createCommonColumns<T extends Record<string, unknown>>(
  columnHelper: ColumnHelper<T>,
) {
  return {
    selection: columnHelper.display({
      id: "select",
      header: ({ table }) => {
        return (
          <Checkbox
            checked={
              table.getIsAllPageRowsSelected() ||
              (table.getIsSomePageRowsSelected() && "indeterminate")
            }
            onCheckedChange={(checked) =>
              table.toggleAllPageRowsSelected(!!checked)
            }
            aria-label="Select all"
          />
        );
      },
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(!!checked)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    }),
  };
}
