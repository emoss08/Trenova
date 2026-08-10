import { TextareaField } from "@/components/fields/textarea-field";
import type { LiveTender, LiveTenderOffer } from "@/lib/graphql/tender";
import { TenderOfferStatusBadge, TenderStatusBadge } from "@trenova/shared/components/status-badge";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form } from "@trenova/shared/components/ui/form";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  cancelTenderPayloadSchema,
  recordTenderResponsePayloadSchema,
  TENDER_CHANNEL_LABEL,
  TENDER_MODE_LABEL,
  type RecordTenderResponsePayload,
} from "@trenova/shared/types/tender";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleAlertIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { formatOfferCountdown, formatOfferRate } from "./tender-vocabulary";

const COUNTDOWN_TICK_MS = 30_000;

function useNowSeconds(): number {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));

  useEffect(() => {
    const interval = window.setInterval(
      () => setNow(Math.floor(Date.now() / 1000)),
      COUNTDOWN_TICK_MS,
    );
    return () => window.clearInterval(interval);
  }, []);

  return now;
}

function TenderCancelDialog({
  open,
  onOpenChange,
  isSubmitting,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isSubmitting: boolean;
  onConfirm: (reason: string) => void;
}) {
  const form = useForm({
    resolver: zodResolver(cancelTenderPayloadSchema),
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ reason: "" });
  }, [open, reset]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset({ reason: "" });
  }, [onOpenChange, reset]);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-[440px]">
        <DialogHeader>
          <DialogTitle>Cancel Tender</DialogTitle>
          <DialogDescription>
            Outstanding offers are withdrawn and carriers can no longer accept. The reason is
            recorded on the tender.
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((values) => onConfirm(values.reason))(event);
          }}
        >
          <div className="pb-4">
            <TextareaField
              control={control}
              name="reason"
              label="Reason"
              rules={{ required: true }}
              placeholder="e.g., Covered internally with a company driver"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Keep tender
            </Button>
            <Button
              type="submit"
              variant="destructive"
              isLoading={isSubmitting}
              loadingText="Canceling..."
            >
              Cancel tender
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

function RecordResponseDialog({
  offer,
  onOpenChange,
  isSubmitting,
  onConfirm,
}: {
  offer: LiveTenderOffer;
  onOpenChange: (open: boolean) => void;
  isSubmitting: boolean;
  onConfirm: (payload: RecordTenderResponsePayload) => void;
}) {
  const form = useForm({
    resolver: zodResolver(recordTenderResponsePayloadSchema),
    defaultValues: { action: "Accept" as const, declineReason: "" },
  });
  const { control, handleSubmit, setValue, watch } = form;
  const action = watch("action");

  return (
    <Dialog open onOpenChange={(nextOpen) => !nextOpen && onOpenChange(false)}>
      <DialogContent className="sm:max-w-[440px]">
        <DialogHeader>
          <DialogTitle>Record Carrier Response</DialogTitle>
          <DialogDescription>
            {offer.carrier?.name ?? "The carrier"} responded off-channel — by phone or a direct
            email. Record it here so the tender advances.
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((values) => onConfirm(values))(event);
          }}
        >
          <div className="flex flex-col gap-3 pb-4">
            <div className="flex gap-2">
              <Button
                type="button"
                variant={action === "Accept" ? "default" : "outline"}
                className="flex-1"
                onClick={() => setValue("action", "Accept")}
              >
                Accepted
              </Button>
              <Button
                type="button"
                variant={action === "Decline" ? "destructive" : "outline"}
                className="flex-1"
                onClick={() => setValue("action", "Decline")}
              >
                Declined
              </Button>
            </div>
            {action === "Decline" && (
              <TextareaField
                control={control}
                name="declineReason"
                label="Decline Reason"
                rules={{ required: true }}
                placeholder="e.g., No truck available in the area"
              />
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText="Recording...">
              Record response
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

function OfferRow({
  offer,
  isCurrent,
  nowSeconds,
  onRecordResponse,
}: {
  offer: LiveTenderOffer;
  isCurrent: boolean;
  nowSeconds: number;
  onRecordResponse: ((offer: LiveTenderOffer) => void) | null;
}) {
  const countdown =
    offer.status === "Sent" ? formatOfferCountdown(offer.expiresAt, nowSeconds) : "";

  return (
    <div
      className={cn(
        "flex flex-col gap-1 rounded-md border p-2",
        isCurrent ? "border-brand/50 bg-brand/5" : "bg-card",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <Badge variant="outline" className="h-4 shrink-0 rounded px-1 text-[9px] tabular-nums">
            #{offer.rank}
          </Badge>
          <span className="truncate text-xs font-medium">
            {offer.carrier?.name ?? "Unknown carrier"}
          </span>
        </div>
        <TenderOfferStatusBadge status={offer.status} className="shrink-0 text-[9px]" />
      </div>

      <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[10px] text-muted-foreground">
        <span className="tabular-nums">{formatOfferRate(offer.rate, offer.rateMethod)}</span>
        <span>· {TENDER_CHANNEL_LABEL[offer.channel]}</span>
        {countdown && (
          <span
            className={cn(
              "tabular-nums",
              countdown === "expired" && "text-red-600 dark:text-red-400",
            )}
          >
            · {countdown}
          </span>
        )}
        {offer.respondedAt != null && (
          <span>· responded {formatUnixDateTime(offer.respondedAt)}</span>
        )}
      </div>

      {offer.status === "Declined" && offer.declineReason && (
        <span className="text-[10px] text-muted-foreground italic">“{offer.declineReason}”</span>
      )}
      {offer.status === "DeliveryFailed" && offer.deliveryError && (
        <span className="text-[10px] text-red-600 dark:text-red-400">{offer.deliveryError}</span>
      )}

      {offer.status === "Sent" && onRecordResponse && (
        <div>
          <Button
            size="sm"
            variant="outline"
            className="h-6 px-2 text-[10px]"
            onClick={() => onRecordResponse(offer)}
          >
            Record response
          </Button>
        </div>
      )}
    </div>
  );
}

/**
 * The live tender, in full: where the ladder stands, every offer's fate, and
 * the two interventions a dispatcher has — recording an off-channel response
 * and pulling the tender back.
 */
export function TenderLivePanel({
  tender,
  isTendering,
  onCancelTender,
  onRecordResponse,
  onAssignManually,
}: {
  tender: LiveTender;
  isTendering: boolean;
  onCancelTender: (params: { tenderId: string; reason: string }) => Promise<unknown>;
  onRecordResponse: (params: {
    offerId: string;
    payload: RecordTenderResponsePayload;
  }) => Promise<unknown>;
  /** Opens the manual carrier-assign flow; shown on the NeedsReview banner. */
  onAssignManually?: () => void;
}) {
  const nowSeconds = useNowSeconds();
  const [cancelOpen, setCancelOpen] = useState(false);
  const [respondingTo, setRespondingTo] = useState<LiveTenderOffer | null>(null);

  const offers = [...(tender.offers ?? [])].sort((a, b) => a.rank - b.rank);
  const isLive = tender.status === "Active" || tender.status === "NeedsReview";

  const handleCancel = useCallback(
    (reason: string) => {
      void onCancelTender({ tenderId: tender.id, reason })
        .then(() => setCancelOpen(false))
        .catch(() => {
          // handled by the mutation's onError; the dialog stays open for a retry
        });
    },
    [onCancelTender, tender.id],
  );

  const handleRecord = useCallback(
    (payload: RecordTenderResponsePayload) => {
      if (!respondingTo) return;
      void onRecordResponse({ offerId: respondingTo.id, payload })
        .then(() => setRespondingTo(null))
        .catch(() => {
          // handled by the mutation's onError; the dialog stays open for a retry
        });
    },
    [onRecordResponse, respondingTo],
  );

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-1.5">
        <TenderStatusBadge status={tender.status} />
        <Badge variant="outline" className="h-4 rounded px-1 text-[9px]">
          {TENDER_MODE_LABEL[tender.mode]}
        </Badge>
        {tender.routingGuide && (
          <span className="text-[10px] text-muted-foreground">via {tender.routingGuide.name}</span>
        )}
      </div>

      {tender.status === "NeedsReview" && (
        <Alert variant="destructive">
          <TriangleAlertIcon className="size-4" aria-hidden />
          <AlertTitle>Carrier accepted, but auto-assignment failed</AlertTitle>
          <AlertDescription>
            The accepting carrier could not be assigned to the move automatically — the move may
            have gained coverage in the meantime, or the assignment was blocked. Assign the carrier
            manually with the regular carrier-assign flow, or cancel this tender.
            {onAssignManually && (
              <Button
                size="sm"
                variant="outline"
                className="mt-2 h-6 px-2 text-[10px]"
                onClick={onAssignManually}
              >
                Assign manually
              </Button>
            )}
          </AlertDescription>
        </Alert>
      )}

      <div className="flex flex-col gap-1.5">
        {offers.map((offer) => (
          <OfferRow
            key={offer.id}
            offer={offer}
            isCurrent={tender.status === "Active" && offer.rank === tender.currentRank}
            nowSeconds={nowSeconds}
            onRecordResponse={isLive ? setRespondingTo : null}
          />
        ))}
        {offers.length === 0 && (
          <p className="py-3 text-center text-[11px] text-muted-foreground">
            No offers on this tender.
          </p>
        )}
      </div>

      {isLive && (
        <div>
          <Button
            size="sm"
            variant="outline"
            className="h-6 px-2 text-[10px] text-destructive hover:text-destructive"
            disabled={isTendering}
            onClick={() => setCancelOpen(true)}
          >
            Cancel tender
          </Button>
        </div>
      )}

      <TenderCancelDialog
        open={cancelOpen}
        onOpenChange={setCancelOpen}
        isSubmitting={isTendering}
        onConfirm={handleCancel}
      />
      {respondingTo && (
        <RecordResponseDialog
          offer={respondingTo}
          onOpenChange={(open) => {
            if (!open) setRespondingTo(null);
          }}
          isSubmitting={isTendering}
          onConfirm={handleRecord}
        />
      )}
    </div>
  );
}
