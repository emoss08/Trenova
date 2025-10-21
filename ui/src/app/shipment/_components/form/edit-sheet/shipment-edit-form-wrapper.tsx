"use no memo";
import { useShipmentDetails } from "@/app/shipment/queries/shipment";
import { useDataTable } from "@/components/data-table/data-table-provider";
import { FormSaveDock } from "@/components/form";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { shipmentSchema, ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ShipmentDetailsSkeleton } from "../shipment-details-skeleton";
import { ShipmentFormWrapper } from "../shipment-form-wrapper";
import { ShipmentGeneralInfoForm } from "../shipment-general-info-form";

export function ShipmentEditFormWrapper({
  shipmentId,
}: {
  shipmentId: ShipmentSchema["id"];
}) {
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const queryClient = useQueryClient();
  const { table } = useDataTable();
  const { isPopout, closePopout } = usePopoutWindow();

  const { data: shipmentDetails, isLoading: isDetailsLoading } =
    useShipmentDetails({
      shipmentId: shipmentId ?? "",
      enabled: !!shipmentId && searchParams.modalType === "edit",
    });

  const form = useForm<ShipmentSchema>({
    resolver: zodResolver(shipmentSchema) as any,
    resetOptions: {
      keepDirtyValues: true,
    },
    values: shipmentDetails,
  });

  const {
    setError,
    reset,
    handleSubmit,
    formState: { isSubmitting, errors },
  } = form;

  console.info("Shipment Form Errors:", errors);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: ShipmentSchema) => {
      const response = await api.shipments.update(shipmentId, values);
      return response.data;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["shipment", shipmentId],
      });

      const previousShipment = queryClient.getQueryData([
        "shipment",
        shipmentId,
      ]);

      queryClient.setQueryData(["shipment", shipmentId], newValues);

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

      table.resetRowSelection();
      setSearchParams({ modalType: null, entityId: null });

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
  }, [isSubmitting, handleSubmit, onSubmit]);

  return (
    <FormProvider {...form}>
      <ShipmentFormWrapper onSubmit={onSubmit}>
        {isDetailsLoading ? (
          <ShipmentDetailsSkeleton />
        ) : (
          <>
            <ShipmentGeneralInfoForm className="max-h-[calc(100vh-10.5rem)]" />
            <FormSaveDock position="right" />
          </>
        )}
      </ShipmentFormWrapper>
    </FormProvider>
  );
}
