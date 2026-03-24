import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { transferOwnershipSchema, type TransferOwnershipPayload } from "@/types/shipment";
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
    setError,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: TransferOwnershipPayload) =>
      apiService.shipmentService.transferOwnership(shipmentId, values.ownerId),
    resourceName: "Shipment",
    setFormError: setError,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      toast.success("Ownership transferred", {
        description: "The shipment has been transferred to the new owner.",
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
      <DialogContent className="sm:max-w-100">
        <DialogHeader>
          <DialogTitle>Transfer Ownership</DialogTitle>
          <DialogDescription>Transfer this shipment to a different user.</DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <UserAutocompleteField
                control={control}
                name="ownerId"
                label="New Owner"
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Close
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText="Transferring...">
              Transfer Ownership
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
