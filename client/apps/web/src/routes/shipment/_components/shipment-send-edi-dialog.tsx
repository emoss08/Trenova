import { useT } from "@trenova/shared/i18n/use-t";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { apiService } from "@/services/api";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useRichT } from "@trenova/shared/i18n/rich";

type ShipmentSendEDIDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipment: Shipment;
};

export function ShipmentSendEDIDialog({
  open,
  onOpenChange,
  shipment,
}: ShipmentSendEDIDialogProps) {
  const t = useT();
  const rt = useRichT();

  const queryClient = useQueryClient();
  const ediPartner = shipment.customer?.ediPartner;
  const mutation = useMutation({
    mutationFn: () =>
      apiService.ediService.submitLoadTender({ sourceShipmentId: shipment.id ?? "" }),
    onSuccess: async () => {
      toast.success(t("EDI load tender submitted"));
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["edi-outbound-transfer-list"] }),
        queryClient.invalidateQueries({ queryKey: ["shipment-list"] }),
      ]);
    },
    onError: () => toast.error(t("Failed to submit EDI load tender")),
  });

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("Send EDI load tender")}</AlertDialogTitle>
          <AlertDialogDescription>
            {rt(
              "{0} will be tendered to <b>{1}</b> for approval by the receiving organization.",
              { b: (c) => <span className="text-foreground font-medium">{c}</span> },
              shipment.proNumber ?? t("This shipment"),
              ediPartner
                ? `${ediPartner.name} (${ediPartner.code})`
                : t("the customer's EDI partner"),
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
          <AlertDialogAction disabled={mutation.isPending} onClick={() => mutation.mutate()}>
            {t("Send tender")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
