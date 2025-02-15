import { Button } from "@/components/ui/button";
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
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { useResponsiveDimensions } from "@/hooks/use-responsive-dimensions";
import { http } from "@/lib/http-client";
import {
  shipmentSchema,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { Shipment } from "@/types/shipment";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { useShipmentDetails } from "../queries/shipment";
import { ShipmentForm } from "./sidebar/form/shipment-form";

export function ShipmentEditSheet({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<Shipment>) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const dimensions = useResponsiveDimensions(sheetRef, open);

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
    watch,
    setError,
    formState: { isDirty, isSubmitting, errors },
    handleSubmit,
    reset,
  } = form;

  console.debug("shipment values", watch());
  console.debug("errors", errors);

  useEffect(() => {
    if (shipmentDetails.data && !isDetailsLoading) {
      reset(shipmentDetails.data);
    }
  }, [shipmentDetails.data, isDetailsLoading, reset]);

  const { mutateAsync } = useMutation({
    mutationFn: async (values: ShipmentSchema) => {
      const response = await http.put(
        `/shipments/${currentRecord?.id}`,
        values,
      );
      return response.data;
    },
    onSuccess: () => {
      toast.success("Shipment updated successfully");
      onOpenChange(false);

      // Reset the form again to ensure the form is cleared with the new values
      reset();

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
  });

  const onSubmit = useCallback(
    async (values: ShipmentSchema) => {
      console.debug("onSubmit", values);
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        className="w-[1000px] sm:max-w-[500px] p-0"
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
                  onBack={() => onOpenChange(false)}
                />
              )}
            </SheetBody>
            <SheetFooter className="p-3">
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isSubmitting}>
                Save Changes
              </Button>
            </SheetFooter>
          </Form>
        </FormProvider>
      </SheetContent>
    </Sheet>
  );
}
