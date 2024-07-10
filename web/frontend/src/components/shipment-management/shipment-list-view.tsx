import { getTractors } from "@/services/EquipmentRequestService";
import {
  assignTractorToShipment,
  getShipments,
} from "@/services/ShipmentRequestService";
import type { Tractor, TractorFilterForm } from "@/types/equipment";
import type {
  Shipment,
  ShipmentSearchForm,
  ShipmentStatus,
} from "@/types/shipment";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { DragDropContext } from "react-beautiful-dnd";
import { useForm, useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { Pagination } from "../common/pagination";
import { ErrorLoadingData } from "../common/table/data-table-components";
import { ScrollArea } from "../ui/scroll-area";
import { Skeleton } from "../ui/skeleton";
import { ShipmentConfirmDialog } from "./assign-dialog";
import { ShipmentToolbar } from "./shipment-advanced-filter";
import { ShipmentList } from "./shipment-list";
import { TractorList } from "./tractor-list";

const ITEMS_PER_PAGE = 10;

export function ShipmentListView({
  finalStatuses,
  progressStatuses,
}: {
  finalStatuses: ShipmentStatus[];
  progressStatuses: ShipmentStatus[];
}) {
  const { watch } = useFormContext<ShipmentSearchForm>();
  const { searchQuery, statusFilter } = watch();
  const [shipmentPage, setShipmentPage] = useState(1);
  const [tractorPage, setTractorPage] = useState(1);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedTractor, setSelectedTractor] = useState<Tractor | null>(null);
  const [selectedShipment, setSelectedShipment] = useState<Shipment | null>(
    null,
  );

  const tractorFilterForm = useForm<TractorFilterForm>({
    defaultValues: {
      searchQuery: "",
      status: "Available",
      fleetCodeId: "",
      expandEquipDetails: false,
      expandWorkerDetails: false,
    },
  });

  const { watch: watchTractor } = tractorFilterForm;
  const {
    searchQuery: tractorSearchQuery,
    status: tractorStatusFilter,
    fleetCodeId,
  } = watchTractor();

  const {
    data: tractorData,
    isLoading: isTractorLoading,
    isError: isTractorError,
  } = useQuery({
    queryKey: [
      "tractors",
      tractorSearchQuery,
      tractorStatusFilter,
      fleetCodeId,
      tractorPage,
    ],
    queryFn: () =>
      getTractors(
        tractorStatusFilter,
        (tractorPage - 1) * ITEMS_PER_PAGE,
        ITEMS_PER_PAGE,
        fleetCodeId,
      ),
    staleTime: Infinity,
  });

  const {
    data: shipmentData,
    isLoading: isShipmentLoading,
    isError: isShipmentError,
  } = useQuery({
    queryKey: ["shipments", searchQuery, statusFilter, shipmentPage],
    queryFn: () =>
      getShipments(
        searchQuery,
        statusFilter,
        (shipmentPage - 1) * ITEMS_PER_PAGE,
        ITEMS_PER_PAGE,
      ),
    staleTime: Infinity,
  });

  useEffect(() => {
    setShipmentPage(1);
  }, [searchQuery, statusFilter]);

  useEffect(() => {
    setTractorPage(1);
  }, [tractorSearchQuery, tractorStatusFilter, fleetCodeId]);

  const handleDragEnd = (result: any) => {
    if (!result.destination) return;

    const tractorId = result.draggableId;
    const shipmentId = result.destination.droppableId;

    const tractor = tractorData?.results.find((t) => t.id === tractorId);
    const shipment = shipmentData?.results.find((s) => s.id === shipmentId);

    if (tractor && shipment) {
      setSelectedTractor(tractor);
      setSelectedShipment(shipment);
      setIsModalOpen(true);
    }
  };

  type AssignWorkerPayload = {
    tractorId: string;
    shipmentId: string;
  };

  const assignMutaton = useMutation({
    mutationFn: (payload: AssignWorkerPayload) => {
      return assignTractorToShipment(payload);
    },
  });

  const handleAssignTractor = () => {
    if (!selectedTractor || !selectedShipment) return;

    toast.promise(
      assignMutaton.mutateAsync({
        tractorId: selectedTractor.id,
        shipmentId: selectedShipment.id,
      }),
      {
        loading: "Assigning tractor to shipment...",
        success: (data) => data.message || "Tractor assigned to shipment.",
        error: (data) => {
          const resp = data.response.data;
          return resp.message || "Failed to assign tractor to shipment.";
        },
      },
    );

    console.log(
      `Assigning tractor ${selectedTractor?.code} to shipment ${selectedShipment?.proNumber}`,
    );
    setIsModalOpen(false);
    setSelectedTractor(null);
    setSelectedShipment(null);
  };

  return (
    <>
      <DragDropContext onDragEnd={handleDragEnd}>
        <div className="flex w-full space-x-10">
          <div className="w-1/4">
            <h2 className="mb-4 text-lg font-semibold">Tractors</h2>
            {isTractorLoading ? (
              <Skeleton className="h-[50vh] w-full" />
            ) : isTractorError ? (
              <ErrorLoadingData />
            ) : (
              <>
                <TractorList
                  tractors={tractorData?.results || []}
                  form={tractorFilterForm}
                />
                <Pagination
                  currentPage={tractorPage}
                  totalPages={Math.ceil(
                    (tractorData?.count || 0) / ITEMS_PER_PAGE,
                  )}
                  onPageChange={setTractorPage}
                />
              </>
            )}
          </div>
          <div className="w-3/4 space-y-4">
            <ShipmentToolbar />
            {isShipmentLoading ? (
              <Skeleton className="h-[50vh] w-full" />
            ) : isShipmentError ? (
              <ErrorLoadingData />
            ) : (
              <>
                <ScrollArea className="h-[77vh]">
                  <ShipmentList
                    shipments={shipmentData?.results || []}
                    finalStatuses={finalStatuses}
                    progressStatuses={progressStatuses}
                  />
                </ScrollArea>
                <Pagination
                  currentPage={shipmentPage}
                  totalPages={Math.ceil(
                    (shipmentData?.count || 0) / ITEMS_PER_PAGE,
                  )}
                  onPageChange={setShipmentPage}
                />
              </>
            )}
          </div>
        </div>
      </DragDropContext>
      {isModalOpen && selectedTractor && selectedShipment && (
        <ShipmentConfirmDialog
          open={isModalOpen}
          onOpenChange={setIsModalOpen}
          handleAssignTractor={handleAssignTractor}
          selectedShipment={selectedShipment}
          selectedTractor={selectedTractor}
        />
      )}
    </>
  );
}
