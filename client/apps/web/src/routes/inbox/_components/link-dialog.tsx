import {
  CustomerAutocompleteField,
  ShipmentAutocompleteField,
} from "@/components/autocomplete-fields";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { linkInboundMessage, type InboundMessageDetail } from "@/lib/graphql/inbox";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

/** The column's length, so a long reason is a field error rather than a failed save. */
const MAX_REASON_LENGTH = 2000;

export const linkMessageSchema = z
  .object({
    shipmentId: z.string().nullable(),
    customerId: z.string().nullable(),
    reason: z
      .string()
      .trim()
      .min(1, { error: "Say why this is the right record" })
      .max(MAX_REASON_LENGTH, { error: `At most ${MAX_REASON_LENGTH} characters` }),
  })
  .refine((value) => Boolean(value.shipmentId) || Boolean(value.customerId), {
    error: "Choose a shipment, a customer, or both",
    path: ["shipmentId"],
  });

export type LinkMessageValues = z.infer<typeof linkMessageSchema>;

/**
 * Says what a message is about, by hand, when the reading could not.
 *
 * The reason is required and stored beside the link. A link is a claim the
 * next person will act on — filing a POD, answering a customer — and a claim
 * nobody can check is one nobody should trust.
 */
export function LinkMessageDialog({
  message,
  open,
  onOpenChange,
  onLinked,
}: {
  message: InboundMessageDetail;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onLinked: () => Promise<void> | void;
}) {
  const t = useT();
  const form = useForm<LinkMessageValues>({
    resolver: zodResolver(linkMessageSchema),
    defaultValues: {
      shipmentId: message.matchedShipmentId ?? null,
      customerId: message.matchedCustomerId ?? null,
      reason: "",
    },
  });

  useEffect(() => {
    if (open) {
      form.reset({
        shipmentId: message.matchedShipmentId ?? null,
        customerId: message.matchedCustomerId ?? null,
        reason: "",
      });
    }
  }, [open, form, message.matchedShipmentId, message.matchedCustomerId]);

  const linkMutation = useApiMutation({
    mutationFn: (values: LinkMessageValues) =>
      linkInboundMessage(message.id, {
        shipmentId: values.shipmentId || null,
        customerId: values.customerId || null,
        reason: values.reason.trim(),
      }),
    onSuccess: async () => {
      toast.success(t("Linked"));
      onOpenChange(false);
      await onLinked();
    },
    form,
    resourceName: "Message",
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <form onSubmit={form.handleSubmit((values) => linkMutation.mutate(values))}>
          <DialogHeader>
            <DialogTitle>{t("Link by hand")}</DialogTitle>
            <DialogDescription>
              {t("Say which shipment or customer this message is about, and why.")}
            </DialogDescription>
          </DialogHeader>
          <FormGroup cols={1} className="py-4">
            <FormControl>
              <ShipmentAutocompleteField
                control={form.control}
                name="shipmentId"
                label={t("Shipment")}
                placeholder={t("Find a shipment")}
                clearable
              />
            </FormControl>
            <FormControl>
              <CustomerAutocompleteField
                control={form.control}
                name="customerId"
                label={t("Customer")}
                placeholder={t("Find a customer")}
                clearable
              />
            </FormControl>
            <FormControl>
              <TextareaField
                control={form.control}
                name="reason"
                label={t("Why this is the right record")}
                placeholder={t(
                  "The PRO in the subject line matches, and the sender is their dispatcher",
                )}
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={linkMutation.isPending} loadingText={t("Linking…")}>
              {t("Link")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
