import { ShipmentInfo } from "@/components/shipment-management/shipment-list";
import { getShipments } from "@/services/ShipmentRequestService";
import { Shipment, ShipmentSearchForm, ShipmentStatus } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { DragDropContext, Draggable, Droppable } from "react-beautiful-dnd";
import { useFormContext } from "react-hook-form";
import { ErrorLoadingData } from "../common/table/data-table-components";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "../ui/alert-dialog";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "../ui/pagination";
import { ScrollArea } from "../ui/scroll-area";
import { Skeleton } from "../ui/skeleton";
import { ShipmentToolbar } from "./shipment-advanced-filter";

const ITEMS_PER_PAGE = 10;

// Mock list of workers
const mockWorkers = [
  { id: "1", name: "John Doe" },
  { id: "2", name: "Jane Smith" },
  { id: "3", name: "Bob Johnson" },
  { id: "4", name: "Alice Brown" },
  { id: "5", name: "Charlie Davis" },
];

export function ShipmentListView({
  finalStatuses,
  progressStatuses,
}: {
  finalStatuses: ShipmentStatus[];
  progressStatuses: ShipmentStatus[];
}) {
  const { watch } = useFormContext<ShipmentSearchForm>();
  const { searchQuery, statusFilter } = watch();
  const [page, setPage] = useState(1);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedWorker, setSelectedWorker] = useState<
    (typeof mockWorkers)[0] | null
  >(null);
  const [selectedShipment, setSelectedShipment] = useState<any | null>(null);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["shipments", searchQuery, statusFilter, page],
    queryFn: () =>
      getShipments(
        searchQuery,
        statusFilter,
        (page - 1) * ITEMS_PER_PAGE,
        ITEMS_PER_PAGE,
      ),
    staleTime: Infinity,
  });

  useEffect(() => {
    setPage(1);
  }, [searchQuery, statusFilter]);

  const totalPages = data ? Math.ceil(data.count / ITEMS_PER_PAGE) : 0;

  const handlePageChange = (newPage: number) => {
    setPage(Math.max(1, Math.min(newPage, totalPages)));
  };

  const handleDragEnd = (result: any) => {
    if (!result.destination) return;

    const workerId = result.draggableId;
    const shipmentId = result.destination.droppableId;

    const worker = mockWorkers.find((w) => w.id === workerId);
    const shipment = data?.results.find((s: any) => s.id === shipmentId);

    if (worker && shipment) {
      setSelectedWorker(worker);
      setSelectedShipment(shipment);
      setIsModalOpen(true);
    }
  };

  const handleAssignWorker = () => {
    // Mock implementation - replace with actual API call later
    console.log(
      `Assigning worker ${selectedWorker?.name} to shipment ${selectedShipment?.proNumber}`,
    );
    setIsModalOpen(false);
    setSelectedWorker(null);
    setSelectedShipment(null);
  };

  const renderPaginationItems = () => {
    const items = [];
    const maxVisiblePages = 5;

    if (totalPages <= maxVisiblePages) {
      for (let i = 1; i <= totalPages; i++) {
        items.push(
          <PaginationItem key={i}>
            <PaginationLink
              onClick={() => handlePageChange(i)}
              isActive={page === i}
            >
              {i}
            </PaginationLink>
          </PaginationItem>,
        );
      }
    } else {
      items.push(
        <PaginationItem key={1}>
          <PaginationLink
            onClick={() => handlePageChange(1)}
            isActive={page === 1}
          >
            1
          </PaginationLink>
        </PaginationItem>,
      );

      if (page > 3) {
        items.push(<PaginationEllipsis key="ellipsis1" />);
      }

      const start = Math.max(2, page - 1);
      const end = Math.min(page + 1, totalPages - 1);

      for (let i = start; i <= end; i++) {
        items.push(
          <PaginationItem key={i}>
            <PaginationLink
              onClick={() => handlePageChange(i)}
              isActive={page === i}
            >
              {i}
            </PaginationLink>
          </PaginationItem>,
        );
      }

      if (page < totalPages - 2) {
        items.push(<PaginationEllipsis key="ellipsis2" />);
      }

      if (totalPages > 1) {
        items.push(
          <PaginationItem key={totalPages}>
            <PaginationLink
              onClick={() => handlePageChange(totalPages)}
              isActive={page === totalPages}
            >
              {totalPages}
            </PaginationLink>
          </PaginationItem>,
        );
      }
    }

    return items;
  };

  return (
    <>
      <DragDropContext onDragEnd={handleDragEnd}>
        <div className="flex w-full space-x-10">
          <div className="w-1/4">
            <h2 className="mb-4 text-lg font-semibold">Workers</h2>
            <Droppable droppableId="workersList">
              {(provided) => (
                <ul
                  {...provided.droppableProps}
                  ref={provided.innerRef}
                  className="space-y-2"
                >
                  {mockWorkers.map((worker, index) => (
                    <Draggable
                      key={worker.id}
                      draggableId={worker.id}
                      index={index}
                    >
                      {(provided) => (
                        <li
                          ref={provided.innerRef}
                          {...provided.draggableProps}
                          {...provided.dragHandleProps}
                          className="bg-background rounded p-2 shadow"
                        >
                          {worker.name}
                        </li>
                      )}
                    </Draggable>
                  ))}
                  {provided.placeholder}
                </ul>
              )}
            </Droppable>
          </div>
          <div className="w-3/4 space-y-4">
            {isLoading ? (
              <Skeleton className="h-[50vh] w-full" />
            ) : isError ? (
              <ErrorLoadingData />
            ) : (
              <>
                <ShipmentToolbar />
                <ScrollArea className="h-[77vh]">
                  <Droppable droppableId="shipmentList">
                    {(provided) => (
                      <div ref={provided.innerRef} {...provided.droppableProps}>
                        {data?.results && data.results.length > 0 ? (
                          data.results.map((shipment: any) => (
                            <Droppable
                              key={shipment.id}
                              droppableId={shipment.id}
                            >
                              {(provided, snapshot) => (
                                <div
                                  ref={provided.innerRef}
                                  {...provided.droppableProps}
                                  className={`mb-2 transition-colors duration-200 ${
                                    snapshot.isDraggingOver
                                      ? "bg-green-500/50"
                                      : ""
                                  }`}
                                >
                                  <ShipmentInfo
                                    shipment={shipment}
                                    finalStatuses={finalStatuses}
                                    progressStatuses={progressStatuses}
                                  />
                                  <div style={{ display: "none" }}>
                                    {provided.placeholder}
                                  </div>
                                </div>
                              )}
                            </Droppable>
                          ))
                        ) : (
                          <div className="text-muted-foreground py-8 text-center">
                            No shipments found for the given criteria.
                          </div>
                        )}
                        {provided.placeholder}
                      </div>
                    )}
                  </Droppable>
                </ScrollArea>
                {totalPages > 1 && (
                  <Pagination>
                    <PaginationContent>
                      <PaginationItem>
                        <PaginationPrevious
                          onClick={() => handlePageChange(page - 1)}
                          isActive={page !== 1}
                        />
                      </PaginationItem>
                      {renderPaginationItems()}
                      <PaginationItem>
                        <PaginationNext
                          onClick={() => handlePageChange(page + 1)}
                          isActive={page !== totalPages}
                        />
                      </PaginationItem>
                    </PaginationContent>
                  </Pagination>
                )}
              </>
            )}
          </div>
        </div>
      </DragDropContext>
      {isModalOpen && selectedWorker && selectedShipment && (
        <ShipmentConfirmDialog
          open={isModalOpen}
          onOpenChange={setIsModalOpen}
          handleAssignWorker={handleAssignWorker}
          selectedShipment={selectedShipment}
          selectedWorker={selectedWorker}
        />
      )}
    </>
  );
}

function ShipmentConfirmDialog({
  open,
  onOpenChange,
  handleAssignWorker,
  selectedWorker,
  selectedShipment,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  handleAssignWorker?: () => void;
  selectedWorker: any;
  selectedShipment: Shipment;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Assign Worker to Shipment</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to assign {selectedWorker?.name} to shipment{" "}
            {selectedShipment?.proNumber}?
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => onOpenChange(false)}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction onClick={handleAssignWorker}>
            Confirm
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
