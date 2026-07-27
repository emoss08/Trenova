import { ExceptionsList } from "@/components/work-modules/exceptions-list";
import { UnassignedQueueList } from "@/components/work-modules/unassigned-queue-list";
import {
  ATTENTION_ROWS,
  ATTENTION_TONE_DOT_CLASSES,
  type AttentionRowConfig,
} from "@/config/attention-rows";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useCallback, useState } from "react";
import { Link, useNavigate } from "react-router";
import { WidgetCount, WidgetEmpty, WidgetShell, WidgetSkeleton } from "../widget-shell";
import type { WidgetProps } from "../widget-registry";

const SHIPMENTS_HREF = "/shipment-management/shipments";

/** Opening a load from the home screen lands on the shipment workspace with it expanded. */
function shipmentHref(shipmentId: string): string {
  return `${SHIPMENTS_HREF}?expanded=${encodeURIComponent(shipmentId)}`;
}

export function AttentionWidget({ widget, data }: WidgetProps) {
  const total = ATTENTION_ROWS.reduce((sum, row) => sum + (data.attention?.[row.key] ?? 0), 0);

  return (
    <WidgetShell
      title={widget.title || "Needs Attention"}
      badge={<WidgetCount value={total} tone={total > 0 ? "warning" : "muted"} />}
    >
      {data.attentionLoading ? (
        <WidgetSkeleton rows={5} />
      ) : (
        <div className="flex flex-col gap-0.5">
          {ATTENTION_ROWS.filter((row) => data.attention?.[row.key] != null).map((row) => (
            <AttentionTile key={row.key} row={row} count={data.attention?.[row.key] ?? 0} />
          ))}
        </div>
      )}
    </WidgetShell>
  );
}

function AttentionTile({ row, count }: { row: AttentionRowConfig; count: number }) {
  const hasWork = count > 0;

  return (
    <Link
      to={row.path}
      className="flex items-center gap-2 rounded px-1.5 py-1 transition-colors hover:bg-muted/60"
    >
      <span
        className={cn(
          "size-1.5 shrink-0 rounded-full",
          hasWork ? ATTENTION_TONE_DOT_CLASSES[row.tone] : "bg-muted-foreground/40",
        )}
      />
      <span className="min-w-0 flex-1 truncate text-xs">{row.label}</span>
      <span
        className={cn(
          "font-table text-[10.5px] tabular-nums",
          hasWork ? "font-semibold text-foreground" : "text-muted-foreground",
        )}
      >
        {count > 999 ? "999+" : count}
      </span>
    </Link>
  );
}

export function UnassignedWidget({ widget }: WidgetProps) {
  const navigate = useNavigate();
  const [summary, setSummary] = useState({
    totalCount: undefined as number | undefined,
    pendingRevenue: 0,
  });

  const handleSelect = useCallback(
    (shipmentId: string) => void navigate(shipmentHref(shipmentId)),
    [navigate],
  );

  return (
    <WidgetShell
      title={widget.title || "Unassigned Loads"}
      badge={<WidgetCount value={summary.totalCount} tone="warning" />}
      actions={
        summary.pendingRevenue > 0 ? (
          <span className="shrink-0 font-table text-[9.5px] text-muted-foreground tabular-nums">
            {formatCurrency(summary.pendingRevenue)} waiting
          </span>
        ) : undefined
      }
      href={SHIPMENTS_HREF}
      hrefLabel="Open dispatch board"
    >
      <UnassignedQueueList
        limit={widget.config.limit ?? undefined}
        onSelect={handleSelect}
        onSummary={setSummary}
      />
    </WidgetShell>
  );
}

export function ExceptionsWidget({ widget }: WidgetProps) {
  const navigate = useNavigate();
  const [count, setCount] = useState(0);

  const handleSelect = useCallback(
    (shipmentId: string) => void navigate(shipmentHref(shipmentId)),
    [navigate],
  );

  return (
    <WidgetShell
      title={widget.title || "Exceptions"}
      badge={<WidgetCount value={count} tone={count > 0 ? "danger" : "muted"} />}
      href={SHIPMENTS_HREF}
      hrefLabel="Open dispatch board"
      bodyClassName="p-0"
    >
      <ExceptionsList
        limit={widget.config.limit ?? undefined}
        onSelect={handleSelect}
        onCount={setCount}
      />
    </WidgetShell>
  );
}

export function DetentionWatchWidget({ widget, data }: WidgetProps) {
  const items = data.shipmentAnalytics.detentionWatchlist.items.slice(
    0,
    widget.config.limit ?? undefined,
  );

  return (
    <WidgetShell
      title={widget.title || "Detention Watch"}
      badge={<WidgetCount value={items.length} tone={items.length > 0 ? "warning" : "muted"} />}
      href={SHIPMENTS_HREF}
    >
      {data.shipmentAnalyticsLoading ? (
        <WidgetSkeleton />
      ) : items.length === 0 ? (
        <WidgetEmpty>Nothing dwelling past threshold.</WidgetEmpty>
      ) : (
        <div className="flex flex-col gap-0.5">
          {items.map((item) => (
            <Link
              key={item.shipmentId}
              to={shipmentHref(item.shipmentId)}
              className="flex items-center gap-2 rounded px-1.5 py-1 transition-colors hover:bg-muted/60"
            >
              <span
                className={cn(
                  "size-1.5 shrink-0 rounded-full",
                  item.tone === "danger" ? "bg-destructive" : "bg-warning",
                )}
              />
              <span className="min-w-0 flex-1 truncate text-xs">{item.customer}</span>
              <span className="shrink-0 font-table text-[10px] text-muted-foreground tabular-nums">
                {item.dwellLabel}
              </span>
            </Link>
          ))}
        </div>
      )}
    </WidgetShell>
  );
}

export function TomorrowsPickupsWidget({ widget, data }: WidgetProps) {
  const items = data.shipmentAnalytics.tomorrowsPickups.pickups.slice(
    0,
    widget.config.limit ?? undefined,
  );

  return (
    <WidgetShell
      title={widget.title || "Tomorrow's Pickups"}
      badge={<WidgetCount value={items.length} />}
      href={SHIPMENTS_HREF}
    >
      {data.shipmentAnalyticsLoading ? (
        <WidgetSkeleton rows={5} />
      ) : items.length === 0 ? (
        <WidgetEmpty>No pickups scheduled for tomorrow.</WidgetEmpty>
      ) : (
        <div className="flex flex-col gap-0.5">
          {items.map((item) => (
            <Link
              key={item.shipmentId}
              to={shipmentHref(item.shipmentId)}
              className="flex flex-col gap-0.5 rounded px-1.5 py-1 transition-colors hover:bg-muted/60"
            >
              <div className="flex items-baseline justify-between gap-2">
                <span className="truncate font-table text-[10.5px] font-semibold tabular-nums">
                  {item.proNumber}
                </span>
                <span className="shrink-0 font-table text-[9.5px] text-muted-foreground tabular-nums">
                  {formatToUserTimezone(item.pickupWindowStart, {
                    showTimeZone: false,
                    showSeconds: false,
                  })}
                </span>
              </div>
              <div className="flex items-baseline justify-between gap-2 text-[10px] text-muted-foreground">
                <span className="truncate">
                  {item.customer} · {item.origin} → {item.destination}
                </span>
                <span className="shrink-0 truncate">{item.driver || "Unassigned"}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </WidgetShell>
  );
}

/**
 * The queue widgets that are counts plus a destination. Each one is a link into
 * the workspace that owns the work, which is all Home has to do for them: the
 * page it points at is where the work actually happens.
 */
function CountWidget({
  title,
  count,
  loading,
  href,
  hrefLabel,
  emptyMessage,
  tone,
}: {
  title: string;
  count: number | undefined;
  loading: boolean;
  href: string;
  hrefLabel: string;
  emptyMessage: string;
  tone: "danger" | "warning" | "brand";
}) {
  return (
    <WidgetShell title={title} href={href} hrefLabel={hrefLabel} scroll={false}>
      {loading ? (
        <WidgetSkeleton rows={1} />
      ) : !count ? (
        <WidgetEmpty>{emptyMessage}</WidgetEmpty>
      ) : (
        <div className="flex flex-1 flex-col items-start justify-center gap-1">
          <span
            className={cn(
              "font-mono text-[28px] leading-none font-semibold tracking-tight tabular-nums",
              tone === "danger" && "text-destructive",
              tone === "warning" && "text-warning",
            )}
          >
            {count}
          </span>
          <span className="text-2xs text-muted-foreground">waiting on you</span>
        </div>
      )}
    </WidgetShell>
  );
}

export function MyApprovalsWidget({ widget, data }: WidgetProps) {
  return (
    <CountWidget
      title={widget.title || "My Approvals"}
      count={data.attention?.pendingApprovals ?? undefined}
      loading={data.attentionLoading}
      href="/billing/pending-approvals"
      hrefLabel="Open approvals"
      emptyMessage="Nothing waiting on your approval."
      tone="brand"
    />
  );
}

export function BillingQueueWidget({ widget, data }: WidgetProps) {
  return (
    <CountWidget
      title={widget.title || "Billing Queue"}
      count={data.attention?.billingQueue ?? undefined}
      loading={data.attentionLoading}
      href="/billing/queue"
      hrefLabel="Open billing queue"
      emptyMessage="The billing queue is clear."
      tone="brand"
    />
  );
}

export function ServiceFailuresWidget({ widget, data }: WidgetProps) {
  return (
    <CountWidget
      title={widget.title || "Service Failures"}
      count={data.attention?.serviceFailures ?? undefined}
      loading={data.attentionLoading}
      href="/shipment-management/service-failures"
      hrefLabel="Open service failures"
      emptyMessage="No open service failures."
      tone="danger"
    />
  );
}

export function EDIAttentionWidget({ widget, data }: WidgetProps) {
  return (
    <CountWidget
      title={widget.title || "EDI Needs Attention"}
      count={data.attention?.ediAttention ?? undefined}
      loading={data.attentionLoading}
      href="/edi/overview"
      hrefLabel="Open EDI"
      emptyMessage="EDI is running clean."
      tone="warning"
    />
  );
}

/**
 * Credential expiry is answered by three canned reports rather than a live
 * count, so this widget is a signpost to them rather than a number it would
 * have to invent.
 */
export function ExpiringCredentialsWidget({ widget }: WidgetProps) {
  const targets = [
    {
      label: "Driver licenses & medical cards",
      href: "/reports?canned=expiring-worker-credentials",
    },
    { label: "Tractor registrations", href: "/reports?canned=expiring-tractor-registrations" },
    { label: "Trailer registrations", href: "/reports?canned=expiring-trailer-registrations" },
  ];

  return (
    <WidgetShell title={widget.title || "Expiring Credentials"} href="/reports">
      <div className="flex flex-col gap-0.5">
        {targets.map((target) => (
          <Link
            key={target.href}
            to={target.href}
            className="rounded px-1.5 py-1 text-xs transition-colors hover:bg-muted/60"
          >
            {target.label}
          </Link>
        ))}
      </div>
    </WidgetShell>
  );
}
