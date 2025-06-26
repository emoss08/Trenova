import { Button, FormSaveButton } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { toUnixTimeStamp } from "@/lib/date";
import { http } from "@/lib/http-client";
import {
  shipmentCancellationSchema,
  type ShipmentCancellationSchema,
} from "@/lib/schemas/shipment-cancellation-schema";
import { useUser } from "@/stores/user-store";
import { APIError } from "@/types/errors";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, type Path, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ShipmentCancellationForm } from "./shipment-cancellation-form";

type ShipmentCancellationDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId?: string;
};

export function ShipmentCancellationDialog({
  open,
  onOpenChange,
  shipmentId,
}: ShipmentCancellationDialogProps) {
  const user = useUser();

  const form = useForm({
    resolver: zodResolver(shipmentCancellationSchema),
    defaultValues: {
      cancelReason: "",
      shipmentId: shipmentId || "",
      canceledById: user?.id || "",
      canceledAt: toUnixTimeStamp(new Date()) || 0,
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const { mutateAsync } = useMutation({
    mutationFn: async (values: ShipmentCancellationSchema) => {
      const response = await http.post(`/shipments/cancel/`, values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Shipment cancelled successfully", {
        description: `The shipment has been cancelled`,
      });
      onOpenChange(false);
      reset();

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: ["assignment-list", "shipment"],
        options: {
          correlationId: `create-shipment-move-assignment-${Date.now()}`,
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
          setError(fieldError.name as Path<ShipmentCancellationSchema>, {
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
    async (values: ShipmentCancellationSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[400px]">
        <DialogHeader>
          <DialogTitle>Cancel Shipment</DialogTitle>
          <DialogDescription>
            Cancel the shipment and all associated moves.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="space-y-0 p-0"
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              handleSubmit(onSubmit)(e);
            }}
          >
            <DialogBody>
              <ShipmentCancellationForm />
            </DialogBody>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(!open)}
              >
                Cancel
              </Button>
              <FormSaveButton
                type="button"
                onClick={() => handleSubmit(onSubmit)()}
                isSubmitting={isSubmitting}
                title="shipment cancellation"
                text="Confirm Cancellation"
                variant="destructive"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
