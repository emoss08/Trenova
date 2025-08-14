/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
import { useCallback, useEffect, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { useShipmentDetails } from "../../queries/shipment";
import { ShipmentFormWrapper } from "./shipment-form-wrapper";
import { ShipmentGeneralInfoForm } from "./shipment-general-info-form";

export function ShipmentEditFormWrapper({
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

      const previousShipment = queryClient.getQueryData([
        "shipment",
        currentRecord?.id,
      ]);

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
        keepDirty: false,
        keepValues: false,
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
    <FormProvider {...form}>
      <ShipmentFormWrapper onSubmit={onSubmit}>
        <ShipmentGeneralInfoForm className="max-h-[calc(100vh-11rem)]" />
        <FormSaveDock position="right" />
      </ShipmentFormWrapper>
    </FormProvider>
  );
}
