import { useDataTable } from "@/components/data-table/data-table-provider";
import { FormSaveDock } from "@/components/form";
import { Form } from "@/components/ui/form";
import { ScrollArea, ScrollAreaShadow } from "@/components/ui/scroll-area";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { shipmentSchema, ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { api } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { lazy, useCallback, useEffect, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { useShipmentDetails } from "../../queries/shipment";

const ShipmentBillingDetails = lazy(
  () => import("./billing-details/shipment-billing-details"),
);
const ShipmentGeneralInformation = lazy(
  () => import("./shipment-general-information"),
);
const ShipmentCommodityDetails = lazy(
  () => import("./commodity/commodity-details"),
);
const ShipmentMovesDetails = lazy(() => import("./move/move-details"));
const ShipmentServiceDetails = lazy(
  () => import("./service-details/shipment-service-details"),
);

export function ShipmentGeneralInfoForm({
  currentRecord,
}: {
  currentRecord?: ShipmentSchema | null;
}) {
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const previousRecordIdRef = useRef<string | number | null>(null);
  const initialLoadRef = useRef(false);
  const { data: shipmentDetails, isLoading: isDetailsLoading } =
    useShipmentDetails({
      shipmentId: currentRecord?.id ?? "",
      enabled: !!currentRecord?.id && searchParams.modalType === "edit",
    });
  const queryClient = useQueryClient();
  const { table, isLoading } = useDataTable();
  const { isPopout, closePopout } = usePopoutWindow();

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
      const response = await api.shipments.update(currentRecord?.id, values);
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
  }, [isSubmitting, handleSubmit, onSubmit]);

  return (
    <ScrollArea className="flex flex-col overflow-y-auto px-4 max-h-[calc(100vh-12rem)]">
      <FormProvider {...form}>
        <Form className="space-y-0 p-0 pb-16" onSubmit={handleSubmit(onSubmit)}>
          <ShipmentServiceDetails />
          <ShipmentBillingDetails />
          <ShipmentGeneralInformation />
          <ShipmentCommodityDetails />
          <ShipmentMovesDetails />
          <FormSaveDock position="right" className="pb-2" />
        </Form>
      </FormProvider>
      <ScrollAreaShadow />
    </ScrollArea>
  );
}
