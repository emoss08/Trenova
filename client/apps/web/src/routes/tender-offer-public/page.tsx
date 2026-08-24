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
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { PublicTenderOffer } from "@trenova/shared/types/tender";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckCircle2Icon, CircleSlashIcon, ClockIcon, TriangleAlertIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useLocation, useParams } from "react-router";

const tenderService = new TenderService();

type OfferIntent = "accept" | "decline" | null;

function OfferSummary({
  offer,
  intent,
  isSubmitting,
  onAccept,
  onDecline,
}: {
  offer: PublicTenderOffer;
  intent: OfferIntent;
  isSubmitting: boolean;
  onAccept: () => void;
  onDecline: (reason: string) => void;
}) {
  const [declineOpen, setDeclineOpen] = useState(intent === "decline");
  const [reason, setReason] = useState("");

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle className="text-base">Load offer for {offer.carrierName}</CardTitle>
        <p className="text-muted-foreground text-xs">
          PRO {offer.shipmentProNumber} · respond before the offer expires
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-col">
          <SummaryRow label="Origin" value={offer.originSummary} />
          <SummaryRow label="Destination" value={offer.destinationSummary} />
          <SummaryRow label="Pickup" value={offer.pickupWindow} />
          <SummaryRow label="Delivery" value={offer.deliveryWindow} />
          <SummaryRow label="Equipment" value={offer.equipmentSummary} />
          <SummaryRow label="Weight" value={offer.weightSummary} />
        </div>

        <Separator />

        <div className="flex items-center justify-between">
          <span className="text-muted-foreground text-xs">Rate</span>
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
              Offer expires {formatUnixDateTime(offer.expiresAt)}
            </span>
          </div>
        )}

        <div className="flex flex-col gap-2 pt-1">
          <div className="flex gap-2">
            <Button
              type="button"
              className={cn("flex-1", intent === "accept" && "ring-ring ring-2 ring-offset-2")}
              isLoading={isSubmitting && !declineOpen}
              loadingText="Accepting..."
              disabled={isSubmitting}
              onClick={onAccept}
            >
              Accept load
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
              loadingText="Declining..."
              onClick={() => {
                if (!declineOpen) {
                  setDeclineOpen(true);
                  return;
                }
                onDecline(reason);
              }}
            >
              {declineOpen ? "Confirm decline" : "Decline"}
            </Button>
          </div>

          {declineOpen && (
            <div className="flex flex-col gap-1">
              <label htmlFor="decline-reason" className="text-muted-foreground text-xs">
                Reason (optional)
              </label>
              <Textarea
                id="decline-reason"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="e.g., No truck available in the area"
                rows={3}
                maxLength={500}
              />
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * The public accept/decline page an external carrier lands on from an emailed
 * offer link. It never auto-submits — the /accept and /decline variants only
 * pre-highlight the matching button — and it leaks nothing about why a link is
 * unusable.
 */
export function TenderOfferPublicPage() {
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
    mutationFn: (params: { action: "accept" | "decline"; reason: string }) =>
      params.action === "accept"
        ? tenderService.acceptPublicOffer(token)
        : tenderService.declinePublicOffer(token, params.reason),
    onSuccess: () => {
      setSubmitError(null);
      setSubmitted(true);
    },
    onError: (error: unknown) => {
      setSubmitError(publicLinkErrorKind(error));
    },
  });

  let content: React.ReactNode;
  if (submitted) {
    content = (
      <StatusCard
        icon={<CheckCircle2Icon className="size-8 text-green-600" aria-hidden />}
        title="Response recorded"
        body="Thank you — the dispatcher has been notified of your response."
      />
    );
  } else if (submitError === "throttled") {
    content = (
      <StatusCard
        icon={<ClockIcon className="text-muted-foreground size-8" aria-hidden />}
        title="Too many attempts"
        body="Please wait a minute and try the link from your email again."
      />
    );
  } else if (submitError === "unavailable") {
    content = (
      <StatusCard
        icon={<TriangleAlertIcon className="text-muted-foreground size-8" aria-hidden />}
        title="Temporarily unavailable"
        body="Your response could not be recorded because of a temporary problem. Nothing has been submitted — please try again in a moment."
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
            Try again
          </Button>
        }
      />
    );
  } else if (submitError === "invalid") {
    content = (
      <StatusCard
        icon={<CircleSlashIcon className="text-muted-foreground size-8" aria-hidden />}
        title="This offer link is no longer valid"
        body="The offer may have expired, been withdrawn, or already been answered. Contact the broker if you believe this is an error."
      />
    );
  } else if (previewQuery.isLoading) {
    content = (
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col gap-3 py-6">
          <Skeleton className="h-5 w-48 rounded" />
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
          title="Too many attempts"
          body="Please wait a minute and try the link from your email again."
        />
      ) : kind === "unavailable" ? (
        <StatusCard
          icon={<TriangleAlertIcon className="text-muted-foreground size-8" aria-hidden />}
          title="Temporarily unavailable"
          body="The offer could not be loaded because of a temporary problem. Please try again in a moment."
          action={
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="mt-2"
              isLoading={previewQuery.isRefetching}
              loadingText="Retrying..."
              onClick={() => void previewQuery.refetch()}
            >
              Try again
            </Button>
          }
        />
      ) : (
        <StatusCard
          icon={<CircleSlashIcon className="text-muted-foreground size-8" aria-hidden />}
          title="This offer link is no longer valid"
          body="The offer may have expired, been withdrawn, or already been answered. Contact the broker if you believe this is an error."
        />
      );
  } else if (previewQuery.data?.responded) {
    content = (
      <StatusCard
        icon={<CheckCircle2Icon className="text-muted-foreground size-8" aria-hidden />}
        title="Already answered"
        body="A response has already been recorded for this offer. Contact the broker if anything changed."
      />
    );
  } else if (previewQuery.data) {
    content = (
      <OfferSummary
        offer={previewQuery.data}
        intent={intent}
        isSubmitting={respondMutation.isPending}
        onAccept={() => respondMutation.mutate({ action: "accept", reason: "" })}
        onDecline={(reason) => respondMutation.mutate({ action: "decline", reason })}
      />
    );
  } else {
    content = (
      <StatusCard
        icon={<CircleSlashIcon className="text-muted-foreground size-8" aria-hidden />}
        title="This offer link is no longer valid"
        body="The offer may have expired, been withdrawn, or already been answered. Contact the broker if you believe this is an error."
      />
    );
  }

  return (
    <>
      <Metadata title="Load Offer" description="Review and respond to a load offer" />
      <PublicPageShell footer="Powered by Trenova. Questions about this load? Reply to the offer email.">
        {content}
      </PublicPageShell>
    </>
  );
}
