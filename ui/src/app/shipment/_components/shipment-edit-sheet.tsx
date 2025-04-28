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
import { useCallback, useEffect, useRef } from "react";
import { FormProvider } from "react-hook-form";
import { useShipmentDetails } from "../queries/shipment";
import { ShipmentForm } from "./form/shipment-form";

export function ShipmentEditSheet({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<Shipment>) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const { isPopout, closePopout } = usePopoutWindow();
  const initialLoadRef = useRef(false);

  const {
    data: shipmentDetails,
    isLoading: isDetailsLoading,
    isError: isDetailsError,
  } = useShipmentDetails({
    shipmentId: currentRecord?.id ?? "",
    enabled: !!currentRecord?.id && open, // * Only fetch data if the sheet is open
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
      onOpenChange(false);

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
    },
    onSettled: () => {
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

  const handleClose = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

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
        open &&
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
      <Sheet open={open} onOpenChange={handleClose}>
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
                  open={open}
                  sheetRef={sheetRef}
                  selectedShipment={shipmentDetails}
                  isLoading={isDetailsLoading}
                  onBack={handleClose}
                  isError={isDetailsError}
                />
              </SheetBody>
              <FormSaveDock position="right" />
            </Form>
          </FormProvider>
        </SheetContent>
      </Sheet>
      {/* {showWarning && (
        <AlertDialog open={showWarning} onOpenChange={onClose}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
              <AlertDialogDescription>
                You have unsaved changes. Are you sure you want to close this
                form? All changes will be lost.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={onClose}>
                Continue Editing
              </AlertDialogCancel>
              <AlertDialogAction onClick={handleConfirmClose}>
                Discard Changes
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )} */}
    </>
  );
}
