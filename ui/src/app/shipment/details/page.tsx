import { SelectField } from "@/components/fields/select-field";
import { MetaTags } from "@/components/meta-tags";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { shipmentStatusChoices } from "@/lib/choices";
import { http } from "@/lib/http-client";
import {
    shipmentSchema,
    type ShipmentSchema,
} from "@/lib/schemas/shipment-schema";
import { type APIError } from "@/types/errors";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation } from "@tanstack/react-query";
import { useCallback } from "react";
import {
    FormProvider,
    type Path,
    useForm,
    useFormContext,
} from "react-hook-form";
import { useParams } from "react-router";
import { toast } from "sonner";
import { useShipmentDetails } from "../queries/shipment";

export function ShipmentDetails() {
  const { id } = useParams<"id">();

  // Fetch the shipment information from the server
  const shipmentDetailsQuery = useShipmentDetails({
    shipmentId: id ?? "",
    enabled: Boolean(id),
  });

  const shipmentDetails = shipmentDetailsQuery.data;
  //   const isLoading = shipmentDetailsQuery.isLoading;

  const form = useForm<ShipmentSchema>({
    resolver: yupResolver(shipmentSchema),
    defaultValues: shipmentDetails,
  });

  return (
    <>
      <MetaTags title="Shipment Details" description="Shipment Details" />
      <FormProvider {...form}>
        <ShipmentForm shipmentId={id ?? ""} />
      </FormProvider>
    </>
  );
}

function ShipmentForm({ shipmentId }: { shipmentId: string }) {
  const methods = useFormContext<ShipmentSchema>();
  const {
    control,
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = methods;

  const { mutateAsync } = useMutation({
    mutationFn: async (values: ShipmentSchema) => {
      const response = await http.put(`/shipments/${shipmentId}`, values);
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
          correlationId: `create-shipment-${shipmentId}-${Date.now()}`,
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
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup cols={2} className="gap-4">
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="status"
            label="Current Status"
            placeholder="Current Status"
            description="Indicates the current status of the shipment."
            options={shipmentStatusChoices}
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}
