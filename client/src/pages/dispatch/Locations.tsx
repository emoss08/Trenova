/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { DataTable } from "@/components/common/table/data-table";
import { DataTableColumnHeader } from "@/components/common/table/data-table-column-header";
import { StatusBadge } from "@/components/common/table/data-table-components";
import { DataTableColumnExpand } from "@/components/common/table/data-table-expand";
import { LocationTableSheet } from "@/components/location/location-table-dialog";
import { LocationTableEditSheet } from "@/components/location/location-table-edit-dialog";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { tableStatusChoices } from "@/lib/choices";
import { truncateText, upperFirst } from "@/lib/utils";
import { type Location } from "@/types/location";
import { type FilterConfig } from "@/types/tables";
import type { ColumnDef, Row } from "@tanstack/react-table";
import { Suspense, lazy } from "react";

const LocationChart = lazy(
  () => import("../../components/location/chart/table-chart"),
);

const renderSubComponent = ({ row }: { row: Row<Location> }) => {
  return (
    <Suspense fallback={null}>
      <LocationChart row={row} />
    </Suspense>
  );
};

function LocationColor({
  color,
  locationName,
  locationCategoryName,
}: {
  color: string;
  locationName: string;
  locationCategoryName?: string;
}) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="text-foreground flex items-center space-x-2 text-sm font-medium">
            <div
              className={"mx-2 size-2 rounded-xl"}
              style={{ backgroundColor: color }}
            />
            {locationName}
          </div>
        </TooltipTrigger>
        <TooltipContent align="start">
          {locationCategoryName && <p>{locationCategoryName}</p>}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

const columns: ColumnDef<Location>[] = [
  {
    id: "expander",
    footer: (props) => props.column.id,
    header: () => null,
    cell: ({ row }) => {
      return <DataTableColumnExpand row={row} />;
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
          <LocationColor
            color={row.original.locationColor}
            locationName={row.original.name}
            locationCategoryName={row.original.locationCategoryName as string}
          />
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
    accessorKey: "locationCategoryName",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Location Category" />
    ),
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

const filters: FilterConfig<Location>[] = [
  {
    columnName: "status",
    title: "Status",
    options: tableStatusChoices,
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
      tableFacetedFilters={filters}
      TableSheet={LocationTableSheet}
      TableEditSheet={LocationTableEditSheet}
      renderSubComponent={renderSubComponent}
      getRowCanExpand={() => true}
      addPermissionName="add_location"
    />
  );
}
