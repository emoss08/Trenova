import { Button } from "@/components/ui/button";
import { formatCurrency } from "@/lib/utils";
import type { Shipment, Stop, StopType } from "@/types/shipment";
import {
  AlertTriangleIcon,
  DollarSignIcon,
  MapPinIcon,
  MessageSquareIcon,
  PlusIcon,
  UploadIcon,
} from "lucide-react";

const STOP_KIND: Record<StopType, string> = {
  Pickup: "PICKUP",
  Delivery: "DELIVERY",
  SplitPickup: "SPLIT-PU",
  SplitDelivery: "SPLIT-DL",
};

type StopState = "done" | "current" | "upcoming";

function stopState(stop: Stop): StopState {
  if (stop.status === "Completed") return "done";
  if (stop.status === "InTransit") return "current";
  return "upcoming";
}

function parseDecimal(value: string | number | null | undefined): number {
  if (value === null || value === undefined) return 0;
  if (typeof value === "number") return value;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

const COMPACT_STOP_FORMAT = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

function formatStopTime(timestamp: number | null | undefined): string {
  if (!timestamp) return "—";
  // "Apr 21, 08:00" — matches the design spec; the timezone-aware long form
  // (formatToUserTimezone) wrapped onto two lines in the timeline column.
  return COMPACT_STOP_FORMAT.format(new Date(timestamp * 1000));
}

function StopDot({ state }: { state: StopState }) {
  if (state === "current") {
    return (
      <span
        aria-hidden
        className="absolute top-[3px] -left-[13px] inline-block size-[9px] rounded-full bg-brand"
        style={{
          boxShadow: "0 0 0 3px color-mix(in oklch, var(--brand) 18%, transparent)",
          border: "1.5px solid var(--card)",
        }}
      />
    );
  }
  if (state === "done") {
    return (
      <span
        aria-hidden
        className="absolute top-[3px] -left-[13px] inline-block size-[9px] rounded-full bg-success"
        style={{ border: "1.5px solid var(--card)" }}
      />
    );
  }
  return (
    <span
      aria-hidden
      className="absolute top-[3px] -left-[13px] inline-block size-[9px] rounded-full bg-muted"
      style={{ border: "1.5px dashed var(--border)" }}
    />
  );
}

function stopNote(stop: Stop): string {
  if (stop.actualArrival) {
    const arrived = formatStopTime(stop.actualArrival);
    if (stop.actualDeparture) {
      return `${arrived} → departed ${formatStopTime(stop.actualDeparture)}`;
    }
    return `Arrived ${arrived}`;
  }
  if (stop.scheduledWindowEnd && stop.scheduledWindowStart) {
    return `Window ${formatStopTime(stop.scheduledWindowStart)} – ${formatStopTime(stop.scheduledWindowEnd)}`;
  }
  if (stop.scheduledWindowStart) {
    return `Scheduled ${formatStopTime(stop.scheduledWindowStart)}`;
  }
  return "—";
}

function RouteTimeline({ stops }: { stops: Stop[] }) {
  if (stops.length === 0) {
    return <p className="text-[11px] text-muted-foreground">No stops on this shipment.</p>;
  }
  return (
    <div className="relative pl-4">
      <div
        aria-hidden
        className="absolute top-2 bottom-2 left-[5px] bg-border"
        style={{ width: "1.5px" }}
      />
      {stops.map((stop, i) => {
        const state = stopState(stop);
        const time = formatStopTime(stop.actualArrival ?? stop.scheduledWindowStart);
        const kind = STOP_KIND[stop.type] ?? stop.type.toUpperCase();
        const loc = stop.location?.name ?? "—";
        return (
          <div
            key={stop.id ?? `${stop.locationId}-${i}`}
            className="relative pb-2 text-[11px] last:pb-0"
          >
            <StopDot state={state} />
            <div className="grid grid-cols-[88px_1fr] gap-x-2 leading-tight">
              <span className="font-table text-[10px] text-muted-foreground tabular-nums">
                {time}
              </span>
              <div className="flex min-w-0 flex-col gap-0.5">
                <div className="flex min-w-0 items-baseline gap-2">
                  <span
                    className={`shrink-0 font-table text-[9.5px] font-semibold tracking-wider ${
                      state === "current" ? "text-brand" : "text-muted-foreground"
                    }`}
                  >
                    {kind}
                  </span>
                  <span className="truncate font-medium text-foreground">{loc}</span>
                </div>
                <span className="truncate font-table text-[10.5px] text-muted-foreground tabular-nums">
                  {stopNote(stop)}
                </span>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function FinancialsBlock({ shipment }: { shipment: Shipment }) {
  const freight = parseDecimal(shipment.freightChargeAmount as unknown as string);
  const other = parseDecimal(shipment.otherChargeAmount as unknown as string);
  const total = parseDecimal(shipment.totalChargeAmount as unknown as string);
  const accessorialsTotal = (shipment.additionalCharges ?? []).reduce(
    (sum, c) => sum + parseDecimal(c.amount as unknown as string) * (c.unit ?? 1),
    0,
  );

  const rows: { label: string; value: string; bold?: boolean; tone?: string }[] = [
    { label: "Linehaul", value: formatCurrency(freight) },
    { label: "Accessorials", value: formatCurrency(accessorialsTotal) },
    { label: "Other charges", value: formatCurrency(other - accessorialsTotal) },
    { label: "Total revenue", value: formatCurrency(total), bold: true },
  ];

  return (
    <dl className="grid grid-cols-1 gap-1 text-[11px]">
      {rows.map((row) => (
        <div
          key={row.label}
          className={`flex items-center justify-between py-[3px] ${row.bold ? "mt-1 border-t border-border pt-2" : ""}`}
        >
          <dt className="text-muted-foreground">{row.label}</dt>
          <dd
            className={`font-table tabular-nums ${row.bold ? "font-semibold" : "font-medium"}`}
            style={row.tone ? { color: row.tone } : undefined}
          >
            {row.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

function DocumentsBlock({ shipment }: { shipment: Shipment }) {
  const docRows: { label: string; value: string; tone: "ok" | "missing" | "warn" }[] = [
    {
      label: "BOL",
      value: shipment.bol ? `${shipment.bol}.pdf` : "—",
      tone: shipment.bol ? "ok" : "missing",
    },
    {
      label: "PRO",
      value: shipment.proNumber ?? "—",
      tone: shipment.proNumber ? "ok" : "missing",
    },
    {
      label: "POD",
      value: shipment.actualDeliveryDate ? "Received" : "Pending",
      tone: shipment.actualDeliveryDate ? "ok" : "warn",
    },
    {
      label: "Customer invoice",
      value: shipment.billedAt ? "Issued" : "—",
      tone: shipment.billedAt ? "ok" : "missing",
    },
  ];

  const tone = (t: "ok" | "missing" | "warn") =>
    t === "ok" ? "text-success" : t === "warn" ? "text-warning" : "text-muted-foreground";

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-1 text-[11px]">
        {docRows.map((d) => (
          <div key={d.label} className="flex items-center justify-between">
            <span className="text-muted-foreground">{d.label}</span>
            <span className={`font-table text-[10.5px] tabular-nums ${tone(d.tone)}`}>
              {d.value}
            </span>
          </div>
        ))}
      </div>
      <Button variant="outline" size="xs" className="w-full justify-center">
        <UploadIcon className="size-3" />
        Upload
      </Button>
    </div>
  );
}

function QuickActionsBlock() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-1.5">
        <Button variant="outline" size="xs" className="justify-start">
          <MessageSquareIcon className="size-3" />
          Message driver
        </Button>
        <Button variant="outline" size="xs" className="justify-start">
          <MapPinIcon className="size-3" />
          Update ETA
        </Button>
        <Button variant="outline" size="xs" className="justify-start">
          <DollarSignIcon className="size-3" />
          Add accessorial
        </Button>
        <Button
          variant="outline"
          size="xs"
          className="justify-start border-destructive/30 text-destructive hover:bg-destructive/5"
        >
          <AlertTriangleIcon className="size-3" />
          Cancel shipment
        </Button>
      </div>
      <h5 className="cc-label mt-1">Comments · 0</h5>
      <div className="rounded border border-border bg-muted/40 p-2 text-[10.5px] leading-snug text-muted-foreground">
        <p>
          No comments yet. Use{" "}
          <span className="font-table text-foreground">@mentions</span> to ping a teammate.
        </p>
        <button
          type="button"
          className="mt-1.5 inline-flex items-center gap-1 text-[10.5px] font-medium text-brand hover:underline"
        >
          <PlusIcon className="size-3" />
          Add comment
        </button>
      </div>
    </div>
  );
}

export function ExpandedRow({ shipment }: { shipment: Shipment }) {
  const stops = shipment.moves?.flatMap((m) => m.stops ?? []) ?? [];
  return (
    <div className="grid grid-cols-1 gap-5 px-4 py-3 md:grid-cols-[2fr_1.4fr_1fr_1fr]">
      <section className="min-w-0">
        <h4 className="cc-label mb-1.5">Route timeline</h4>
        <RouteTimeline stops={stops} />
      </section>
      <section className="min-w-0">
        <h4 className="cc-label mb-1.5">Financials</h4>
        <FinancialsBlock shipment={shipment} />
      </section>
      <section className="min-w-0">
        <h4 className="cc-label mb-1.5">Documents</h4>
        <DocumentsBlock shipment={shipment} />
      </section>
      <section className="min-w-0">
        <h4 className="cc-label mb-1.5">Quick actions</h4>
        <QuickActionsBlock />
      </section>
    </div>
  );
}
