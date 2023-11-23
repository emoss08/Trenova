/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { DataTable, StatusBadge } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { ColumnDef, Row } from "@tanstack/react-table";
import { truncateText, upperFirst } from "@/lib/utils";
import { FleetCodeDialog } from "@/components/fleet-codes/fleet-code-table-dialog";
import { FleetCodeEditDialog } from "@/components/fleet-codes/fleet-code-table-edit-dialog";
import { Location } from "@/types/location";
import { ChevronDownIcon, ChevronRightIcon } from "@radix-ui/react-icons";
import { LocationChart } from "@/components/location/table-chart";

const renderSubComponent = ({ row }: { row: Row<Location> }) => {
  return <LocationChart row={row} />;
};

const columns: ColumnDef<Location>[] = [
  {
    id: "expander",
    footer: (props) => props.column.id,
    header: () => null,
    cell: ({ row }) => {
      return row.getCanExpand() ? (
        <button
          {...{
            onClick: row.getToggleExpandedHandler(),
            style: { cursor: "pointer" },
          }}
        >
          {row.getIsExpanded() ? (
            <ChevronDownIcon style={{ width: "1.2em", height: "1.2em" }} />
          ) : (
            <ChevronRightIcon style={{ width: "1.2em", height: "1.2em" }} />
          )}
        </button>
      ) : (
        "🔵"
      );
    },
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
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => {
      if (row.original.locationColor) {
        return (
          <div className="flex items-center space-x-2 text-sm font-mediumtext-gray-900 dark:text-gray-100">
            <div
              className={"h-5 w-5 rounded-xl mx-2"}
              style={{ backgroundColor: row.original.locationColor }}
            />
            {row.original.name}
          </div>
        );
      } else {
        return row.original.name;
      }
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "location",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Location" />
    ),
    cell: ({ row }) =>
      `${upperFirst(row.original.city)}, ${row.original.state}`,
  },
  {
    accessorKey: "pickupCount",
    header: "Total Pickups",
  },
  {
    accessorKey: "waitTimeAvg",
    header: "Avg. Wait Time (mins)",
    cell: ({ row }) => {
      return row.original.waitTimeAvg && row.original.waitTimeAvg.toFixed(1);
    },
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => truncateText(row.original.description as string, 50),
  },
];

export default function Locations() {
  return (
    <DataTable
      queryKey="locations-table-data"
      columns={columns}
      link="/locations/"
      name="Locations"
      exportModelName="Location"
      filterColumn="name"
      TableSheet={FleetCodeDialog}
      TableEditSheet={FleetCodeEditDialog}
      renderSubComponent={renderSubComponent}
      getRowCanExpand={() => true}
    />
  );
}
