import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { EquipmentStatusBadge } from "@/components/status-badge";
import { InternalLink } from "@/components/ui/link";
import { type Tractor } from "@/types/tractor";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<Tractor>[] {
  const columnHelper = createColumnHelper<Tractor>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    commonColumns.selection,
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      cell: ({ row }) => {
        const status = row.original.status;
        return <EquipmentStatusBadge status={status} />;
      },
    },
    {
      accessorKey: "code",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Code" />
      ),
    },
    {
      id: "equipmentType",
      accessorKey: "equipmentType",
      header: "Equipment Type",
      cell: ({ row }) => {
        const equipType = row.original.equipmentType;
        const isEquipType = !!equipType;

        return isEquipType ? (
          <DataTableColorColumn
            color={equipType?.color}
            text={equipType?.code ?? ""}
          />
        ) : (
          <p>No equipment type</p>
        );
      },
    },
    {
      id: "assignedWorkers",
      header: "Assigned Workers",
      cell: ({ row }) => {
        const { primaryWorker, secondaryWorker } = row.original;

        const isPrimaryWorker = !!primaryWorker;
        const isSecondaryWorker = !!secondaryWorker;

        return isPrimaryWorker ? (
          <div className="flex flex-col gap-0.5">
            <p>
              <InternalLink to="/dispatch/configurations/workers">
                {primaryWorker?.firstName} {primaryWorker?.lastName}
              </InternalLink>
            </p>
            {isSecondaryWorker && (
              <div className="flex items-center gap-1 text-muted-foreground text-2xs">
                <p>Co-Driver:</p>
                <InternalLink
                  to="/dispatch/configurations/workers"
                  className="text-2xs text-muted-foreground"
                >
                  {secondaryWorker?.firstName} {secondaryWorker?.lastName}
                </InternalLink>
              </div>
            )}
          </div>
        ) : (
          <p>No assigned workers</p>
        );
      },
    },
  ];
}
