import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import type { GetPreviousRatesRequest, PreviousRateSummary } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { HistoryIcon } from "lucide-react";
import { useState } from "react";

type PreviousRatesDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  request: GetPreviousRatesRequest;
};

export function PreviousRatesDialog({ open, onOpenChange, request }: PreviousRatesDialogProps) {
  const t = useT();

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
            {t("Previous Rates")}
          </DialogTitle>
          <DialogDescription>
            {t("Historical rates for this lane, service, and shipment type")}
          </DialogDescription>
        </DialogHeader>
        <div className="max-h-80 overflow-y-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Spinner className="size-5" />
            </div>
          ) : !rates || rates.items.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <HistoryIcon className="text-muted-foreground/40 mb-2 size-6" />
              <p className="text-muted-foreground text-sm">{t("No previous rates found for this lane")}</p>
            </div>
          ) : (
            <>
              <p className="text-muted-foreground mb-2 text-xs">
                {t("{0} previous rate{1} found", rates.total, rates.total !== 1 ? "s" : "")}
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
  const t = useT();

  return (
    <div className="bg-card rounded-lg border p-3">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium">{rate.proNumber}</span>
        <span className="text-muted-foreground text-xs">
          {formatToUserTimezone(rate.createdAt, { showTime: false })}
        </span>
      </div>
      <div className="mt-2 grid grid-cols-3 gap-2">
        <div>
          <p className="text-2xs text-muted-foreground">{t("Freight")}</p>
          <p className="text-xs font-medium">{formatCurrency(Number(rate.freightChargeAmount))}</p>
        </div>
        <div>
          <p className="text-2xs text-muted-foreground">{t("Other")}</p>
          <p className="text-xs font-medium">{formatCurrency(Number(rate.otherChargeAmount))}</p>
        </div>
        <div>
          <p className="text-2xs text-muted-foreground">{t("Total")}</p>
          <p className="text-primary text-xs font-semibold">
            {formatCurrency(Number(rate.totalChargeAmount))}
          </p>
        </div>
      </div>
      {(rate.pieces || rate.weight) && (
        <div className="text-2xs text-muted-foreground mt-1 flex gap-3">
          {rate.pieces && <span>{t("{0} pcs", rate.pieces)}</span>}
          {rate.weight && <span>{t("{0} lbs", rate.weight)}</span>}
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
  const t = useT();

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
        {t("View Previous Rates")}
      </Button>
      {open && <PreviousRatesDialog open={open} onOpenChange={setOpen} request={request} />}
    </>
  );
}
