import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
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
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  cancelTenderPayloadSchema,
  recordTenderResponsePayloadSchema,
  TENDER_CHANNEL_LABEL,
  TENDER_MODE_LABEL,
  TENDER_RESPONSE_SOURCE_LABEL,
  type BounceTenderPayload,
  type RecordTenderResponsePayload,
} from "@trenova/shared/types/tender";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { useCallback, useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { TenderBounceDialog } from "./tender-bounce-dialog";
import { formatCounterMargin, formatOfferCountdown, formatOfferRate } from "./tender-vocabulary";
import type { RecordOfferResponseParams } from "./use-dispatch-actions";

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
  const t = useT();

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
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Cancel tender")}</DialogTitle>
          <DialogDescription>
            {t(
              "Outstanding offers are withdrawn and carriers can no longer accept. The reason is recorded on the tender.",
            )}
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
              label={t("Reason")}
              rules={{ required: true }}
              placeholder={t("e.g., Covered internally with a company driver")}
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              {t("Keep tender")}
            </Button>
            <Button
              type="submit"
              variant="destructive"
              isLoading={isSubmitting}
              loadingText={t("Canceling...")}
            >
              {t("Cancel tender")}
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
  const t = useT();

  const form = useForm({
    resolver: zodResolver(recordTenderResponsePayloadSchema),
    defaultValues: {
      action: "Accept" as RecordTenderResponsePayload["action"],
      declineReason: "",
      counterRate: "",
      counterNote: "",
    },
  });
  const { control, handleSubmit, setValue } = form;
  const action = useWatch({ control, name: "action" });

  return (
    <Dialog open onOpenChange={(nextOpen) => !nextOpen && onOpenChange(false)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Record carrier response")}</DialogTitle>
          <DialogDescription>
            {t(
              "{0} responded off-channel — by phone or a direct email. Record it here so the tender advances.",
              offer.carrier?.name ?? t("The carrier"),
            )}
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
                {t("Accepted")}
              </Button>
              <Button
                type="button"
                variant={action === "Decline" ? "destructive" : "outline"}
                className="flex-1"
                onClick={() => setValue("action", "Decline")}
              >
                {t("Declined")}
              </Button>
              <Button
                type="button"
                variant={action === "Counter" ? "default" : "outline"}
                className="flex-1"
                onClick={() => setValue("action", "Counter")}
              >
                {t("Countered")}
              </Button>
            </div>
            {action === "Counter" && (
              <>
                <InputField
                  control={control}
                  name="counterRate"
                  label={t("Counter rate")}
                  description={t(
                    "The rate the carrier asked for, in the offer's terms ({0}).",
                    formatOfferRate(offer.rate, offer.rateMethod),
                  )}
                  rules={{ required: true }}
                  inputMode="decimal"
                  autoComplete="off"
                  placeholder={t("e.g., 1850.00")}
                />
                <TextareaField
                  control={control}
                  name="counterNote"
                  label={t("Carrier's note")}
                  placeholder={t("e.g., Fuel is up on this lane")}
                />
              </>
            )}
            {action === "Decline" && (
              <TextareaField
                control={control}
                name="declineReason"
                label={t("Decline reason")}
                rules={{ required: true }}
                placeholder={t("e.g., No truck available in the area")}
              />
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText={t("Recording...")}>
              {t("Record response")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

type CounterDecision = { offer: LiveTenderOffer; approve: boolean };

function CounterDecisionDialog({
  decision,
  onOpenChange,
  isSubmitting,
  onConfirm,
}: {
  decision: CounterDecision;
  onOpenChange: (open: boolean) => void;
  isSubmitting: boolean;
  onConfirm: (payload: RecordTenderResponsePayload) => void;
}) {
  const t = useT();
  const { offer, approve } = decision;
  const carrierName = offer.carrier?.name ?? t("The carrier");
  const counterRate = formatOfferRate(offer.counterRate ?? "", offer.rateMethod);

  const form = useForm({
    resolver: zodResolver(recordTenderResponsePayloadSchema),
    defaultValues: {
      action: (approve ? "Accept" : "Decline") as RecordTenderResponsePayload["action"],
      declineReason: "",
      counterRate: "",
      counterNote: "",
    },
  });
  const { control, handleSubmit } = form;

  return (
    <Dialog open onOpenChange={(nextOpen) => !nextOpen && onOpenChange(false)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{approve ? t("Take counter-offer") : t("Refuse counter-offer")}</DialogTitle>
          <DialogDescription>
            {approve
              ? t(
                  "{0} is assigned to the move at {1} instead of the offered {2}, and the rest of the tender is closed.",
                  carrierName,
                  counterRate,
                  formatOfferRate(offer.rate, offer.rateMethod),
                )
              : t(
                  "{0} is told the counter of {1} was refused, and the tender goes on to the next carrier.",
                  carrierName,
                  counterRate,
                )}
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((values) => onConfirm(values))(event);
          }}
        >
          {!approve && (
            <div className="pb-4">
              <TextareaField
                control={control}
                name="declineReason"
                label={t("Reason")}
                rules={{ required: true }}
                placeholder={t("e.g., Above what the customer pays on this lane")}
              />
            </div>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button
              type="submit"
              variant={approve ? "default" : "destructive"}
              isLoading={isSubmitting}
              loadingText={t("Recording...")}
            >
              {approve ? t("Take counter") : t("Refuse counter")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

function CounterSummary({ offer }: { offer: LiveTenderOffer }) {
  const t = useT();

  if (!offer.counterRate) return null;
  const margin = formatCounterMargin(offer.counterMarginPercent);

  return (
    <div className="flex flex-col gap-0.5 text-2xs">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
        <span className="text-foreground tabular-nums">
          {offer.status === "CounterRejected"
            ? t("Counter of {0} refused", formatOfferRate(offer.counterRate, offer.rateMethod))
            : t("Countered at {0}", formatOfferRate(offer.counterRate, offer.rateMethod))}
        </span>
        {margin && (
          <span className="text-muted-foreground tabular-nums">{t("· margin {0}", margin)}</span>
        )}
        {offer.status === "Countered" && offer.counterNeedsApproval && (
          <Badge variant="warning" className="h-4 rounded-badge px-1.5 text-3xs">
            {t("Needs approval")}
          </Badge>
        )}
      </div>
      {offer.counterNote && (
        <span className="text-muted-foreground italic">“{offer.counterNote}”</span>
      )}
    </div>
  );
}

function OfferRow({
  offer,
  isCurrent,
  nowSeconds,
  onRecordResponse,
  onDecideCounter,
  canApproveCounters,
  onBounce,
}: {
  offer: LiveTenderOffer;
  isCurrent: boolean;
  nowSeconds: number;
  onRecordResponse: ((offer: LiveTenderOffer) => void) | null;
  onDecideCounter: ((decision: CounterDecision) => void) | null;
  canApproveCounters: boolean;
  onBounce: ((offer: LiveTenderOffer) => void) | null;
}) {
  const t = useT();

  const countdown =
    offer.status === "Sent" || offer.status === "Countered"
      ? formatOfferCountdown(offer.expiresAt, nowSeconds)
      : "";
  const canTakeCounter = !offer.counterNeedsApproval || canApproveCounters;
  const tookCounter =
    offer.status === "Accepted" && !!offer.counterRate && offer.agreedRate !== offer.rate;

  return (
    <div
      className={cn(
        "flex flex-col gap-1 rounded-md border p-2",
        isCurrent ? "border-brand/50 bg-brand/5" : "bg-card",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <Badge
            variant="neutral"
            appearance="outline"
            className="h-4 shrink-0 rounded-md px-1 text-3xs tabular-nums"
          >
            #{offer.rank}
          </Badge>
          <span className="truncate text-xs font-medium">
            {offer.carrier?.name ?? t("Unknown carrier")}
          </span>
        </div>
        <TenderOfferStatusBadge status={offer.status} className="shrink-0 text-3xs" />
      </div>

      <div className="text-muted-foreground flex flex-wrap items-center gap-x-2 gap-y-0.5 text-2xs">
        <span className="tabular-nums">{formatOfferRate(offer.rate, offer.rateMethod)}</span>
        <span>· {TENDER_CHANNEL_LABEL[offer.channel]}</span>
        {offer.reofferedFromId && <span>{t("· asked again")}</span>}
        {countdown && (
          <span
            className={cn("tabular-nums", countdown === "expired" && "text-danger-foreground")}
            title={offer.status === "Countered" ? t("Time left to decide the counter") : undefined}
          >
            · {countdown}
          </span>
        )}
        {offer.respondedAt != null && (
          <span>{t("· responded {0}", formatUnixDateTime(offer.respondedAt))}</span>
        )}
        {offer.respondedAt != null && offer.responseSource && (
          <span title={t("How the carrier's response reached us")}>
            · {TENDER_RESPONSE_SOURCE_LABEL[offer.responseSource]}
          </span>
        )}
      </div>

      {(offer.status === "Countered" || offer.status === "CounterRejected") && (
        <CounterSummary offer={offer} />
      )}
      {tookCounter && (
        <span className="text-2xs tabular-nums">
          {t(
            "Agreed at {0} after a counter-offer",
            formatOfferRate(offer.agreedRate, offer.rateMethod),
          )}
        </span>
      )}

      {(offer.status === "Declined" || offer.status === "CounterRejected") &&
        offer.declineReason && (
          <span className="text-muted-foreground text-2xs italic">“{offer.declineReason}”</span>
        )}
      {offer.status === "Revoked" && (
        <span className="text-muted-foreground text-2xs">
          {offer.revokeReason
            ? t("Backed out: “{0}”", offer.revokeReason)
            : t("Backed out after accepting")}
        </span>
      )}
      {offer.status === "DeliveryFailed" && offer.deliveryError && (
        <span className="text-2xs text-danger-foreground">{offer.deliveryError}</span>
      )}

      {offer.status === "Sent" && onRecordResponse && (
        <div>
          <Button
            size="sm"
            variant="outline"
            className="h-6 px-2 text-2xs"
            onClick={() => onRecordResponse(offer)}
          >
            {t("Record response")}
          </Button>
        </div>
      )}
      {offer.status === "Countered" && onDecideCounter && (
        <div className="flex flex-wrap items-center gap-1.5">
          {canTakeCounter ? (
            <Button
              size="sm"
              variant="outline"
              className="h-6 px-2 text-2xs"
              onClick={() => onDecideCounter({ offer, approve: true })}
            >
              {t("Take counter")}
            </Button>
          ) : (
            <span className="text-muted-foreground text-2xs">
              {t("Below the margin floor; someone who can approve tenders has to take it.")}
            </span>
          )}
          <Button
            size="sm"
            variant="outline"
            className="h-6 px-2 text-2xs"
            onClick={() => onDecideCounter({ offer, approve: false })}
          >
            {t("Refuse counter")}
          </Button>
        </div>
      )}
      {offer.status === "Accepted" && onBounce && (
        <div>
          <Button
            size="sm"
            variant="outline"
            className="text-destructive hover:text-destructive h-6 px-2 text-2xs"
            onClick={() => onBounce(offer)}
          >
            {t("Carrier backed out")}
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
  onBounceTender,
  onAssignManually,
}: {
  tender: LiveTender;
  isTendering: boolean;
  onCancelTender: (params: { tenderId: string; reason: string }) => Promise<unknown>;
  onRecordResponse: (params: RecordOfferResponseParams) => Promise<unknown>;
  /** Records that the accepting carrier backed out; offered while the tender is in review. */
  onBounceTender?: (params: { tenderId: string; payload: BounceTenderPayload }) => Promise<unknown>;
  /** Opens the manual carrier-assign flow; shown on the NeedsReview banner. */
  onAssignManually?: () => void;
}) {
  const t = useT();

  const nowSeconds = useNowSeconds();
  const [cancelOpen, setCancelOpen] = useState(false);
  const [respondingTo, setRespondingTo] = useState<LiveTenderOffer | null>(null);
  const [deciding, setDeciding] = useState<CounterDecision | null>(null);
  const [bouncing, setBouncing] = useState<LiveTenderOffer | null>(null);

  // OfferRow repeats per offer, so the checks are resolved once here and passed
  // down. They mirror the API's own gates — tender:update for recording an
  // off-channel response, tender:cancel for pulling the tender back, and
  // shipment_move:assign for the manual-assign escape hatch — so a user who
  // would only earn a 403 never sees the control.
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const canRecordResponse = hasPermission(Resource.Tender, Operation.Update);
  const canCancelTender = hasPermission(Resource.Tender, Operation.Cancel);
  const canAssignMove = hasPermission(Resource.ShipmentMove, Operation.Assign);
  const canApproveCounters = hasPermission(Resource.Tender, Operation.Approve);

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

  const handleDecideCounter = useCallback(
    (payload: RecordTenderResponsePayload) => {
      if (!deciding) return;
      void onRecordResponse({ offerId: deciding.offer.id, payload })
        .then(() => setDeciding(null))
        .catch(() => {
          // handled by the mutation's onError; the dialog stays open for a retry
        });
    },
    [onRecordResponse, deciding],
  );

  const handleBounce = useCallback(
    (payload: BounceTenderPayload) => {
      if (!onBounceTender) return;
      void onBounceTender({ tenderId: tender.id, payload })
        .then(() => setBouncing(null))
        .catch(() => {
          // handled by the mutation's onError; the dialog stays open for a retry
        });
    },
    [onBounceTender, tender.id],
  );

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-1.5">
        <TenderStatusBadge status={tender.status} />
        <Badge variant="neutral" appearance="outline" className="h-4 rounded-md px-1 text-3xs">
          {TENDER_MODE_LABEL[tender.mode]}
        </Badge>
        {tender.routingGuide && (
          <span className="text-muted-foreground text-2xs">
            {t("via {0}", tender.routingGuide.name)}
          </span>
        )}
        {tender.escalationRound > 0 && (
          <Badge variant="warning" appearance="outline" className="h-4 rounded-md px-1 text-3xs">
            {t("Escalation round {0}", tender.escalationRound)}
          </Badge>
        )}
        {tender.bounceCount > 0 && (
          <span className="text-muted-foreground text-2xs">
            {t(
              "{0, plural, one {· a carrier backed out once} other {· carriers backed out # times}}",
              tender.bounceCount,
            )}
          </span>
        )}
      </div>

      {tender.status === "NeedsReview" && (
        <Alert variant="destructive">
          <AlertTriangleIcon className="size-4" aria-hidden />
          <AlertTitle>{t("Carrier accepted, but auto-assignment failed")}</AlertTitle>
          <AlertDescription>
            {t(
              "The accepting carrier could not be assigned to the move automatically — the move may have gained coverage in the meantime, or the assignment was blocked. Assign the carrier manually with the regular carrier-assign flow, or cancel this tender.",
            )}
            {onAssignManually && canAssignMove && (
              <Button
                size="sm"
                variant="outline"
                className="mt-2 h-6 px-2 text-2xs"
                onClick={onAssignManually}
              >
                {t("Assign manually")}
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
            onRecordResponse={isLive && canRecordResponse ? setRespondingTo : null}
            onDecideCounter={isLive && canRecordResponse ? setDeciding : null}
            canApproveCounters={canApproveCounters}
            onBounce={
              tender.status === "NeedsReview" &&
              offer.id === tender.acceptedOfferId &&
              onBounceTender &&
              canRecordResponse
                ? setBouncing
                : null
            }
          />
        ))}
        {offers.length === 0 && (
          <p className="text-muted-foreground py-3 text-center text-xs">
            {t("No offers on this tender.")}
          </p>
        )}
      </div>

      {isLive && canCancelTender && (
        <div>
          <Button
            size="sm"
            variant="outline"
            className="text-destructive hover:text-destructive h-6 px-2 text-2xs"
            disabled={isTendering}
            onClick={() => setCancelOpen(true)}
          >
            {t("Cancel tender")}
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
      {deciding && (
        <CounterDecisionDialog
          decision={deciding}
          onOpenChange={(open) => {
            if (!open) setDeciding(null);
          }}
          isSubmitting={isTendering}
          onConfirm={handleDecideCounter}
        />
      )}
      {bouncing && (
        <TenderBounceDialog
          carrierName={bouncing.carrier?.name ?? null}
          onOpenChange={(open) => {
            if (!open) setBouncing(null);
          }}
          isSubmitting={isTendering}
          onConfirm={handleBounce}
        />
      )}
    </div>
  );
}
