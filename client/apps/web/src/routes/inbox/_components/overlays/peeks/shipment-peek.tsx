import "../../../_styles/triage-overlays.css";
import { recordPath } from "@/config/record-links";
import type { TriageShipmentPeek } from "@/lib/graphql/inbox";
import { inbox } from "@/lib/queries/inbox";
import { useQuery } from "@tanstack/react-query";
import {
  CheckIcon,
  Copy01Icon,
  LinkExternal01Icon,
  Truck01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@trenova/shared/components/ui/dialog";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import { useNavigate } from "react-router";
import { inlineErrorText } from "../../inline-error";
import { useTriageActions } from "../../use-triage-actions";
import { PhaseBadge, peekPhase } from "./peek-parts";

const WHEN: Intl.DateTimeFormatOptions = { weekday: "short", hour: "numeric", minute: "2-digit" };

function when(at: number | null | undefined): string {
  return at === null || at === undefined ? "" : formatUnixInUserTimezone(at, WHEN, "");
}

type Moment = { key: string; label: string; detail: string; state: "done" | "current" | "ahead" };

/** The shipment's life as a short timeline: picked up, on its way, delivered. */
function moments(t: TranslateFn, shipment: TriageShipmentPeek): Moment[] {
  const phase = peekPhase(shipment);
  const rows: Moment[] = [];
  shipment.stops.forEach((stop, index) => {
    const place = [stop.city, stop.state].filter((part) => part !== "").join(", ");
    const pickup = stop.kind.toLowerCase().includes("pick") || index === 0;
    const at = stop.done ? (stop.departedAt ?? stop.arrivedAt) : stop.scheduledAt;
    const label = pickup
      ? stop.done
        ? t("Picked up")
        : t("Pickup")
      : stop.done
        ? t("Delivered")
        : t("Delivery");
    if (!pickup && phase !== "delivered" && rows.length > 0 && shipment.progress > 0) {
      rows.push({
        key: `transit-${index}`,
        label: t("In transit"),
        detail: shipment.eta === null ? "" : t("ETA {0}", when(shipment.eta)),
        state: "current",
      });
    }
    rows.push({
      key: `${stop.kind}-${index}`,
      label,
      detail: [place, when(at)].filter((part) => part !== "").join(" · "),
      state: stop.done ? "done" : "ahead",
    });
  });

  return rows;
}

/**
 * A shipment the message is about, drawn over the reader: where it is on its
 * route, its stops, who is driving it, its numbers, and what is already on
 * file — enough to answer the message without leaving it.
 */
export function ShipmentPeek({ shipmentId, onClose }: { shipmentId: string; onClose: () => void }) {
  const t = useT();
  const query = useQuery(inbox.shipmentPeek(shipmentId));
  const shipment = query.data;

  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent size="sm" showCloseButton={false} className="gap-3.5">
        {shipment === undefined ? (
          query.isError ? (
            <>
              <DialogTitle>{t("Shipment")}</DialogTitle>
              <p role="alert" className="text-danger text-sm">
                {inlineErrorText(t, query.error)}
              </p>
            </>
          ) : (
            <>
              <DialogTitle className="sr-only">{t("Shipment")}</DialogTitle>
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-24 w-full" />
            </>
          )
        ) : (
          <ShipmentBody shipment={shipment} onClose={onClose} />
        )}
      </DialogContent>
    </Dialog>
  );
}

function ShipmentBody({
  shipment,
  onClose,
}: {
  shipment: TriageShipmentPeek;
  onClose: () => void;
}) {
  const t = useT();
  const navigate = useNavigate();
  const actions = useTriageActions();
  const phase = peekPhase(shipment);
  const origin = shipment.stops[0];
  const destination = shipment.stops.at(-1);
  const progress = Math.min(1, Math.max(0, shipment.progress));
  const path = recordPath("shipment", shipment.id);
  const subtitle = [shipment.customerName, shipment.equipment]
    .filter((part) => part !== "")
    .join(" · ");

  return (
    <>
      <header className="flex items-start gap-2.5">
        <div className="flex min-w-0 flex-1 flex-col">
          <DialogTitle className="font-mono text-xl font-semibold">
            {shipment.proNumber}
          </DialogTitle>
          {subtitle !== "" && <span className="text-foreground-subtle text-sm">{subtitle}</span>}
        </div>
        <PhaseBadge phase={phase} />
        <Button variant="ghost" size="icon-sm" aria-label={t("Close")} onClick={onClose}>
          <XCloseIcon className="size-3.5" />
        </Button>
      </header>

      {origin !== undefined && destination !== undefined && (
        <div className="mt-1 grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
          <div className="flex flex-col leading-tight">
            <b className="text-base font-medium">{origin.city}</b>
            <span className="text-foreground-subtle text-xs">{origin.state}</span>
          </div>
          <div
            role="progressbar"
            aria-label={t("Route")}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={Math.round(progress * 100)}
            className="tb-route-track relative h-0.5"
          >
            <i
              className={cn(
                "tb-route-fill absolute inset-0 rounded-full",
                phase === "late" ? "bg-warning" : "bg-success",
              )}
              style={{ "--tb-progress": progress } as CSSProperties}
            />
            {phase !== "delivered" && (
              <span
                aria-hidden
                className="border-brand bg-canvas text-brand absolute top-1/2 -mt-2.75 -ml-2.75 flex size-5.5 items-center justify-center rounded-full border-[1.5px]"
                style={{ left: `${progress * 100}%` }}
              >
                <Truck01Icon className="size-2.75" />
              </span>
            )}
          </div>
          <div className="flex flex-col text-right leading-tight">
            <b className="text-base font-medium">{destination.city}</b>
            <span className="text-foreground-subtle text-xs">{destination.state}</span>
          </div>
        </div>
      )}

      <ol className="relative flex flex-col before:absolute before:top-2.25 before:bottom-2.25 before:left-0.75 before:w-px before:bg-border">
        {moments(t, shipment).map((moment) => (
          <li
            key={moment.key}
            className={cn(
              "relative grid grid-cols-[minmax(0,1fr)_auto] gap-2 py-1 pl-4 text-sm",
              moment.state === "current" ? "text-foreground" : "text-foreground-muted",
            )}
          >
            <span
              aria-hidden
              className={cn(
                "absolute top-2.25 left-0 size-1.75 rounded-full border-[1.5px]",
                moment.state === "done" && "bg-success border-success",
                moment.state === "current" && "bg-canvas border-warning",
                moment.state === "ahead" && "bg-canvas border-border-strong",
              )}
            />
            <span>{moment.label}</span>
            <em className="text-foreground-subtle font-mono text-2xs not-italic">
              {moment.detail}
            </em>
          </li>
        ))}
      </ol>

      <DescriptionList columns={2}>
        <DescriptionItem label={t("Driver")}>
          {shipment.driverName === "" ? <DescriptionEmpty /> : shipment.driverName}
        </DescriptionItem>
        <DescriptionItem label={t("BOL")} valueClassName="font-mono" numeric>
          {shipment.bol === "" ? <DescriptionEmpty /> : shipment.bol}
        </DescriptionItem>
        <DescriptionItem label={t("PO")} valueClassName="font-mono" numeric>
          {shipment.poNumber === "" ? <DescriptionEmpty /> : shipment.poNumber}
        </DescriptionItem>
        {phase === "delivered" ? (
          <DescriptionItem label={t("Delivered")}>
            {shipment.deliveredAt === null ? <DescriptionEmpty /> : when(shipment.deliveredAt)}
          </DescriptionItem>
        ) : (
          <DescriptionItem
            label={t("ETA")}
            valueClassName={phase === "late" ? "text-warning" : undefined}
          >
            {shipment.eta === null ? <DescriptionEmpty /> : when(shipment.eta)}
          </DescriptionItem>
        )}
      </DescriptionList>

      <div className="flex flex-col gap-1.5">
        <span className="text-foreground-subtle text-xs">{t("On file")}</span>
        {shipment.documentsOnFile.length === 0 ? (
          <span className="text-foreground-subtle text-sm">{t("Nothing filed yet")}</span>
        ) : (
          <ul className="flex flex-wrap gap-1.25">
            {shipment.documentsOnFile.map((document) => (
              <li
                key={document}
                className="bg-surface-hover text-foreground-muted inline-flex h-5.5 items-center gap-1.25 rounded-badge px-2 text-xs"
              >
                <CheckIcon aria-hidden className="text-success size-2.5" strokeWidth={2.6} />
                {document}
              </li>
            ))}
          </ul>
        )}
      </div>

      <footer className="border-border-subtle flex items-center gap-1.5 border-t pt-3">
        <Button
          variant="ghost"
          size="sm"
          onClick={() =>
            actions.copyText(new URL(path, window.location.origin).toString(), t("Link copied"))
          }
        >
          <Copy01Icon className="size-3.5" />
          {t("Copy link")}
        </Button>
        <span className="flex-1" />
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            onClose();
            void navigate(path);
          }}
        >
          {t("Open in Shipments")}
          <LinkExternal01Icon className="size-3.5" />
        </Button>
      </footer>
    </>
  );
}
