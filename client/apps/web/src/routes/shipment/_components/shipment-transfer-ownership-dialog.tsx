import { useT } from "@trenova/shared/i18n/use-t";
import { UserAutocompleteField } from "@/components/autocomplete-fields";
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
import {
  transferOwnershipSchema,
  type TransferOwnershipPayload,
} from "@trenova/shared/types/shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type ShipmentTransferOwnershipDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId: string;
};

export function ShipmentTransferOwnershipDialog({
  open,
  onOpenChange,
  shipmentId,
}: ShipmentTransferOwnershipDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();

  const form = useForm<TransferOwnershipPayload>({
    resolver: zodResolver(transferOwnershipSchema),
    defaultValues: {
      ownerId: "",
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: TransferOwnershipPayload) =>
      apiService.shipmentService.transferOwnership(shipmentId, values.ownerId),
    resourceName: "Shipment",
    form,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success(t("Ownership transferred"), {
        description: t("The shipment has been transferred to the new owner."),
      });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset({ ownerId: "" });
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: TransferOwnershipPayload) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Transfer ownership")}</DialogTitle>
          <DialogDescription>{t("Transfer this shipment to a different user.")}</DialogDescription>
        </DialogHeader>
        <Form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <FormGroup cols={1}>
            <FormControl>
              <UserAutocompleteField
                control={control}
                name="ownerId"
                label={t("New owner")}
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              {t("Close")}
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText={t("Transferring...")}>
              {t("Transfer ownership")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
