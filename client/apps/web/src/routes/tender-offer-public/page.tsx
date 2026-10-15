import { useT } from "@trenova/shared/i18n/use-t";
import { Metadata } from "@/components/metadata";
import { publicLinkErrorKind, type PublicLinkErrorKind } from "@/components/public-page/error-kind";
import { PublicPageShell } from "@/components/public-page/public-page-shell";
import { StatusCard } from "@/components/public-page/status-card";
import { SummaryRow } from "@/components/public-page/summary-row";
import { TenderService } from "@/services/tender";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Input } from "@trenova/shared/components/ui/input";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  MAX_COUNTER_NOTE_LENGTH,
  publicTenderCounterPayloadSchema,
  type PublicTenderCounterPayload,
  type PublicTenderOffer,
} from "@trenova/shared/types/tender";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  ClockIcon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { useLocation, useParams } from "react-router";

const tenderService = new TenderService();

type OfferIntent = "accept" | "decline" | null;

type OfferPanel = "decline" | "counter" | null;

type OfferResponse =
  | { action: "accept" }
  | { action: "decline"; reason: string }
  | { action: "counter"; counter: PublicTenderCounterPayload };

function CounterForm({
  offer,
  isSubmitting,
  onCounter,
}: {
  offer: PublicTenderOffer;
  isSubmitting: boolean;
  onCounter: (counter: PublicTenderCounterPayload) => void;
}) {
  const t = useT();

  const [rate, setRate] = useState("");
  const [note, setNote] = useState("");
  const [rateError, setRateError] = useState<string | null>(null);

  const submit = () => {
    const parsed = publicTenderCounterPayloadSchema.safeParse({ rate, note });
    if (!parsed.success) {
      setRateError(parsed.error.issues[0]?.message ?? t("Enter a valid rate"));
      return;
    }
    setRateError(null);
    onCounter(parsed.data);
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-1">
        <label htmlFor="counter-rate" className="text-muted-foreground text-xs">
          {t("Your rate ({0})", offer.rateMethodLabel)}
        </label>
        <Input
          id="counter-rate"
          inputMode="decimal"
          autoComplete="off"
          value={rate}
          onChange={(event) => setRate(event.target.value)}
          placeholder={offer.rateValue || undefined}
          aria-invalid={rateError !== null}
          aria-describedby={rateError ? "counter-rate-error" : undefined}
        />
        {rateError && (
          <span id="counter-rate-error" role="alert" className="text-2xs text-danger-foreground">
            {rateError}
          </span>
        )}
      </div>
      <div className="flex flex-col gap-1">
        <label htmlFor="counter-note" className="text-muted-foreground text-xs">
          {t("Note to the broker (optional)")}
        </label>
        <Textarea
          id="counter-note"
          value={note}
          onChange={(event) => setNote(event.target.value)}
          placeholder={t("e.g., Fuel is up on this lane")}
          rows={2}
          maxLength={MAX_COUNTER_NOTE_LENGTH}
        />
      </div>
      <Button
        type="button"
        isLoading={isSubmitting}
        loadingText={t("Sending...")}
        disabled={isSubmitting}
        onClick={submit}
      >
        {t("Send counter-offer")}
      </Button>
    </div>
  );
}

function OfferSummary({
  offer,
  intent,
  isSubmitting,
  onRespond,
}: {
  offer: PublicTenderOffer;
  intent: OfferIntent;
  isSubmitting: boolean;
  onRespond: (response: OfferResponse) => void;
}) {
  const t = useT();

  const [panel, setPanel] = useState<OfferPanel>(intent === "decline" ? "decline" : null);
  const [reason, setReason] = useState("");
  const declineOpen = panel === "decline";

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle className="text-base">{t("Load offer for {0}", offer.carrierName)}</CardTitle>
        <p className="text-muted-foreground text-xs">
          {t("PRO {0} · respond before the offer expires", offer.shipmentProNumber)}
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-col">
          <SummaryRow label={t("Origin")} value={offer.originSummary} />
          <SummaryRow label={t("Destination")} value={offer.destinationSummary} />
          <SummaryRow label={t("Pickup")} value={offer.pickupWindow} />
          <SummaryRow label={t("Delivery")} value={offer.deliveryWindow} />
          <SummaryRow label={t("Equipment")} value={offer.equipmentSummary} />
          <SummaryRow label={t("Weight")} value={offer.weightSummary} />
        </div>

        <Separator />

        <div className="flex items-center justify-between">
          <span className="text-muted-foreground text-xs">{t("Rate")}</span>
          <span className="text-sm font-semibold tabular-nums">
            {offer.rateAmount}
            <span className="text-muted-foreground ml-1 text-xs font-normal">
              {offer.rateMethodLabel}
            </span>
          </span>
        </div>

        {offer.expiresAt > 0 && (
          <div className="bg-muted/40 flex items-center gap-1.5 rounded-md border px-2 py-1.5">
            <ClockIcon className="text-muted-foreground size-3.5 shrink-0" aria-hidden />
            <span className="text-muted-foreground text-xs">
              {t("Offer expires {0}", formatUnixDateTime(offer.expiresAt))}
            </span>
          </div>
        )}

        <div className="flex flex-col gap-2 pt-1">
          <div className="flex gap-2">
            <Button
              type="button"
              className={cn("flex-1", intent === "accept" && "ring-ring ring-2 ring-offset-2")}
              isLoading={isSubmitting && panel === null}
              loadingText={t("Accepting...")}
              disabled={isSubmitting}
              onClick={() => onRespond({ action: "accept" })}
            >
              {t("Accept load")}
            </Button>
            <Button
              type="button"
              variant={declineOpen ? "destructive" : "outline"}
              className={cn(
                "flex-1",
                intent === "decline" && !declineOpen && "ring-ring ring-2 ring-offset-2",
              )}
              disabled={isSubmitting}
              isLoading={isSubmitting && declineOpen}
              loadingText={t("Declining...")}
              onClick={() => {
                if (!declineOpen) {
                  setPanel("decline");
                  return;
                }
                onRespond({ action: "decline", reason });
              }}
            >
              {declineOpen ? t("Confirm decline") : t("Decline")}
            </Button>
          </div>

          {offer.counterAllowed && panel !== "counter" && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={isSubmitting}
              onClick={() => setPanel("counter")}
            >
              {t("Propose a different rate")}
            </Button>
          )}

          {declineOpen && (
            <div className="flex flex-col gap-1">
              <label htmlFor="decline-reason" className="text-muted-foreground text-xs">
                {t("Reason (optional)")}
              </label>
              <Textarea
                id="decline-reason"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder={t("e.g., No truck available in the area")}
                rows={3}
                maxLength={500}
              />
            </div>
          )}

          {offer.counterAllowed && panel === "counter" && (
            <CounterForm
              offer={offer}
              isSubmitting={isSubmitting}
              onCounter={(counter) => onRespond({ action: "counter", counter })}
            />
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function sendResponse(token: string, response: OfferResponse): Promise<void> {
  switch (response.action) {
    case "accept":
      return tenderService.acceptPublicOffer(token);
    case "decline":
      return tenderService.declinePublicOffer(token, response.reason);
    case "counter":
      return tenderService.counterPublicOffer(token, response.counter);
  }
}

/**
 * The public accept/decline page an external carrier lands on from an emailed
 * offer link. When the broker takes counter-offers the carrier can also
 * propose a rate of its own. It never auto-submits — the /accept and /decline variants only
 * pre-highlight the matching button — and it leaks nothing about why a link is
 * unusable.
 */
export function TenderOfferPublicPage() {
  const t = useT();

  const { token = "" } = useParams();
  const { pathname } = useLocation();

  const intent: OfferIntent = useMemo(() => {
    if (pathname.endsWith("/accept")) return "accept";
    if (pathname.endsWith("/decline")) return "decline";
    return null;
  }, [pathname]);

  const [submitted, setSubmitted] = useState(false);
  const [submitError, setSubmitError] = useState<PublicLinkErrorKind | null>(null);

  const previewQuery = useQuery({
    queryKey: ["public-tender-offer", token] as const,
    queryFn: () => tenderService.getPublicOffer(token),
    retry: false,
  });

  const respondMutation = useMutation({
    mutationFn: (response: OfferResponse) => sendResponse(token, response),
    onSuccess: () => {
      setSubmitError(null);
      setSubmitted(true);
    },
    onError: (error: unknown) => {
      setSubmitError(publicLinkErrorKind(error));
    },
  });

  let content: React.ReactNode;
  if (submitted && respondMutation.variables?.action === "counter") {
    content = (
      <StatusCard
        icon={<CheckCircleIcon className="size-8 text-success-foreground" aria-hidden />}
        title={t("Counter-offer sent")}
        body={t(
          "The broker has your rate and will take or refuse it. You will hear back by email; the load is not yours until they confirm.",
        )}
      />
    );
  } else if (submitted) {
    content = (
      <StatusCard
        icon={<CheckCircleIcon className="size-8 text-success-foreground" aria-hidden />}
        title={t("Response recorded")}
        body={t("Thank you — the dispatcher has been notified of your response.")}
      />
    );
  } else if (submitError === "throttled") {
    content = (
      <StatusCard
        icon={<ClockIcon className="text-muted-foreground size-8" aria-hidden />}
        title={t("Too many attempts")}
        body={t("Please wait a minute and try the link from your email again.")}
      />
    );
  } else if (submitError === "unavailable") {
    content = (
      <StatusCard
        icon={<AlertTriangleIcon className="text-muted-foreground size-8" aria-hidden />}
        title={t("Temporarily unavailable")}
        body={t(
          "Your response could not be recorded because of a temporary problem. Nothing has been submitted — please try again in a moment.",
        )}
        action={
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => {
              const lastResponse = respondMutation.variables;
              setSubmitError(null);
              if (lastResponse) {
                respondMutation.mutate(lastResponse);
              }
            }}
          >
            {t("Try again")}
          </Button>
        }
      />
    );
  } else if (submitError === "invalid") {
    content = (
      <StatusCard
        icon={<SlashCircle01Icon className="text-muted-foreground size-8" aria-hidden />}
        title={t("This offer link is no longer valid")}
        body={t(
          "The offer may have expired, been withdrawn, or already been answered. Contact the broker if you believe this is an error.",
        )}
      />
    );
  } else if (previewQuery.isLoading) {
    content = (
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col gap-3 py-6">
          <Skeleton className="h-5 w-48 rounded-md" />
          <Skeleton className="h-24 rounded-md" />
          <Skeleton className="h-9 rounded-md" />
        </CardContent>
      </Card>
    );
  } else if (previewQuery.isError) {
    const kind = publicLinkErrorKind(previewQuery.error);
    content =
      kind === "throttled" ? (
        <StatusCard
          icon={<ClockIcon className="text-muted-foreground size-8" aria-hidden />}
          title={t("Too many attempts")}
          body={t("Please wait a minute and try the link from your email again.")}
        />
      ) : kind === "unavailable" ? (
        <StatusCard
          icon={<AlertTriangleIcon className="text-muted-foreground size-8" aria-hidden />}
          title={t("Temporarily unavailable")}
          body={t(
            "The offer could not be loaded because of a temporary problem. Please try again in a moment.",
          )}
          action={
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="mt-2"
              isLoading={previewQuery.isRefetching}
              loadingText={t("Retrying...")}
              onClick={() => void previewQuery.refetch()}
            >
              {t("Try again")}
            </Button>
          }
        />
      ) : (
        <StatusCard
          icon={<SlashCircle01Icon className="text-muted-foreground size-8" aria-hidden />}
          title={t("This offer link is no longer valid")}
          body={t(
            "The offer may have expired, been withdrawn, or already been answered. Contact the broker if you believe this is an error.",
          )}
        />
      );
  } else if (previewQuery.data?.responded) {
    content = (
      <StatusCard
        icon={<CheckCircleIcon className="text-muted-foreground size-8" aria-hidden />}
        title={t("Already answered")}
        body={t(
          "A response has already been recorded for this offer. Contact the broker if anything changed.",
        )}
      />
    );
  } else if (previewQuery.data) {
    content = (
      <OfferSummary
        offer={previewQuery.data}
        intent={intent}
        isSubmitting={respondMutation.isPending}
        onRespond={(response) => respondMutation.mutate(response)}
      />
    );
  } else {
    content = (
      <StatusCard
        icon={<SlashCircle01Icon className="text-muted-foreground size-8" aria-hidden />}
        title={t("This offer link is no longer valid")}
        body={t(
          "The offer may have expired, been withdrawn, or already been answered. Contact the broker if you believe this is an error.",
        )}
      />
    );
  }

  return (
    <>
      <Metadata title={t("Load offer")} description={t("Review and respond to a load offer")} />
      <PublicPageShell
        footer={t("Powered by Trenova. Questions about this load? Reply to the offer email.")}
      >
        {content}
      </PublicPageShell>
    </>
  );
}
