import { CarrierAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { linkEdiCarrierInvoiceToCarrier } from "@/lib/graphql/carrier-settlement";
import { useMutation } from "@tanstack/react-query";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type LinkCarrierFormValues = {
  carrierId: string;
};

export function LinkCarrierDialog({
  open,
  invoiceId,
  invoiceNumber,
  suggestedCarrierId,
  onOpenChange,
  onLinked,
}: {
  open: boolean;
  invoiceId: string;
  invoiceNumber: string;
  suggestedCarrierId?: string | null;
  onOpenChange: (open: boolean) => void;
  onLinked: () => void;
}) {
  const form = useForm<LinkCarrierFormValues>({
    defaultValues: { carrierId: suggestedCarrierId ?? "" },
  });
  const { control, handleSubmit, watch, reset } = form;
  const carrierId = watch("carrierId");

  // Default values only apply on mount, so re-seed the form whenever the dialog
  // opens or a suggestion arrives — otherwise a previous invoice's carrier (or a
  // stale empty value) leaks into this one.
  useEffect(() => {
    if (!open) return;
    reset({ carrierId: suggestedCarrierId ?? "" });
  }, [open, suggestedCarrierId, reset]);

  const mutation = useMutation({
    mutationFn: (values: LinkCarrierFormValues) =>
      linkEdiCarrierInvoiceToCarrier(invoiceId, values.carrierId),
    onSuccess: () => {
      toast.success("Invoice linked to carrier");
      onOpenChange(false);
      onLinked();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to link carrier"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Link invoice to carrier</DialogTitle>
          <DialogDescription>
            Ties invoice {invoiceNumber || invoiceId} to a carrier in the master so it can be
            matched against that carrier&apos;s assignments.
          </DialogDescription>
        </DialogHeader>
        <CarrierAutocompleteField<LinkCarrierFormValues>
          control={control}
          name="carrierId"
          label="Carrier"
          placeholder="Search by code or name"
          rules={{ required: true }}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={!carrierId || mutation.isPending}
            onClick={handleSubmit((values) => mutation.mutate(values))}
          >
            Link Carrier
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
