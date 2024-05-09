import { cn, shipmentStatusToReadable } from "@/lib/utils";
import { getShipmentCountByStatus } from "@/services/ShipmentRequestService";
import { QueryKeyWithParams } from "@/types";
import { ShipmentSearchForm } from "@/types/shipment";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Control, UseFormSetValue, UseFormWatch } from "react-hook-form";
import { InputField } from "../../common/fields/input";
import { Button } from "../../ui/button";

function FilterOptions({
  setValue,
  searchQuery,
}: {
  setValue: UseFormSetValue<ShipmentSearchForm>;
  searchQuery?: string;
}) {
  const [selectedStatus, setSelectedStatus] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["shipmentCountByStatus", searchQuery] as QueryKeyWithParams<
      "shipmentCountByStatus",
      [string]
    >,
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
        className={cn(
          "hover:bg-foreground hover:text-background flex w-full select-none flex-row items-center justify-between",
          selectedStatus === null ? "bg-foreground text-background" : "",
        )}
        onClick={() => {
          setValue("statusFilter", "");
          setSelectedStatus(null);
        }}
      >
        <div className="text-sm font-semibold">All Shipments</div>
        <div className="ml-2 text-sm font-semibold">{totalCount}</div>
      </Button>
      {sortedResults &&
        sortedResults.map(({ status, count }) => (
          <Button
            key={status}
            variant="outline"
            className={cn(
              "hover:bg-foreground hover:text-background flex w-full flex-row justify-between",
              selectedStatus === status && "bg-foreground text-background",
            )}
            onClick={() => {
              setValue("statusFilter", status);
              setSelectedStatus(status);
            }}
          >
            <div className="text-sm font-semibold">
              {shipmentStatusToReadable(status)}
            </div>
            <div className="text-sm font-semibold">{count}</div>
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
            <MagnifyingGlassIcon className="text-muted-foreground size-4" />
          }
        />
      </div>
      <p className="text-muted-foreground mb-4 text-sm font-semibold">
        Filter Shipments
      </p>
      <FilterOptions setValue={setValue} searchQuery={searchQuery} />
    </>
  );
}
