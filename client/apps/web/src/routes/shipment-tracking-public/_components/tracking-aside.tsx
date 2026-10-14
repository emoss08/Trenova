import { dayAndTime, holdLabel, shows } from "@/lib/shipment-tracking";
import { ShipmentTrackingService } from "@/services/shipment-tracking";
import {
  AlertTriangleIcon,
  Download01Icon,
  File06Icon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo, formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import type { PublicShipmentView } from "@trenova/shared/types/shipment-tracking";

const shipmentTrackingService = new ShipmentTrackingService();

function Heading({ children }: { children: React.ReactNode }) {
  return <h2 className="text-sm font-semibold">{children}</h2>;
}

function Updates({ view, now }: { view: PublicShipmentView; now: number }) {
  const t = useT();
  if (!view.notes || view.notes.length === 0) {
    return null;
  }

  return (
    <section className="flex flex-col gap-3">
      <Heading>{t("Updates from {0}", view.companyName)}</Heading>
      <ol className="flex flex-col gap-4">
        {view.notes.map((note, index) => (
          <li key={`${note.postedAt}-${index}`} className="grid grid-cols-[8px_1fr] gap-x-3">
            <span
              className={cn(
                "mt-1.5 size-2 rounded-full",
                index === 0 ? "st-dot" : "bg-border",
              )}
            />
            <div className="flex flex-col gap-1">
              <p className="text-sm leading-relaxed whitespace-pre-wrap">{note.text}</p>
              <time
                className="text-xs text-muted-foreground tabular-nums"
                title={formatUnixDateTime(note.postedAt)}
              >
                {formatSecondsAgo(now - note.postedAt)}
              </time>
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}

function Details({ view }: { view: PublicShipmentView }) {
  const t = useT();
  const items: { label: string; value: string }[] = [];
  if (view.customerReference) items.push({ label: t("Your reference"), value: view.customerReference });
  if (view.bol) items.push({ label: t("BOL"), value: view.bol });
  if (view.carrierName) items.push({ label: t("Carrier"), value: view.carrierName });
  if (view.equipment) items.push({ label: t("Equipment"), value: view.equipment });
  if (view.pieces != null) items.push({ label: t("Pieces"), value: view.pieces.toLocaleString() });
  if (view.weight != null) {
    items.push({ label: t("Weight"), value: t("{0} lb", view.weight.toLocaleString()) });
  }
  if (items.length === 0) {
    return null;
  }

  return (
    <section className="flex flex-col gap-1">
      <Heading>{t("Shipment")}</Heading>
      <DescriptionList layout="split" className="divide-y divide-border-subtle">
        {items.map((item) => (
          <DescriptionItem key={item.label} label={item.label}>
            {item.value}
          </DescriptionItem>
        ))}
      </DescriptionList>
    </section>
  );
}

function ProofOfDelivery({
  view,
  token,
  now,
}: {
  view: PublicShipmentView;
  token: string;
  now: number;
}) {
  const t = useT();
  if (!shows(view, "ProofOfDelivery")) {
    return null;
  }

  return (
    <section className="flex flex-col gap-2">
      <Heading>{t("Proof of delivery")}</Heading>
      {view.proofOfDelivery ? (
        <div className="flex items-center gap-3 rounded-lg bg-background p-3 ring-1 ring-foreground/10">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-success-subtle text-success-subtle-foreground">
            <File06Icon className="size-4" aria-hidden />
          </span>
          <div className="flex min-w-0 flex-1 flex-col">
            <span className="truncate text-sm font-medium">{view.proofOfDelivery.fileName}</span>
            <span className="text-xs text-muted-foreground tabular-nums">
              {t("Filed {0}", dayAndTime(t, view.proofOfDelivery.filedAt, now, { inline: true }))}
            </span>
          </div>
          <a
            href={shipmentTrackingService.proofOfDeliveryUrl(token)}
            rel="noreferrer noopener"
            className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
          >
            <Download01Icon className="size-3.5" aria-hidden />
            {t("Download")}
          </a>
        </div>
      ) : (
        <p className="rounded-lg border border-dashed border-border p-3 text-xs text-muted-foreground">
          {t("It appears here as soon as it is on file after delivery.")}
        </p>
      )}
    </section>
  );
}

/** What sits beside the route: holds, the shipper's notes, references, the POD. */
export function TrackingAside({
  view,
  token,
  now,
}: {
  view: PublicShipmentView;
  token: string;
  now: number;
}) {
  const t = useT();

  return (
    <aside className="flex flex-col gap-9">
      {view.holds?.map((hold) => (
        <Alert key={`${hold.type}-${hold.startedAt}`} variant="warning" size="sm">
          <AlertTriangleIcon aria-hidden />
          <AlertTitle>{holdLabel(t, hold)}</AlertTitle>
          <AlertDescription>
            {t("On hold since {0}", dayAndTime(t, hold.startedAt, now, { inline: true }))}
          </AlertDescription>
        </Alert>
      ))}
      <Updates view={view} now={now} />
      <Details view={view} />
      <ProofOfDelivery view={view} token={token} now={now} />
    </aside>
  );
}
