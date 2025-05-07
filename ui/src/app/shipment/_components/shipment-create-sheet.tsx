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
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  shipmentSchema,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { TableSheetProps } from "@/types/data-table";
import { MoveStatus } from "@/types/move";
import { RatingMethod, ShipmentStatus } from "@/types/shipment";
import { StopStatus, StopType } from "@/types/stop";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useRef } from "react";
import { FormProvider } from "react-hook-form";
import { ShipmentForm } from "./form/shipment-form";

export function ShipmentCreateSheet({ open, onOpenChange }: TableSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useFormWithSave({
    resourceName: "Shipment",
    formOptions: {
      resolver: zodResolver(shipmentSchema),
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
    formState: { isSubmitting, isSubmitSuccessful },
  } = form;

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
              <ShipmentForm open={open} sheetRef={sheetRef} />
            </SheetBody>
            <FormSaveDock />
          </Form>
        </FormProvider>
      </SheetContent>
    </Sheet>
  );
}
