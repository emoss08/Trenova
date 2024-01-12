/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
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
import { InputField } from "@/components/common/fields/input";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { WorkerSortOptions } from "@/components/shipment-management/map-view/shipment-map-filter";
import { useDebounce } from "@/hooks/useDebounce";
import { useFleetCodes, useUsers } from "@/hooks/useQueries";
import { DEBOUNCE_DELAY } from "@/lib/constants";
import { getDispatchControl } from "@/services/OrganizationRequestService";
import { getWorkers } from "@/services/WorkerRequestService";
import { QueryKeys } from "@/types";
import { DispatchControl } from "@/types/dispatch";
import { Worker } from "@/types/worker";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { WorkerList, WorkerListSkeleton } from "./worker-list";

type WorkerSearchForm = {
  searchQuery: string;
  fleetFilter: string;
};

export function ShipmentMapAside() {
  const queryClient = useQueryClient();
  const { control, watch } = useForm<WorkerSearchForm>({
    defaultValues: {
      searchQuery: "",
      fleetFilter: "",
    },
  });

  const searchQuery = watch("searchQuery");

  const debouncedSearchQuery = useDebounce(searchQuery, DEBOUNCE_DELAY);

  const {
    selectFleetCodes,
    isLoading: isFleetCodesLoading,
    isError: isFleetCodeError,
  } = useFleetCodes();
  const {
    selectUsersData,
    isLoading: isUsersLoading,
    isError: isUserError,
  } = useUsers();
  const {
    data: dispatchControlData,
    isLoading: isDispatchControlDataLoading,
    isError: isDispatchControlError,
  } = useQuery({
    queryKey: ["dispatchControl"] as QueryKeys[],
    queryFn: async () => getDispatchControl(),
    initialData: (): DispatchControl[] | undefined =>
      queryClient.getQueryData(["dispatchControl"]),
    staleTime: Infinity,
  });
  const {
    data: workersData,
    isLoading: isWorkersLoading,
    isError: isWorkerError,
  } = useQuery({
    queryKey: ["workersList", debouncedSearchQuery],
    queryFn: async () => getWorkers(100, debouncedSearchQuery),
    initialData: (): Worker[] | undefined =>
      queryClient.getQueryData(["workersList", debouncedSearchQuery]),
    staleTime: Infinity,
  });

  // Store first element of dispatchControlData in variable
  const dispatchControlDataArray = dispatchControlData?.[0];

  const sortOptions = [
    {
      id: "status", // TODO: Change this once the HOS integration is an option.
      title: "Status",
      options: [
        {
          value: true,
          label: "Active",
        },
        {
          value: false,
          label: "Inactive",
        },
      ],
    },
    {
      id: "fleetCode",
      title: "Fleet",
      options: selectFleetCodes,
      loading: isFleetCodesLoading,
    },
    {
      id: "manager",
      title: "Manager",
      options: selectUsersData,
      loading: isUsersLoading,
    },

    {
      id: "endorsements",
      title: "Endorsements",
      options: [
        {
          value: "H",
          label: "Hazmat",
        },
        {
          value: "T",
          label: "Tanker",
        },
        {
          value: "X",
          label: "Tanker & Hazmat",
        },
        {
          value: "P",
          label: "Doubles",
        },
      ],
    },
  ];

  const isLoading =
    isFleetCodesLoading ||
    isUsersLoading ||
    isDispatchControlDataLoading ||
    isWorkersLoading;

  const isError =
    isFleetCodeError || isUserError || isDispatchControlError || isWorkerError;

  if (isError) {
    return (
      <aside className="w-96 items-center rounded-md border p-4 ring-accent-foreground/20">
        <div className="mt-52">
          <ErrorLoadingData />
        </div>
      </aside>
    );
  }

  return (
    <aside className="w-96 rounded-md border p-4 ring-accent-foreground/20">
      {isLoading ? (
        <WorkerListSkeleton />
      ) : (
        <>
          {/* Fixed search field at the top */}
          <InputField
            name="searchQuery"
            control={control}
            placeholder="Search Workers..."
            icon={
              <MagnifyingGlassIcon className="h-4 w-4 text-muted-foreground" />
            }
          />
          {/* Worker Sort Options */}
          <WorkerSortOptions sortOptions={sortOptions} />
          {/* Worker List */}
          <WorkerList
            dispatchControlData={dispatchControlDataArray as DispatchControl}
            workersData={workersData as Worker[]}
          />
        </>
      )}
    </aside>
  );
}
