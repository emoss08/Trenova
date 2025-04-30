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
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  shipmentSchema,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { type Shipment } from "@/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { FormProvider } from "react-hook-form";
import { useShipmentDetails } from "../queries/shipment";
import { ShipmentForm } from "./form/shipment-form";

export function ShipmentEditSheet({
  currentRecord,
}: EditTableSheetProps<Shipment>) {
  const { table, rowSelection, isLoading } = useDataTable();
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

  const form = useFormWithSave({
    resourceName: "Shipment",
    formOptions: {
      resolver: zodResolver(shipmentSchema),
      defaultValues: shipmentDetails,
      mode: "onChange",
    },
    mutationFn: async (values: ShipmentSchema) => {
      const response = await http.put<Shipment>(
        `/shipments/${currentRecord?.id}`,
        values,
      );
      return response.data;
    },
    onSuccess: () => {
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

      // * Reset the row selection
      table.resetRowSelection();

      // * Close the sheet
      setSearchParams({ modalType: null, entityId: null });

      // * If the page is a popout, close it
      if (isPopout) {
        closePopout();
      }
    },
  });

  const {
    reset,
    handleSubmit,
    onSubmit,
    formState: { isSubmitting, isSubmitSuccessful },
  } = form;

  // Update form values when currentRecord changes and is not loading
  useEffect(() => {
    if (
      !isLoading &&
      currentRecord &&
      currentRecord.id !== previousRecordIdRef.current
    ) {
      reset(currentRecord);
      previousRecordIdRef.current = currentRecord.id;
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

  // const {
  //   showWarning,
  //   handleClose: onClose,
  //   handleConfirmClose,
  //   handleCancelClose,
  // } = useUnsavedChanges({
  //   control: control,
  //   onClose: handleClose,
  // });

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    if (isSubmitSuccessful) {
      reset();
    }
  }, [isSubmitSuccessful, reset]);

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
              <SheetBody className="p-0">
                <ShipmentForm
                  open={!!selectedRowKey}
                  sheetRef={sheetRef}
                  selectedShipment={shipmentDetails}
                  isLoading={isDetailsLoading}
                  onBack={() =>
                    setSearchParams({ modalType: null, entityId: null })
                  }
                  isError={isDetailsError}
                />
              </SheetBody>
              <FormSaveDock position="right" />
            </Form>
          </FormProvider>
        </SheetContent>
      </Sheet>
    </>
  );
}
