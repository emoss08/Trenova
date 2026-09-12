import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { canMarkShipmentReadyToBill, canTransferShipmentToBilling } from "@/lib/shipment-utils";
import { SHIPMENT_LIST_KEY } from "@/routes/shipment/_components/shipment-queries";
import { apiService } from "@/services/api";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQueryClient } from "@tanstack/react-query";
import { BanknoteArrowUpIcon, CheckCircle2Icon, SendIcon, type LucideIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";

export type ShipmentBillingAction = {
  id: string;
  label: string;
  icon: LucideIcon;
  isAvailable: (shipment: Shipment) => boolean;
  /** Never rejects: a failure has already been reported to the user when it settles. */
  run: (shipmentId: string) => Promise<void>;
};

export type ShipmentBillingActions = {
  markReadyToBill: ShipmentBillingAction;
  markReadyAndTransferToBilling: ShipmentBillingAction;
  transferToBilling: ShipmentBillingAction;
};

export function useShipmentBillingActions(): ShipmentBillingActions {
  const queryClient = useQueryClient();

  const invalidateShipment = useCallback(
    (shipmentId: string) =>
      Promise.all([
        // The detail key ends in its params; dropping them matches every cached variant.
        queryClient.invalidateQueries({
          queryKey: queries.shipment.get(shipmentId).queryKey.slice(0, -1),
        }),
        queryClient.invalidateQueries({
          queryKey: queries.shipment.billingReadiness(shipmentId).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: [SHIPMENT_LIST_KEY] }),
      ]),
    [queryClient],
  );

  const { mutateAsync: markReadyToBill } = useApiMutation({
    mutationFn: async (shipmentId: string) => {
      const shipment = await apiService.shipmentService.get(shipmentId);
      return apiService.shipmentService.update(shipmentId, {
        ...shipment,
        status: "ReadyToInvoice",
      });
    },
    onSuccess: () => {
      toast.success("Shipment marked ready to bill");
    },
    onSettled: (_data, _error, shipmentId) => invalidateShipment(shipmentId),
    resourceName: "Shipment",
  });

  const { mutateAsync: transferToBilling } = useApiMutation({
    mutationFn: (shipmentId: string) => apiService.shipmentService.transferToBilling(shipmentId),
    onSuccess: () => {
      toast.success("Transferred to billing", {
        description: "The shipment has been added to the billing queue.",
      });
    },
    onSettled: (_data, _error, shipmentId) => invalidateShipment(shipmentId),
    resourceName: "Shipment",
  });

  return useMemo(
    () => ({
      markReadyToBill: {
        id: "mark-ready-to-bill",
        label: "Mark Ready to Bill",
        icon: CheckCircle2Icon,
        isAvailable: canMarkShipmentReadyToBill,
        run: async (shipmentId) => {
          await markReadyToBill(shipmentId).catch(() => undefined);
        },
      },
      markReadyAndTransferToBilling: {
        id: "mark-ready-and-transfer-to-billing",
        label: "Mark Ready & Transfer to Billing",
        icon: BanknoteArrowUpIcon,
        isAvailable: canMarkShipmentReadyToBill,
        run: async (shipmentId) => {
          await markReadyToBill(shipmentId)
            .then(() => transferToBilling(shipmentId))
            .catch(() => undefined);
        },
      },
      transferToBilling: {
        id: "transfer-to-billing",
        label: "Transfer to Billing",
        icon: SendIcon,
        isAvailable: canTransferShipmentToBilling,
        run: async (shipmentId) => {
          await transferToBilling(shipmentId).catch(() => undefined);
        },
      },
    }),
    [markReadyToBill, transferToBilling],
  );
}
