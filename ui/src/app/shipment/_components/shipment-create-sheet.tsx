import { FormSaveDock } from "@/components/form/form-save-dock";
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
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import {
  shipmentSchema,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { api } from "@/services/api";
import { TableSheetProps } from "@/types/data-table";
import { MoveStatus } from "@/types/move";
import { RatingMethod, ShipmentStatus } from "@/types/shipment";
import { StopStatus, StopType } from "@/types/stop";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ShipmentForm } from "./form/shipment-form";

export function ShipmentCreateSheet({ open, onOpenChange }: TableSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useForm({
    resolver: zodResolver(shipmentSchema),
    defaultValues: {
      status: ShipmentStatus.New,
      proNumber: undefined,
      ratingMethod: RatingMethod.FlatRate,
      ratingUnit: 1,
      actualDeliveryDate: undefined,
      actualShipDate: undefined,
      customerId: "",
      bol: "",
      serviceTypeId: "",
      shipmentTypeId: "",
      tractorTypeId: "",
      trailerTypeId: "",
      temperatureMax: undefined,
      temperatureMin: undefined,
      weight: undefined,
      pieces: undefined,
      freightChargeAmount: 0,
      otherChargeAmount: 0,
      totalChargeAmount: 0,
      customer: undefined,
      additionalCharges: [],
      commodities: [],
      moves: [
        {
          sequence: 0,
          loaded: true,
          tractorId: undefined,
          trailerId: undefined,
          assignment: undefined,
          distance: 0,
          status: MoveStatus.New,
          stops: [
            {
              sequence: 0,
              status: StopStatus.New,
              type: StopType.Pickup,
              locationId: "",
              addressLine: "",
              pieces: undefined,
              weight: undefined,
              actualArrival: undefined,
              actualDeparture: undefined,
              plannedArrival: 0,
              plannedDeparture: 0,
              shipmentMoveId: undefined,
              location: null,
            },
            {
              sequence: 1,
              status: StopStatus.New,
              type: StopType.Delivery,
              locationId: "",
              addressLine: "",
              pieces: undefined,
              weight: undefined,
              actualArrival: undefined,
              actualDeparture: undefined,
              plannedArrival: undefined,
              plannedDeparture: undefined,
              shipmentMoveId: undefined,
              location: null,
            },
          ],
        },
      ],
    },
    mode: "onChange",
  });

  const {
    setError,
    reset,
    handleSubmit,
    formState: { isSubmitting, isSubmitSuccessful },
  } = form;

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    reset();
  }, [isSubmitSuccessful, reset, onOpenChange]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: ShipmentSchema) => {
      return await api.shipments.create(values);
    },
    onSuccess: (newData) => {
      toast.success("Shipment Created", {
        description: `Shipment created successfully`,
      });
      handleClose();

      broadcastQueryInvalidation({
        queryKey: ["shipments"],
        options: { correlationId: `create-shipment-${Date.now()}` },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      queryClient.setQueryData(["shipments", newData.data.id], newData.data);
      reset();

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
    <Sheet open={open} onOpenChange={onOpenChange}>
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
              <ShipmentForm />
            </SheetBody>
            <FormSaveDock position="right" />
          </Form>
        </FormProvider>
      </SheetContent>
    </Sheet>
  );
}
