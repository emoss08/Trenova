import { Button } from "@/components/ui/button";
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { toUnixTimeStamp } from "@/lib/date";
import { http } from "@/lib/http-client";
import {
  shipmentCancellationSchema,
  type ShipmentCancellationSchema,
} from "@/lib/schemas/shipment-cancellation-schema";
import { useAuthStore } from "@/stores/user-store";
import { APIError } from "@/types/errors";
import { yupResolver } from "@hookform/resolvers/yup";
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
  const { user } = useAuthStore();

  const form = useForm<ShipmentCancellationSchema>({
    resolver: yupResolver(shipmentCancellationSchema),
    defaultValues: {
      cancelReason: "",
      shipmentId: shipmentId,
      canceledById: user?.id,
      canceledAt: toUnixTimeStamp(new Date()),
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const mutation = useMutation({
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
      await mutation.mutateAsync(values);
    },
    [mutation.mutateAsync],
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
          <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
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
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="submit"
                      isLoading={isSubmitting}
                      loadingText="Cancelling..."
                      variant="destructive"
                    >
                      Confirm Cancellation
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent className="flex items-center gap-2">
                    <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                      Ctrl
                    </kbd>
                    <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                      Enter
                    </kbd>
                    <p>to save and close the shipment cancellation</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
