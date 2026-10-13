import { Metadata } from "@/components/metadata";
import { publicLinkErrorKind } from "@/components/public-page/error-kind";
import { PublicPageShell } from "@/components/public-page/public-page-shell";
import { StatusCard } from "@/components/public-page/status-card";
import {
  holdLabel,
  publicStatusAttrs,
  shows,
  stopPlace,
  stopStanding,
  stopWindow,
} from "@/lib/shipment-tracking";
import { ShipmentTrackingService } from "@/services/shipment-tracking";
import {
  AlertTriangleIcon,
  ClockIcon,
  Download01Icon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { stopTypeLabel } from "@trenova/shared/lib/stop-type";
import type { PublicShipmentView } from "@trenova/shared/types/shipment-tracking";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router";

const shipmentTrackingService = new ShipmentTrackingService();

function Details({ view }: { view: PublicShipmentView }) {
  const t = useT();
  const items: { label: string; value: string }[] = [];
  if (view.bol) {
    items.push({ label: t("BOL"), value: view.bol });
  }
  if (view.customerReference) {
    items.push({ label: t("Your reference"), value: view.customerReference });
  }
  if (view.carrierName) {
    items.push({ label: t("Carrier"), value: view.carrierName });
  }
  if (view.equipment) {
    items.push({ label: t("Equipment"), value: view.equipment });
  }
  if (view.pieces != null) {
    items.push({ label: t("Pieces"), value: view.pieces.toLocaleString() });
  }
  if (view.weight != null) {
    items.push({ label: t("Weight"), value: t("{0} lb", view.weight.toLocaleString()) });
  }
  if (view.lastLocation) {
    items.push({
      label: t("Last reported"),
      value: t(
        "{0}, {1} at {2}",
        view.lastLocation.city,
        view.lastLocation.stateCode,
        formatUnixDateTime(view.lastLocation.reportedAt),
      ),
    });
  }
  if (items.length === 0) {
    return null;
  }

  return (
    <DescriptionList layout="inline">
      {items.map((item) => (
        <DescriptionItem key={item.label} label={item.label}>
          {item.value}
        </DescriptionItem>
      ))}
    </DescriptionList>
  );
}

function Stops({ view }: { view: PublicShipmentView }) {
  const t = useT();

  return (
    <ol className="flex flex-col divide-y divide-border-subtle">
      {view.stops.map((stop) => {
        const standing = stopStanding(t, stop);
        const window = stopWindow(t, stop);
        return (
          <li key={stop.sequence} className="flex flex-col gap-0.5 py-2">
            <span className="text-xs font-medium">
              {t("Stop {0} · {1}", stop.sequence, stopTypeLabel(stop.type))}
            </span>
            <span className="text-xs text-muted-foreground">{stopPlace(stop)}</span>
            {stop.addressLine && (
              <span className="text-xs text-muted-foreground">{stop.addressLine}</span>
            )}
            {window && <span className="text-xs text-muted-foreground tabular-nums">{window}</span>}
            {standing && <span className="text-xs tabular-nums">{standing}</span>}
          </li>
        );
      })}
    </ol>
  );
}

function TrackingCard({ view, token }: { view: PublicShipmentView; token: string }) {
  const t = useT();
  const status = view.status ? publicStatusAttrs(t)[view.status] : null;

  return (
    <Card className="w-full max-w-md">
      <CardHeader className="gap-1">
        <span className="text-xs text-muted-foreground">{view.companyName}</span>
        <div className="flex items-center justify-between gap-2">
          <CardTitle>{t("Load {0}", view.shipmentProNumber)}</CardTitle>
          {status && (
            <Badge variant={phaseTone(status.phase)} title={status.description}>
              {status.text}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <Details view={view} />
        <Stops view={view} />

        {view.holds && view.holds.length > 0 && (
          <section className="flex flex-col gap-1.5">
            <Separator />
            <h2 className="text-xs font-medium">{t("On hold")}</h2>
            <ul className="flex flex-col gap-1">
              {view.holds.map((hold) => (
                <li key={`${hold.type}-${hold.startedAt}`} className="text-xs">
                  {t("{0}, since {1}", holdLabel(t, hold), formatUnixDateTime(hold.startedAt))}
                </li>
              ))}
            </ul>
          </section>
        )}

        {view.notes && view.notes.length > 0 && (
          <section className="flex flex-col gap-1.5">
            <Separator />
            <h2 className="text-xs font-medium">{t("Updates")}</h2>
            <ul className="flex flex-col gap-2">
              {view.notes.map((note) => (
                <li key={`${note.postedAt}-${note.text}`} className="flex flex-col gap-0.5">
                  <span className="text-xs whitespace-pre-wrap">{note.text}</span>
                  <span className="text-xs text-muted-foreground tabular-nums">
                    {formatUnixDateTime(note.postedAt)}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        )}

        {shows(view, "ProofOfDelivery") && (
          <section className="flex flex-col gap-1.5">
            <Separator />
            <h2 className="text-xs font-medium">{t("Proof of delivery")}</h2>
            {view.proofOfDelivery ? (
              <a
                href={shipmentTrackingService.proofOfDeliveryUrl(token)}
                rel="noreferrer noopener"
                className={cn(buttonVariants({ variant: "outline", size: "sm" }), "self-start")}
              >
                <Download01Icon className="size-4" aria-hidden />
                {t("Download {0}", view.proofOfDelivery.fileName)}
              </a>
            ) : (
              <span className="text-xs text-muted-foreground">
                {t("Not on file yet. It appears here once it is.")}
              </span>
            )}
          </section>
        )}

        <div className="flex items-center gap-1.5 rounded-md border bg-muted px-2 py-1.5">
          <ClockIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span className="text-xs text-muted-foreground">
            {t("This link works until {0}.", formatUnixDateTime(view.expiresAt))}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function InvalidLinkCard() {
  const t = useT();

  return (
    <StatusCard
      icon={<SlashCircle01Icon className="size-8 text-muted-foreground" aria-hidden />}
      title={t("This tracking link is no longer valid")}
      body={t(
        "The link may have expired or been turned off. Ask the company that sent it for a new one.",
      )}
    />
  );
}

/**
 * The page a customer opens from a tracking link. It is read-only and shows
 * only what the customer's settings allow; the server decides that, so the
 * page draws whatever parts it was given.
 */
export function ShipmentTrackingPublicPage() {
  const t = useT();
  const { token = "" } = useParams();

  const viewQuery = useQuery({
    queryKey: ["public-shipment-tracking", token] as const,
    queryFn: () => shipmentTrackingService.getPublicView(token),
    retry: false,
    refetchInterval: 5 * 60 * 1000,
  });

  let content: React.ReactNode;
  if (viewQuery.isLoading) {
    content = (
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col gap-3 py-6">
          <Skeleton className="h-5 w-48 rounded-md" />
          <Skeleton className="h-32 rounded-md" />
        </CardContent>
      </Card>
    );
  } else if (viewQuery.isError) {
    const kind = publicLinkErrorKind(viewQuery.error);
    content =
      kind === "throttled" ? (
        <StatusCard
          icon={<ClockIcon className="size-8 text-muted-foreground" aria-hidden />}
          title={t("Too many attempts")}
          body={t("Please wait a minute and open the link again.")}
        />
      ) : kind === "unavailable" ? (
        <StatusCard
          icon={<AlertTriangleIcon className="size-8 text-muted-foreground" aria-hidden />}
          title={t("Temporarily unavailable")}
          body={t(
            "The shipment could not be loaded because of a temporary problem. Please try again in a moment.",
          )}
          action={
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="mt-2"
              isLoading={viewQuery.isRefetching}
              loadingText={t("Retrying...")}
              onClick={() => void viewQuery.refetch()}
            >
              {t("Try again")}
            </Button>
          }
        />
      ) : (
        <InvalidLinkCard />
      );
  } else if (viewQuery.data) {
    content = <TrackingCard view={viewQuery.data} token={token} />;
  } else {
    content = <InvalidLinkCard />;
  }

  return (
    <>
      <Metadata title={t("Shipment tracking")} description={t("Follow a shipment")} />
      <PublicPageShell
        footer={t(
          "Powered by Trenova. Questions about this shipment? Reply to the email the link arrived in.",
        )}
      >
        {content}
      </PublicPageShell>
    </>
  );
}
