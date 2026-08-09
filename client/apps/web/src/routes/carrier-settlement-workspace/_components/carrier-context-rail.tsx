import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import {
  CarrierCostEventStatusBadge,
  CarrierSettlementStatusBadge,
} from "@trenova/shared/components/status-badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  fetchCarrierLedgerEntries,
  fetchCarrierPendingCostEvents,
  fetchCarrierRecentSettlements,
  type CarrierSettlementRow,
} from "@/lib/graphql/carrier-settlement";
import type {
  CarrierCostEventStatus,
  CarrierLedgerEntryType,
  CarrierSettlementStatus,
} from "@trenova/shared/types/carrier-settlement";
import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";

function formatDate(unix?: number | null): string {
  return formatUnixMonthDay(unix, { fallback: "—" });
}

const ledgerEntryLabels: Record<CarrierLedgerEntryType, string> = {
  Bill: "Bill",
  Payment: "Payment",
  Adjustment: "Adjustment",
};

export function CarrierContextRail({
  carrierId,
  carrierName,
  selectedSettlement,
}: {
  carrierId: string | null;
  carrierName: string | null;
  selectedSettlement: CarrierSettlementRow | null;
}) {
  if (!carrierId) {
    return (
      <div className="hidden items-center justify-center rounded-lg border bg-card p-6 text-center text-xs text-muted-foreground lg:flex">
        Carrier context appears here once a settlement is selected.
      </div>
    );
  }

  return (
    <div className="hidden min-h-0 flex-col overflow-hidden rounded-lg border bg-card lg:flex">
      <ScrollArea
        className="min-h-0 flex-1"
        viewportClassName="min-h-0"
        maskVariant="card"
        maskHeight={18}
      >
        <div className="flex flex-col gap-3 p-3">
          <div>
            <h3 className="text-sm font-semibold">{carrierName ?? "Carrier"}</h3>
            <p className="text-[11px] text-muted-foreground">
              Everything affecting this carrier&apos;s payable — cost accruals, recent statements,
              and the AP subledger.
            </p>
          </div>
          <UnsettledCostSection carrierId={carrierId} />
          <RecentSettlementsSection
            carrierId={carrierId}
            selectedSettlementId={selectedSettlement?.id ?? null}
          />
          <LedgerSection carrierId={carrierId} />
        </div>
      </ScrollArea>
    </div>
  );
}

function RailSection({
  title,
  hint,
  action,
  children,
}: {
  title: string;
  hint: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="border-t pt-3 first:border-t-0 first:pt-0">
      <div className="mb-1.5 flex items-center justify-between gap-2">
        <div>
          <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
            {title}
          </h4>
          <p className="text-[10px] text-muted-foreground">{hint}</p>
        </div>
        {action}
      </div>
      {children}
    </div>
  );
}

function UnsettledCostSection({ carrierId }: { carrierId: string }) {
  const { data: events, isLoading } = useQuery({
    queryKey: ["carrier-pending-cost-events", carrierId],
    queryFn: () => fetchCarrierPendingCostEvents(carrierId),
  });

  if (isLoading) {
    return (
      <RailSection title="Unsettled Cost" hint="Accrued cost not yet on a settlement.">
        <Skeleton className="h-16 w-full" />
      </RailSection>
    );
  }

  const list = events ?? [];

  return (
    <RailSection
      title="Unsettled Cost"
      hint="Accrued purchased-transportation cost waiting for the next settlement run."
    >
      {list.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">
          Nothing waiting. New cost accrues automatically as carrier-covered moves complete.
        </p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {list.map((event) => (
            <li key={event.id} className="rounded-md border p-2">
              <div className="flex items-center gap-1.5">
                <span className="font-mono text-[10px] text-muted-foreground">
                  {event.proNumber || "No PRO"}
                </span>
                <span className="text-[10px] text-muted-foreground">
                  {formatDate(event.eventDate)}
                </span>
                <span className="ml-auto text-xs font-semibold">
                  <AmountDisplay value={event.amountMinor} currency={event.currencyCode} />
                </span>
              </div>
              <div className="mt-1 flex items-center gap-1.5">
                <CarrierCostEventStatusBadge status={event.status as CarrierCostEventStatus} />
                <span className="truncate text-[10px] text-muted-foreground">
                  {event.description}
                </span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </RailSection>
  );
}

function RecentSettlementsSection({
  carrierId,
  selectedSettlementId,
}: {
  carrierId: string;
  selectedSettlementId: string | null;
}) {
  const { data: settlements, isLoading } = useQuery({
    queryKey: ["carrier-recent-settlements", carrierId],
    queryFn: () => fetchCarrierRecentSettlements(carrierId, 5),
  });

  if (isLoading) {
    return (
      <RailSection title="Recent Settlements" hint="Latest statements for this carrier.">
        <Skeleton className="h-12 w-full" />
      </RailSection>
    );
  }

  const list = (settlements ?? []).filter((settlement) => settlement.id !== selectedSettlementId);

  return (
    <RailSection title="Recent Settlements" hint="Latest statements for this carrier.">
      {list.length === 0 ? (
        <p className="text-[11px] text-muted-foreground">No other settlements on record.</p>
      ) : (
        <ul className="flex flex-col gap-1">
          {list.map((settlement) => (
            <li key={settlement.id} className="flex items-center gap-2 rounded-md border p-2">
              <div className="min-w-0 flex-1">
                <p className="truncate font-mono text-[11px] font-medium">
                  {settlement.settlementNumber}
                </p>
                <p className="text-[10px] text-muted-foreground">
                  {formatDate(settlement.periodStart)} – {formatDate(settlement.periodEnd)}
                </p>
              </div>
              <div className="flex shrink-0 flex-col items-end gap-0.5">
                <span className="text-[11px] font-semibold">
                  <AmountDisplay
                    value={settlement.netPayableMinor}
                    currency={settlement.currencyCode}
                  />
                </span>
                <CarrierSettlementStatusBadge
                  status={settlement.status as CarrierSettlementStatus}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </RailSection>
  );
}

function LedgerSection({ carrierId }: { carrierId: string }) {
  const { data: entries, isLoading } = useQuery({
    queryKey: ["carrier-ledger-entries", carrierId],
    queryFn: () => fetchCarrierLedgerEntries(carrierId, 100),
  });

  if (isLoading) {
    return (
      <RailSection title="AP Subledger" hint="Bills, payments, and adjustments.">
        <Skeleton className="h-12 w-full" />
      </RailSection>
    );
  }

  const list = entries ?? [];
  const balance = list.reduce((sum, entry) => sum + entry.amountMinor, 0);

  return (
    <RailSection
      title="AP Subledger"
      hint="Bills, payments, and adjustments — the balance reconciles to the GL's AP account."
    >
      <p className="text-sm font-semibold">
        <AmountDisplay value={balance} currency="USD" />
        <span className="ml-1 text-[10px] font-normal text-muted-foreground">
          open balance{list.length >= 100 ? " (latest 100 entries)" : ""}
        </span>
      </p>
      {list.length > 0 && (
        <ul className="mt-1.5 flex flex-col gap-1">
          {list.slice(0, 8).map((entry) => (
            <li
              key={entry.id}
              className="flex items-center gap-2 rounded-md border px-2 py-1 text-[11px]"
            >
              <span className="font-medium">
                {ledgerEntryLabels[entry.entryType as CarrierLedgerEntryType] ?? entry.entryType}
              </span>
              <span className="truncate font-mono text-[10px] text-muted-foreground">
                {entry.documentNumber}
              </span>
              <span className="ml-auto shrink-0 tabular-nums">
                <AmountDisplay value={entry.amountMinor} variant="auto" currency="USD" />
              </span>
            </li>
          ))}
        </ul>
      )}
    </RailSection>
  );
}
