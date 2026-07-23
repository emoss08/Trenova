import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { formatToUserTimezone } from "@/lib/date";
import { formatCurrency } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { GetPreviousRatesRequest, PreviousRateSummary } from "@/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { HistoryIcon } from "lucide-react";
import { useState } from "react";

type PreviousRatesDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  request: GetPreviousRatesRequest;
};

export function PreviousRatesDialog({ open, onOpenChange, request }: PreviousRatesDialogProps) {
  const { data: rates, isLoading } = useQuery({
    queryKey: ["previous-rates", request],
    queryFn: () => apiService.shipmentService.getPreviousRates(request),
    enabled: open && !!request.originLocationId && !!request.destinationLocationId,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <HistoryIcon className="size-4" />
            Previous Rates
          </DialogTitle>
          <DialogDescription>
            Historical rates for this lane, service, and shipment type
          </DialogDescription>
        </DialogHeader>
        <div className="max-h-80 overflow-y-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Spinner className="size-5" />
            </div>
          ) : !rates || rates.items.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <HistoryIcon className="mb-2 size-6 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">No previous rates found for this lane</p>
            </div>
          ) : (
            <>
              <p className="mb-2 text-xs text-muted-foreground">
                {rates.total} previous rate{rates.total !== 1 ? "s" : ""} found
              </p>
              <div className="space-y-2">
                {rates.items.map((rate) => (
                  <RateCard key={rate.shipmentId} rate={rate} />
                ))}
              </div>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function RateCard({ rate }: { rate: PreviousRateSummary }) {
  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium">{rate.proNumber}</span>
        <span className="text-xs text-muted-foreground">
          {formatToUserTimezone(rate.createdAt, { showTime: false })}
        </span>
      </div>
      <div className="mt-2 grid grid-cols-3 gap-2">
        <div>
          <p className="text-2xs text-muted-foreground">Freight</p>
          <p className="text-xs font-medium">{formatCurrency(Number(rate.freightChargeAmount))}</p>
        </div>
        <div>
          <p className="text-2xs text-muted-foreground">Other</p>
          <p className="text-xs font-medium">{formatCurrency(Number(rate.otherChargeAmount))}</p>
        </div>
        <div>
          <p className="text-2xs text-muted-foreground">Total</p>
          <p className="text-xs font-semibold text-primary">
            {formatCurrency(Number(rate.totalChargeAmount))}
          </p>
        </div>
      </div>
      {(rate.pieces || rate.weight) && (
        <div className="mt-1 flex gap-3 text-2xs text-muted-foreground">
          {rate.pieces && <span>{rate.pieces} pcs</span>}
          {rate.weight && <span>{rate.weight} lbs</span>}
        </div>
      )}
    </div>
  );
}

export function PreviousRatesButton({
  request,
  disabled,
}: {
  request: GetPreviousRatesRequest;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);

  const canFetch =
    !!request.originLocationId &&
    !!request.destinationLocationId &&
    !!request.shipmentTypeId &&
    !!request.serviceTypeId;

  return (
    <>
      <Button
        type="button"
        size="xxxs"
        className="text-2xs"
        disabled={disabled || !canFetch}
        onClick={() => setOpen(true)}
      >
        View Previous Rates
      </Button>
      {open && <PreviousRatesDialog open={open} onOpenChange={setOpen} request={request} />}
    </>
  );
}
