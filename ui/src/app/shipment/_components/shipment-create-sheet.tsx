import { FormSaveDock } from "@/components/form/form-save-dock";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button, FormSaveButton } from "@/components/ui/button";
import { Form } from "@/components/ui/form";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useUnsavedChanges } from "@/hooks/use-form";
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { useResponsiveDimensions } from "@/hooks/use-responsive-dimensions";
import { http } from "@/lib/http-client";
import {
  shipmentSchema,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { TableSheetProps } from "@/types/data-table";
import { MoveStatus } from "@/types/move";
import { RatingMethod, ShipmentStatus } from "@/types/shipment";
import { StopStatus, StopType } from "@/types/stop";
import { yupResolver } from "@hookform/resolvers/yup";
import { useCallback, useEffect, useRef } from "react";
import { FormProvider } from "react-hook-form";
import { ShipmentForm } from "./form/shipment-form";

export function ShipmentCreateSheet({ open, onOpenChange }: TableSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const dimensions = useResponsiveDimensions(sheetRef, open);
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useFormWithSave({
    resourceName: "Shipment",
    formOptions: {
      resolver: yupResolver(shipmentSchema),
      defaultValues: {
        status: ShipmentStatus.New,
        proNumber: undefined,
        ratingMethod: RatingMethod.FlatRate,
        ratingUnit: 1,
        moves: [
          {
            sequence: 0,
            loaded: true,
            status: MoveStatus.New,
            stops: [
              {
                sequence: 0,
                status: StopStatus.New,
                type: StopType.Pickup,
              },
              {
                sequence: 1,
                status: StopStatus.New,
                type: StopType.Delivery,
              },
            ],
          },
        ],
      },
      mode: "onChange",
    },
    mutationFn: async (values: ShipmentSchema) => {
      const response = await http.post(`/shipments/`, values);
      return response.data;
    },
    onSuccess: () => {
      onOpenChange(false);

      broadcastQueryInvalidation({
        queryKey: ["shipment", "shipment-list", "stop", "assignment"],
        options: {
          correlationId: `create-shipment-${Date.now()}`,
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
    watch,
    formState: { isDirty, isSubmitting, isSubmitSuccessful, errors },
  } = form;

  console.info("Shipment Form Values", watch());
  console.info("Shipment Form Errors", errors);

  const handleClose = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  const {
    showWarning,
    handleClose: onClose,
    handleConfirmClose,
    handleCancelClose,
  } = useUnsavedChanges({
    isDirty,
    onClose: handleClose,
  });

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    reset();
  }, [isSubmitSuccessful, reset, onOpenChange]);

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
      <Sheet open={open} onOpenChange={onClose}>
        <SheetContent
          className="w-[500px] sm:max-w-[540px] p-0"
          withClose={false}
          ref={sheetRef}
        >
          <VisuallyHidden>
            <SheetHeader>
              <SheetTitle>Shipment Details</SheetTitle>
            </SheetHeader>
            <SheetDescription>Test</SheetDescription>
          </VisuallyHidden>

          <FormProvider {...form}>
            <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
              <SheetBody className="p-0">
                <ShipmentForm
                  dimensions={dimensions}
                  onBack={onClose}
                  isCreate={true}
                />
              </SheetBody>
              <SheetFooter className="p-3">
                <Button type="button" variant="outline" onClick={onClose}>
                  Cancel
                </Button>
                <FormSaveButton
                  isPopout={isPopout}
                  isSubmitting={isSubmitting}
                  title="Shipment"
                />
              </SheetFooter>
              <FormSaveDock
                isDirty={isDirty}
                isSubmitting={isSubmitting}
                position="right"
              />
            </Form>
          </FormProvider>
        </SheetContent>
      </Sheet>

      {showWarning && (
        <AlertDialog open={showWarning} onOpenChange={handleCancelClose}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
              <AlertDialogDescription>
                You have unsaved changes. Are you sure you want to close this
                form? All changes will be lost.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={handleCancelClose}>
                Continue Editing
              </AlertDialogCancel>
              <AlertDialogAction onClick={handleConfirmClose}>
                Discard Changes
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </>
  );
}
