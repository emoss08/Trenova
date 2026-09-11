import { useT } from "@trenova/shared/i18n/use-t";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { DuplicateShipmentRequest } from "@trenova/shared/types/shipment";
import { duplicateShipmentRequestSchema } from "@trenova/shared/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type ShipmentDuplicateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId: string;
};

export function ShipmentDuplicateDialog({
  open,
  onOpenChange,
  shipmentId,
}: ShipmentDuplicateDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();

  const form = useForm({
    resolver: zodResolver(duplicateShipmentRequestSchema),
    defaultValues: {
      shipmentId,
      count: 1,
      overrideDates: false,
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (payload: DuplicateShipmentRequest) =>
      apiService.shipmentService.duplicate(payload),
    resourceName: "Shipment",
    form,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success(t("Shipment duplication started"), {
        description: t("The shipment will be duplicated in the background."),
      });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset({ shipmentId, count: 1, overrideDates: false });
  }, [onOpenChange, reset, shipmentId]);

  const onSubmit = useCallback(
    async (values: DuplicateShipmentRequest) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>{t("Duplicate Shipment")}</DialogTitle>
          <DialogDescription>{t("Create one or more copies of this shipment.")}</DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <NumberField
                control={control}
                name="count"
                label={t("Number of Copies")}
                placeholder="1"
                min={1}
                max={20}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name="overrideDates"
                label={t("Override Dates")}
                description={t("Reset planned arrival and departure times on the duplicated shipment stops.")}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText={t("Duplicating...")}>
              {t("Duplicate")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
