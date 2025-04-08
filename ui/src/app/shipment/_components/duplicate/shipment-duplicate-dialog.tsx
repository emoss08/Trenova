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
import { http } from "@/lib/http-client";
import {
  shipmentDuplicateSchema,
  type ShipmentDuplicateSchema,
} from "@/lib/schemas/shipment-duplicate-schema";
import { type TableSheetProps } from "@/types/data-table";
import { type APIError } from "@/types/errors";
import { type Shipment } from "@/types/shipment";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, type Path, useForm } from "react-hook-form";
import { toast } from "sonner";
import { ShipmentDuplicateForm } from "./shipment-duplicate-form";

type ShipmentDuplicateDialogProps = TableSheetProps & {
  shipment: Shipment;
};

export function ShipmentDuplicateDialog({
  open,
  onOpenChange,
  shipment,
}: ShipmentDuplicateDialogProps) {
  const form = useForm<ShipmentDuplicateSchema>({
    resolver: yupResolver(shipmentDuplicateSchema),
    defaultValues: {
      shipmentID: shipment?.id,
      overrideDates: false,
      includeCommodities: false,
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const { mutateAsync: duplicateShipment } = useMutation({
    mutationFn: async (values: ShipmentDuplicateSchema) => {
      const response = await http.post(`/shipments/duplicate/`, values);

      return response.data;
    },
    onSuccess: () => {
      toast.success("Shipment duplicated successfully");
      onOpenChange(false);
      reset();

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: ["shipment"],
        options: {
          correlationId: `duplicate-shipment-${Date.now()}`,
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
          setError(fieldError.name as Path<ShipmentDuplicateSchema>, {
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
    async (values: ShipmentDuplicateSchema) => {
      await duplicateShipment(values);
    },
    [duplicateShipment],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Duplicate {shipment?.proNumber}</DialogTitle>
          <DialogDescription>
            Duplicate the shipment with the same details.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
            <DialogBody>
              <ShipmentDuplicateForm />
            </DialogBody>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      onClick={() => handleSubmit(onSubmit)()}
                      isLoading={isSubmitting}
                      loadingText="Duplicating..."
                    >
                      Duplicate
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent className="flex items-center gap-2">
                    <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                      Ctrl
                    </kbd>
                    <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                      Enter
                    </kbd>
                    <p>to save and close the duplicate dialog</p>
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
