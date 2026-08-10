import logoRainbow from "@/assets/logo.webp";
import { LazyImage } from "@/components/image";
import { Metadata } from "@/components/metadata";
import { TenderService } from "@/services/tender";
import { ApiRequestError } from "@trenova/shared/lib/api";
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

type OfferErrorKind = "invalid" | "throttled" | "unavailable";

/**
 * Only a definitive 4xx rejection means the link itself is dead. A network
 * failure, a 5xx, or anything unrecognized is a transient server-side problem
 * and must not be presented as an expired offer.
 */
function offerErrorKind(error: unknown): OfferErrorKind {
  if (error instanceof ApiRequestError) {
    if (error.status === 429) return "throttled";
    if (error.status === 422 || error.isValidationError()) return "invalid";
    if (error.status >= 400 && error.status < 500) return "invalid";
  }
  return "unavailable";
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  if (!value) return null;
  return (
    <div className="flex items-start justify-between gap-4 py-1.5">
      <span className="shrink-0 text-xs text-muted-foreground">{label}</span>
      <span className="text-right text-xs font-medium">{value}</span>
    </div>
  );
}

function StatusCard({
  icon,
  title,
  body,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
  action?: React.ReactNode;
}) {
  return (
    <Card className="w-full max-w-md">
      <CardContent className="flex flex-col items-center gap-2 py-8 text-center">
        {icon}
        <p className="text-sm font-semibold">{title}</p>
        <p className="text-xs text-muted-foreground">{body}</p>
        {action}
      </CardContent>
    </Card>
  );
}

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
        <p className="text-xs text-muted-foreground">
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
          <span className="text-xs text-muted-foreground">Rate</span>
          <span className="text-sm font-semibold tabular-nums">
            {offer.rateAmount}
            <span className="ml-1 text-xs font-normal text-muted-foreground">
              {offer.rateMethodLabel}
            </span>
          </span>
        </div>

        {offer.expiresAt > 0 && (
          <div className="flex items-center gap-1.5 rounded-md border bg-muted/40 px-2 py-1.5">
            <ClockIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            <span className="text-xs text-muted-foreground">
              Offer expires {formatUnixDateTime(offer.expiresAt)}
            </span>
          </div>
        )}

        <div className="flex flex-col gap-2 pt-1">
          <div className="flex gap-2">
            <Button
              type="button"
              className={cn("flex-1", intent === "accept" && "ring-2 ring-ring ring-offset-2")}
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
                intent === "decline" && !declineOpen && "ring-2 ring-ring ring-offset-2",
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
              <label htmlFor="decline-reason" className="text-xs text-muted-foreground">
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
  const [submitError, setSubmitError] = useState<OfferErrorKind | null>(null);

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
      setSubmitError(offerErrorKind(error));
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
        icon={<ClockIcon className="size-8 text-muted-foreground" aria-hidden />}
        title="Too many attempts"
        body="Please wait a minute and try the link from your email again."
      />
    );
  } else if (submitError === "unavailable") {
    content = (
      <StatusCard
        icon={<TriangleAlertIcon className="size-8 text-muted-foreground" aria-hidden />}
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
        icon={<CircleSlashIcon className="size-8 text-muted-foreground" aria-hidden />}
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
    const kind = offerErrorKind(previewQuery.error);
    content =
      kind === "throttled" ? (
        <StatusCard
          icon={<ClockIcon className="size-8 text-muted-foreground" aria-hidden />}
          title="Too many attempts"
          body="Please wait a minute and try the link from your email again."
        />
      ) : kind === "unavailable" ? (
        <StatusCard
          icon={<TriangleAlertIcon className="size-8 text-muted-foreground" aria-hidden />}
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
          icon={<CircleSlashIcon className="size-8 text-muted-foreground" aria-hidden />}
          title="This offer link is no longer valid"
          body="The offer may have expired, been withdrawn, or already been answered. Contact the broker if you believe this is an error."
        />
      );
  } else if (previewQuery.data?.responded) {
    content = (
      <StatusCard
        icon={<CheckCircle2Icon className="size-8 text-muted-foreground" aria-hidden />}
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
        icon={<CircleSlashIcon className="size-8 text-muted-foreground" aria-hidden />}
        title="This offer link is no longer valid"
        body="The offer may have expired, been withdrawn, or already been answered. Contact the broker if you believe this is an error."
      />
    );
  }

  return (
    <>
      <Metadata title="Load Offer" description="Review and respond to a load offer" />
      <div className="fixed inset-0 h-svh w-full overflow-y-auto bg-background">
        <div className="flex min-h-full flex-col items-center justify-center gap-6 p-6 md:p-10">
          <LazyImage src={logoRainbow} alt="Trenova Logo" className="size-12 object-contain" />
          {content}
          <p className="text-center text-[11px] text-muted-foreground">
            Powered by Trenova. Questions about this load? Reply to the offer email.
          </p>
        </div>
      </div>
    </>
  );
}
