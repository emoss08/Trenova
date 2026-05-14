import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { EDIPartner } from "@/types/edi";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";

type ShipmentSendEDIDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId: string;
};

export function ShipmentSendEDIDialog({
  open,
  onOpenChange,
  shipmentId,
}: ShipmentSendEDIDialogProps) {
  const queryClient = useQueryClient();
  const [partnerId, setPartnerId] = useState("");
  const mutation = useMutation({
    mutationFn: () =>
      apiService.ediService.submitLoadTender({
        sourceShipmentId: shipmentId,
        ediPartnerId: partnerId,
      }),
    onSuccess: async () => {
      toast.success("EDI load tender submitted");
      await queryClient.invalidateQueries({ queryKey: queries.edi.outboundTransfers._def });
      onOpenChange(false);
      setPartnerId("");
    },
    onError: () => toast.error("Failed to submit EDI load tender"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Send EDI Load Tender</DialogTitle>
          <DialogDescription>Select an active internal outbound partner.</DialogDescription>
        </DialogHeader>
        <Autocomplete<EDIPartner, Record<string, string>>
          link="/edi/partners/select-options/"
          selectedValueLink="/edi/partners/"
          label="EDI partner"
          placeholder="Search EDI partners..."
          value={partnerId}
          onChange={(value) => setPartnerId(value ?? "")}
          getOptionValue={(partner) => partner.id}
          getDisplayValue={(partner) => `${partner.name} (${partner.code})`}
          renderOption={(partner) => (
            <div className="flex min-w-0 flex-col items-start">
              <span className="truncate text-sm font-medium">{partner.name}</span>
              <span className="truncate text-xs text-muted-foreground">{partner.code}</span>
            </div>
          )}
          noResultsMessage="No active internal outbound partners found."
          extraSearchParams={{ kind: "Internal", enabledForOutbound: "true" }}
          initialLimit={20}
          clearable
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={!partnerId} isLoading={mutation.isPending} onClick={() => mutation.mutate()}>
            Send Tender
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
