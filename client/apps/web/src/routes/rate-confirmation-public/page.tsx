import { Metadata } from "@/components/metadata";
import { publicLinkErrorKind, type PublicLinkErrorKind } from "@/components/public-page/error-kind";
import { PublicPageShell } from "@/components/public-page/public-page-shell";
import { StatusCard } from "@/components/public-page/status-card";
import { SummaryRow } from "@/components/public-page/summary-row";
import { RateConfirmationService } from "@/services/rate-confirmation";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Input } from "@trenova/shared/components/ui/input";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import type {
  PublicRateConfirmation,
  PublicRateConfirmationStop,
} from "@trenova/shared/types/rate-confirmation";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckCircle2Icon, CircleSlashIcon, ClockIcon, TriangleAlertIcon } from "lucide-react";
import { useState } from "react";
import { useParams } from "react-router";

const rateConfirmationService = new RateConfirmationService();

type SignerPayload = { signerName: string; signerTitle: string };

function stopSummary(stop: PublicRateConfirmationStop): string {
  return [stop.name, stop.address, stop.window].filter(Boolean).join(" · ");
}

function RateConfirmationSummary({
  rateConfirmation,
  isSubmitting,
  onConfirm,
}: {
  rateConfirmation: PublicRateConfirmation;
  isSubmitting: boolean;
  onConfirm: (payload: SignerPayload) => void;
}) {
  const [signerName, setSignerName] = useState("");
  const [signerTitle, setSignerTitle] = useState("");

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle className="text-base">
          Rate confirmation for {rateConfirmation.carrierName}
        </CardTitle>
        <p className="text-xs text-muted-foreground">
          {[
            rateConfirmation.shipmentProNumber && `PRO ${rateConfirmation.shipmentProNumber}`,
            rateConfirmation.revisionLabel,
          ]
            .filter(Boolean)
            .join(" · ") || "Review and sign the agreement below"}
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-col">
          <SummaryRow label="Broker" value={rateConfirmation.companyName} />
          {rateConfirmation.stops.map((stop) => (
            <SummaryRow
              key={`${stop.sequence}-${stop.type}`}
              label={`Stop ${stop.sequence} · ${stop.type}`}
              value={stopSummary(stop)}
            />
          ))}
        </div>

        <Separator />

        <div className="flex flex-col">
          <SummaryRow label="Rate method" value={rateConfirmation.rateMethodLabel} />
          <SummaryRow label="Base rate" value={rateConfirmation.baseRateLabel} />
          <SummaryRow label="Payment terms" value={rateConfirmation.paymentTermsLabel} />
        </div>

        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">Total</span>
          <span className="text-sm font-semibold tabular-nums">
            {rateConfirmation.totalCost}
            <span className="ml-1 text-xs font-normal text-muted-foreground">
              {rateConfirmation.currency}
            </span>
          </span>
        </div>

        {rateConfirmation.expiresAt > 0 && (
          <div className="flex items-center gap-1.5 rounded-md border bg-muted/40 px-2 py-1.5">
            <ClockIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            <span className="text-xs text-muted-foreground">
              Sign link expires {formatUnixDateTime(rateConfirmation.expiresAt)}
            </span>
          </div>
        )}

        <div className="flex flex-col gap-2 pt-1">
          <div className="flex flex-col gap-1">
            <label htmlFor="signer-name" className="text-xs text-muted-foreground">
              Your name (required)
            </label>
            <Input
              id="signer-name"
              value={signerName}
              onChange={(event) => setSignerName(event.target.value)}
              placeholder="e.g., Jane Smith"
              maxLength={255}
              autoComplete="name"
            />
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="signer-title" className="text-xs text-muted-foreground">
              Title (optional)
            </label>
            <Input
              id="signer-title"
              value={signerTitle}
              onChange={(event) => setSignerTitle(event.target.value)}
              placeholder="e.g., Operations Manager"
              maxLength={255}
              autoComplete="organization-title"
            />
          </div>
          <Button
            type="button"
            className="mt-1"
            isLoading={isSubmitting}
            loadingText="Confirming..."
            disabled={isSubmitting || !signerName.trim()}
            onClick={() =>
              onConfirm({ signerName: signerName.trim(), signerTitle: signerTitle.trim() })
            }
          >
            Confirm rate
          </Button>
          <p className="text-[11px] text-muted-foreground">
            By confirming, you agree to the rate and terms shown above on behalf of{" "}
            {rateConfirmation.carrierName}.
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

function InvalidLinkCard() {
  return (
    <StatusCard
      icon={<CircleSlashIcon className="size-8 text-muted-foreground" aria-hidden />}
      title="This rate confirmation link is no longer valid"
      body="The link may have expired, been revoked, or the rate confirmation may have been updated. Contact the broker if you believe this is an error."
    />
  );
}

function ThrottledCard() {
  return (
    <StatusCard
      icon={<ClockIcon className="size-8 text-muted-foreground" aria-hidden />}
      title="Too many attempts"
      body="Please wait a minute and try the link from your email again."
    />
  );
}

/**
 * The public signing page a carrier lands on from an emailed rate confirmation
 * link. The preview GET never mutates — email scanners prefetch links — and the
 * page leaks nothing about why a link is unusable.
 */
export function RateConfirmationPublicPage() {
  const { token = "" } = useParams();

  const [submitted, setSubmitted] = useState(false);
  const [submitError, setSubmitError] = useState<PublicLinkErrorKind | null>(null);

  const previewQuery = useQuery({
    queryKey: ["public-rate-confirmation", token] as const,
    queryFn: () => rateConfirmationService.getPublicRateConfirmation(token),
    retry: false,
  });

  const confirmMutation = useMutation({
    mutationFn: (payload: SignerPayload) =>
      rateConfirmationService.confirmPublicRateConfirmation(token, payload),
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
        title="Rate confirmed"
        body="Thank you — your signature has been recorded and the broker has been notified."
      />
    );
  } else if (submitError === "throttled") {
    content = <ThrottledCard />;
  } else if (submitError === "unavailable") {
    content = (
      <StatusCard
        icon={<TriangleAlertIcon className="size-8 text-muted-foreground" aria-hidden />}
        title="Temporarily unavailable"
        body="Your signature could not be recorded because of a temporary problem. Nothing has been submitted — please try again in a moment."
        action={
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => {
              const lastSigner = confirmMutation.variables;
              setSubmitError(null);
              if (lastSigner) {
                confirmMutation.mutate(lastSigner);
              }
            }}
          >
            Try again
          </Button>
        }
      />
    );
  } else if (submitError === "invalid") {
    content = <InvalidLinkCard />;
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
        <ThrottledCard />
      ) : kind === "unavailable" ? (
        <StatusCard
          icon={<TriangleAlertIcon className="size-8 text-muted-foreground" aria-hidden />}
          title="Temporarily unavailable"
          body="The rate confirmation could not be loaded because of a temporary problem. Please try again in a moment."
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
        <InvalidLinkCard />
      );
  } else if (previewQuery.data?.confirmed) {
    content = (
      <StatusCard
        icon={<CheckCircle2Icon className="size-8 text-muted-foreground" aria-hidden />}
        title="Already confirmed"
        body="This rate confirmation has already been signed. Contact the broker if anything changed."
      />
    );
  } else if (previewQuery.data) {
    content = (
      <RateConfirmationSummary
        rateConfirmation={previewQuery.data}
        isSubmitting={confirmMutation.isPending}
        onConfirm={(payload) => confirmMutation.mutate(payload)}
      />
    );
  } else {
    content = <InvalidLinkCard />;
  }

  return (
    <>
      <Metadata title="Rate Confirmation" description="Review and sign a rate confirmation" />
      <PublicPageShell footer="Powered by Trenova. Questions about this rate confirmation? Reply to the email it arrived in.">
        {content}
      </PublicPageShell>
    </>
  );
}
