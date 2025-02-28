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
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useUnsavedChanges } from "@/hooks/use-form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { useResponsiveDimensions } from "@/hooks/use-responsive-dimensions";
import { http } from "@/lib/http-client";
import {
  shipmentSchema,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { type Shipment } from "@/types/shipment";
import { yupResolver } from "@hookform/resolvers/yup";
import { useCallback, useEffect, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { useShipmentDetails } from "../queries/shipment";
import { ShipmentForm } from "./form/shipment-form";

export function ShipmentEditSheet({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<Shipment>) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const dimensions = useResponsiveDimensions(sheetRef, open);
  const { isPopout, closePopout } = usePopoutWindow();

  const shipmentDetails = useShipmentDetails({
    shipmentId: currentRecord?.id ?? "",
    enabled: !!currentRecord?.id,
  });

  const isDetailsLoading = shipmentDetails.isLoading;

  const form = useForm<ShipmentSchema>({
    resolver: yupResolver(shipmentSchema),
    defaultValues: {},
  });

  const {
    setError,
    formState: { isDirty, isSubmitting, isSubmitSuccessful, errors },
    handleSubmit,
    reset,
    watch,
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

  useEffect(() => {
    if (shipmentDetails.data && !isDetailsLoading) {
      reset(shipmentDetails.data);
    }
  }, [shipmentDetails.data, isDetailsLoading, reset]);

  const { mutateAsync } = useApiMutation<
    Shipment, // The response data type
    ShipmentSchema, // The variables type
    unknown, // The context type
    ShipmentSchema // The form values type for error handling
  >({
    mutationFn: async (values: ShipmentSchema) => {
      const response = await http.put<Shipment>(
        `/shipments/${currentRecord?.id}`,
        values,
      );
      return response.data;
    },
    onSuccess: () => {
      toast.success("Changes have been saved", {
        description: "Shipment updated successfully",
      });
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
    // Pass in the form's setError function
    setFormError: setError,
    // Provide a resource name for better error logging
    resourceName: "Shipment",
    // You can still add custom onSettled logic
    onSettled: () => {
      if (isPopout) {
        closePopout();
      }
    },
  });

  const onSubmit = useCallback(
    async (values: ShipmentSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

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
                {shipmentDetails.data && (
                  <ShipmentForm
                    dimensions={dimensions}
                    selectedShipment={shipmentDetails.data}
                    isLoading={isDetailsLoading}
                    onBack={onClose}
                    isCreate={false}
                  />
                )}
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
