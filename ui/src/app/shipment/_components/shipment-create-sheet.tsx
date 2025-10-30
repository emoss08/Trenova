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
import { MoveStatus } from "@/lib/schemas/move-schema";
import {
  RatingMethod,
  shipmentSchema,
  ShipmentStatus,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { StopStatus, StopType } from "@/lib/schemas/stop-schema";
import { api } from "@/services/api";
import { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ShipmentCreateForm } from "./form/shipment-form";

export function ShipmentCreateSheet({ open, onOpenChange }: TableSheetProps) {
  const queryClient = useQueryClient();
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useForm({
    resolver: zodResolver(shipmentSchema),
    defaultValues: {
      status: ShipmentStatus.enum.New,
      proNumber: undefined,
      ratingMethod: RatingMethod.enum.FlatRate,
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
      enteredById: undefined,
      comments: [],
      formulaTemplateId: undefined,
      additionalCharges: [],
      commodities: [],
      moves: [
        {
          sequence: 0,
          loaded: true,
          assignment: undefined,
          distance: 0,
          status: MoveStatus.enum.New,
          stops: [
            {
              sequence: 0,
              status: StopStatus.enum.New,
              type: StopType.enum.Pickup,
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
              status: StopStatus.enum.New,
              type: StopType.enum.Delivery,
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

  useEffect(() => {
    reset();
  }, [isSubmitSuccessful, reset]);

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

      queryClient.invalidateQueries({
        queryKey: ["shipment-list"],
      });

      queryClient.setQueryData(["shipments", newData.data.id], newData.data);

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
    <Sheet
      open={open}
      onOpenChange={(newOpen) => {
        onOpenChange(newOpen);
      }}
    >
      <SheetContent
        className="w-[500px] sm:max-w-[540px] p-0"
        withClose={false}
      >
        <VisuallyHidden>
          <SheetHeader>
            <SheetTitle>Shipment Details</SheetTitle>
          </SheetHeader>
          <SheetDescription>
            Create a new shipment by filling out the form below.
          </SheetDescription>
        </VisuallyHidden>
        <FormProvider {...form}>
          <Form
            className="space-y-0 p-0"
            onSubmit={form.handleSubmit(onSubmit)}
          >
            <SheetBody className="p-0">
              <ShipmentCreateForm />
            </SheetBody>
            <FormSaveDock position="right" />
          </Form>
        </FormProvider>
      </SheetContent>
    </Sheet>
  );
}
