import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { Form } from "@/components/ui/form";
import { ScrollArea } from "@/components/ui/scroll-area";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  shipmentSchema,
  type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { type APIError } from "@/types/errors";
import { Shipment } from "@/types/shipment";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation } from "@tanstack/react-query";
import { lazy, useCallback } from "react";
import { FormProvider, type Path, useForm } from "react-hook-form";
import { useParams } from "react-router";
import { toast } from "sonner";
import { useShipmentDetails } from "../queries/shipment";

const RatingDetails = lazy(() => import("./_components/rating-details"));
const EquipmentDetails = lazy(() => import("./_components/equipment-details"));
const GeneralInformation = lazy(
  () => import("./_components/general-information/general-information"),
);

export function ShipmentDetails() {
  const { id } = useParams<"id">();

  // Fetch the shipment information from the server
  const shipmentDetailsQuery = useShipmentDetails({
    shipmentId: id ?? "",
    enabled: Boolean(id),
  });

  const shipmentDetails = shipmentDetailsQuery.data;

  return (
    <>
      <MetaTags title="Shipment Details" description="Shipment Details" />
      {shipmentDetails && <ShipmentForm shipment={shipmentDetails} />}
    </>
  );
}

function ShipmentForm({ shipment }: { shipment: Shipment }) {
  const form = useForm<ShipmentSchema>({
    resolver: yupResolver(shipmentSchema),
    defaultValues: shipment,
  });

  const { setError, handleSubmit, reset } = form;

  const { mutateAsync } = useMutation({
    mutationFn: async (values: ShipmentSchema) => {
      const response = await http.put(`/shipments/${shipment.id}`, values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Changes have been saved.", {
        description: `Shipment updated successfully`,
      });
      reset();

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: ["shipment-list", "shipment"],
        options: {
          correlationId: `update-shipment-${shipment.id}-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as Path<ShipmentSchema>, {
            message: fieldError.reason,
          });
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
  });

  const onSubmit = useCallback(
    async (values: ShipmentSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <SuspenseLoader>
          <div className="max-w-9xl">
            <div className="grid grid-cols-12 gap-2 mx-auto">
              <div className="col-span-8">
                <GeneralInformation />
              </div>
              <div className="col-span-4">
                <ScrollArea className="flex max-h-[calc(100vh-80px)] flex-col overflow-y-auto rounded-lg pr-4">
                  <div className="grid grid-cols-1 gap-4">
                    <RatingDetails />
                    <EquipmentDetails />
                  </div>
                  <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-10 bg-gradient-to-t from-sidebar to-transparent" />
                </ScrollArea>
              </div>
            </div>
          </div>
        </SuspenseLoader>
      </Form>
    </FormProvider>
  );
}
