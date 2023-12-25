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

import { shipmentStatusToReadable } from "@/lib/utils";
import { getShipmentCountByStatus } from "@/services/ShipmentRequestService";
import { QueryKeys } from "@/types";
import { ShipmentSearchForm } from "@/types/order";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Control, UseFormSetValue, UseFormWatch } from "react-hook-form";
import { InputField } from "../common/fields/input";
import { Button } from "../ui/button";

function FilterOptions({
  setValue,
  searchQuery,
}: {
  setValue: UseFormSetValue<ShipmentSearchForm>;
  searchQuery?: string;
}) {
  const [selectedStatus, setSelectedStatus] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["shipmentCountByStatus", searchQuery] as QueryKeys[],
    queryFn: async () => getShipmentCountByStatus(searchQuery),
    staleTime: Infinity,
  });

  if (isLoading) {
    return <div>Loading...</div>;
  }

  const totalCount = (data && data.totalCount) || 0; // Fallback to 0 if totalCount is undefined

  // Define the sort order for each status
  const sortOrder = {
    New: 1,
    "In Progress": 2,
    Completed: 3,
    "On Hold": 4,
    Billed: 5,
    Voided: 6,
    Unknown: 7,
  };

  // Sort the results based on the defined order
  const sortedResults =
    data &&
    data.results.sort((a, b) => {
      return (
        sortOrder[shipmentStatusToReadable(a.status)] -
        sortOrder[shipmentStatusToReadable(b.status)]
      );
    });

  return (
    <div className="flex flex-col space-y-4">
      <Button
        variant="outline"
        className={`w-full hover:bg-foreground hover:text-background flex flex-row justify-between items-center select-none ${
          selectedStatus === null ? "bg-foreground text-background" : ""
        }`}
        onClick={() => {
          setValue("statusFilter", "");
          setSelectedStatus(null);
        }}
      >
        <div className="font-semibold text-sm">All Shipments</div>
        <div className="font-semibold text-sm ml-2">{totalCount}</div>
      </Button>
      {sortedResults &&
        sortedResults.map(({ status, count }) => (
          <Button
            key={status}
            variant="outline"
            className={`w-full hover:bg-foreground hover:text-background flex flex-row justify-between ${
              selectedStatus === status && "bg-foreground text-background"
            }`}
            onClick={() => {
              setValue("statusFilter", status);
              setSelectedStatus(status);
            }}
          >
            <div className="font-semibold text-sm">
              {shipmentStatusToReadable(status)}
            </div>
            <div className="font-semibold text-sm">{count}</div>
          </Button>
        ))}
    </div>
  );
}

export function ShipmentAsideMenus({
  control,
  setValue,
  watch,
}: {
  control: Control<ShipmentSearchForm>;
  setValue: UseFormSetValue<ShipmentSearchForm>;
  watch: UseFormWatch<ShipmentSearchForm>;
}) {
  const searchQuery = watch("searchQuery");

  return (
    <>
      <div className="mb-4">
        <InputField
          name="searchQuery"
          control={control}
          placeholder="Search Shipments..."
          icon={
            <MagnifyingGlassIcon className="h-4 w-4 text-muted-foreground" />
          }
        />
      </div>
      <p className="text-sm font-semibold text-muted-foreground mb-4">
        Filter Shipments
      </p>
      <FilterOptions setValue={setValue} searchQuery={searchQuery} />
    </>
  );
}
