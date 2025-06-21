/* eslint-disable react-hooks/exhaustive-deps */
import { useDataTable } from "@/components/data-table/data-table-provider";
import { FormSaveDock } from "@/components/form";
import { Form } from "@/components/ui/form";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  shipmentSchema,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { type Shipment } from "@/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { memo, useCallback, useEffect, useMemo, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { useShipmentDetails } from "../queries/shipment";
import { ShipmentForm } from "./form/shipment-form";

export function ShipmentEditSheet({
  currentRecord,
}: EditTableSheetProps<ShipmentSchema>) {
  const { table, rowSelection, isLoading } = useDataTable();
  const queryClient = useQueryClient();
  const sheetRef = useRef<HTMLDivElement>(null);
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { isPopout, closePopout } = usePopoutWindow();
  const initialLoadRef = useRef(false);

  const previousRecordIdRef = useRef<string | number | null>(null);
  const selectedRowKey = Object.keys(rowSelection)[0];

  const selectedRow = useMemo(() => {
    if (isLoading && !selectedRowKey) return;
    return table
      .getCoreRowModel()
      .flatRows.find((row) => row.id === selectedRowKey);
  }, [selectedRowKey, isLoading]);

  const index = table
    .getCoreRowModel()
    .flatRows.findIndex((row) => row.id === selectedRow?.id);

  const nextId = useMemo(
    () => table.getCoreRowModel().flatRows[index + 1]?.id,
    [index, isLoading],
  );

  const prevId = useMemo(
    () => table.getCoreRowModel().flatRows[index - 1]?.id,
    [index, isLoading],
  );

  const onPrev = useCallback(() => {
    if (prevId) table.setRowSelection({ [prevId]: true });
  }, [prevId, isLoading]);

  const onNext = useCallback(() => {
    if (nextId) table.setRowSelection({ [nextId]: true });
  }, [nextId, isLoading, table]);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (!selectedRowKey) return;

      // REMINDER: prevent dropdown navigation inside of sheet to change row selection
      const activeElement = document.activeElement;
      const isMenuActive = activeElement?.closest('[role="menu"]');

      if (isMenuActive) return;

      if (e.key === "ArrowUp") {
        e.preventDefault();
        onPrev();
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        onNext();
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [selectedRowKey, onNext, onPrev]);

  const {
    data: shipmentDetails,
    isLoading: isDetailsLoading,
    isError: isDetailsError,
  } = useShipmentDetails({
    shipmentId: currentRecord?.id ?? "",
    enabled: !!currentRecord?.id && searchParams.modalType === "edit", // * Only fetch data if the sheet is open
  });

  const form = useForm({
    resolver: zodResolver(shipmentSchema),
    defaultValues: shipmentDetails,
    mode: "onChange",
  });

  const {
    setError,
    reset,
    handleSubmit,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: ShipmentSchema) => {
      const response = await http.put<Shipment>(
        `/shipments/${currentRecord?.id}`,
        values,
      );
      return response.data;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["shipment", currentRecord?.id],
      });

      // * snapshot of the previous value
      const previousShipment = queryClient.getQueryData([
        "shipment",
        currentRecord?.id,
      ]);

      // * optimistically update to the new value
      queryClient.setQueryData(["shipment", currentRecord?.id], newValues);

      return { previousShipment, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Changes have been saved", {
        description: `Shipment updated successfully`,
      });

      broadcastQueryInvalidation({
        queryKey: ["shipment", "shipment-list", "stop", "assignment"],
        options: {
          correlationId: `update-shipment-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      reset(newValues);

      // * Reset the row selection
      table.resetRowSelection();

      // * Close the sheet
      setSearchParams({ modalType: null, entityId: null });

      // * If the page is a popout, close it
      if (isPopout) {
        closePopout();
      }
    },
    setFormError: setError,
    resourceName: "Shipment",
  });

  const onSubmit = useCallback(
    async (values: ShipmentSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  // Update form values when currentRecord changes and is not loading
  useEffect(() => {
    if (
      !isLoading &&
      currentRecord &&
      currentRecord.id !== previousRecordIdRef.current
    ) {
      reset(currentRecord);
      previousRecordIdRef.current = currentRecord.id ?? null;
    }
  }, [currentRecord, isLoading, reset]);

  useEffect(() => {
    if (shipmentDetails && !isDetailsLoading && !initialLoadRef.current) {
      reset(shipmentDetails, {
        keepDirty: false, // Don't keep dirty state
        keepValues: false, // Don't keep current values
      });
      initialLoadRef.current = true;
    }
  }, [shipmentDetails, isDetailsLoading, reset]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, isSubmitting, handleSubmit, onSubmit]);

  return (
    <>
      <Sheet
        open={!!selectedRowKey}
        onOpenChange={(open) => {
          if (!open) {
            const el = selectedRowKey
              ? document.getElementById(selectedRowKey)
              : null;
            table.resetRowSelection();

            setTimeout(() => el?.focus(), 0);
          }
        }}
      >
        <SheetContent
          className="w-[500px] sm:max-w-[540px] p-0"
          withClose={false}
          ref={sheetRef}
        >
          <VisuallyHidden>
            <SheetHeader>
              <SheetTitle>Shipment Details</SheetTitle>
            </SheetHeader>
            <SheetDescription>{shipmentDetails?.bol}</SheetDescription>
          </VisuallyHidden>

          <FormProvider {...form}>
            <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
              <MemoizedShipmentSheetBody
                selectedShipment={shipmentDetails}
                isLoading={isDetailsLoading}
                isError={isDetailsError}
              />
              <FormSaveDock position="right" />
            </Form>
          </FormProvider>
        </SheetContent>
      </Sheet>
    </>
  );
}

function ShipmentSheetBody({
  isLoading,
  isError,
  selectedShipment,
}: {
  isLoading: boolean;
  isError: boolean;
  selectedShipment?: ShipmentSchema | null;
}) {
  return (
    <SheetBody className="p-0">
      <ShipmentForm
        selectedShipment={selectedShipment}
        isLoading={isLoading}
        isError={isError}
      />
    </SheetBody>
  );
}

const MemoizedShipmentSheetBody = memo(ShipmentSheetBody, (prev, next) => {
  // * we only check if the selectedShipment is the same, rest is useless
  return prev.selectedShipment === next.selectedShipment;
}) as typeof ShipmentSheetBody;
