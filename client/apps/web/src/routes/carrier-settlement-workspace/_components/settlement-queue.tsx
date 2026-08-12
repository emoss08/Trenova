import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { CarrierSettlementStatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  BulkMarkPaidDialog,
  CARRIER_MARK_PAID_METHODS,
} from "@/components/settlements/bulk-mark-paid-dialog";
import { runBulkAction } from "@/lib/bulk-run";
import {
  carrierLifecycleVerbs,
  eligibleCarrierSettlements,
  type CarrierSettlementLifecycleAction,
} from "@/lib/carrier-settlement-lifecycle";
import {
  approveCarrierSettlement,
  markCarrierSettlementPaid,
  postCarrierSettlement,
  submitCarrierSettlement,
  type CarrierSettlementRow,
} from "@/lib/graphql/carrier-settlement";
import { cn } from "@trenova/shared/lib/utils";
import type { CarrierSettlementStatus } from "@trenova/shared/types/carrier-settlement";
import { useMutation } from "@tanstack/react-query";
import { CheckCheck, CircleDollarSign, Search, Send } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";

export type QueueFilter = "all" | "Draft" | "PendingApproval" | "Approved" | "Posted" | "Paid";

const filterChips: Array<{ value: QueueFilter; label: string }> = [
  { value: "all", label: "All" },
  { value: "Draft", label: "Draft" },
  { value: "PendingApproval", label: "Pending" },
  { value: "Approved", label: "Approved" },
  { value: "Posted", label: "Posted" },
  { value: "Paid", label: "Paid" },
];

function carrierName(settlement: CarrierSettlementRow): string {
  if (!settlement.carrier) return "Unknown carrier";
  return settlement.carrier.scac
    ? `${settlement.carrier.name} (${settlement.carrier.scac})`
    : settlement.carrier.name;
}

export function SettlementQueue({
  settlements,
  allSettlements,
  loading,
  filter,
  onFilterChange,
  selectedId,
  onSelect,
  checkedIds,
  onCheckedChange,
  onActionComplete,
}: {
  settlements: CarrierSettlementRow[];
  allSettlements: CarrierSettlementRow[];
  loading: boolean;
  filter: QueueFilter;
  onFilterChange: (filter: QueueFilter) => void;
  selectedId: string | null;
  onSelect: (id: string) => void;
  checkedIds: ReadonlySet<string>;
  onCheckedChange: (ids: ReadonlySet<string>) => void;
  onActionComplete: () => void;
}) {
  const [search, setSearch] = useState("");

  const visible = useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) return settlements;
    return settlements.filter(
      (settlement) =>
        carrierName(settlement).toLowerCase().includes(term) ||
        settlement.settlementNumber.toLowerCase().includes(term),
    );
  }, [settlements, search]);

  const counts = useMemo(() => {
    const byFilter = new Map<QueueFilter, number>();
    for (const chip of filterChips) {
      byFilter.set(chip.value, 0);
    }
    for (const settlement of allSettlements) {
      if (settlement.status === "Voided") continue;
      byFilter.set("all", (byFilter.get("all") ?? 0) + 1);
      const status = settlement.status as QueueFilter;
      if (byFilter.has(status)) {
        byFilter.set(status, (byFilter.get(status) ?? 0) + 1);
      }
    }
    return byFilter;
  }, [allSettlements]);

  const checked = useMemo(
    () => allSettlements.filter((settlement) => checkedIds.has(settlement.id)),
    [allSettlements, checkedIds],
  );

  const toggleChecked = (id: string) => {
    const next = new Set(checkedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    onCheckedChange(next);
  };

  const allVisibleChecked = visible.length > 0 && visible.every((s) => checkedIds.has(s.id));
  const toggleAllVisible = () => {
    if (allVisibleChecked) {
      onCheckedChange(new Set());
      return;
    }
    onCheckedChange(new Set(visible.map((settlement) => settlement.id)));
  };

  return (
    <div className="flex min-h-0 flex-col overflow-hidden rounded-lg border bg-card">
      <div className="flex flex-col gap-2 border-b p-2">
        <div className="relative">
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search carrier or number"
            leftElement={<Search className="size-3.5 text-muted-foreground" />}
            className="h-8 pl-7 text-xs"
            aria-label="Search settlements by carrier name or settlement number"
          />
        </div>
        <div className="flex flex-wrap gap-1">
          {filterChips.map((chip) => {
            const count = counts.get(chip.value) ?? 0;
            return (
              <button
                key={chip.value}
                type="button"
                onClick={() => onFilterChange(chip.value)}
                className={cn(
                  "rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors",
                  filter === chip.value
                    ? "border-primary bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-muted",
                )}
              >
                {chip.label} {count > 0 && <span className="tabular-nums">{count}</span>}
              </button>
            );
          })}
        </div>
      </div>
      <div className="flex items-center gap-2 border-b px-3 py-1.5">
        <Checkbox
          checked={allVisibleChecked}
          onCheckedChange={toggleAllVisible}
          aria-label="Select all visible settlements"
        />
        <span className="text-[11px] text-muted-foreground">
          {checkedIds.size > 0
            ? `${checkedIds.size} selected`
            : "Select settlements to act on several at once"}
        </span>
      </div>
      <ScrollArea
        className="min-h-0 flex-1"
        viewportClassName="min-h-0"
        maskVariant="card"
        maskHeight={18}
      >
        {loading ? (
          <div className="flex flex-col gap-2 p-2">
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-14 w-full" />
          </div>
        ) : visible.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            No settlements match this view.
          </p>
        ) : (
          <ul>
            {visible.map((settlement) => (
              <li key={settlement.id} className="border-b last:border-b-0">
                <div
                  className={cn(
                    "flex w-full items-start gap-2 px-2 py-2 text-left transition-colors",
                    selectedId === settlement.id ? "bg-muted" : "hover:bg-muted/50",
                  )}
                >
                  <Checkbox
                    className="mt-0.5"
                    checked={checkedIds.has(settlement.id)}
                    onCheckedChange={() => toggleChecked(settlement.id)}
                    aria-label={`Select settlement for ${carrierName(settlement)}`}
                  />
                  <button
                    type="button"
                    className="min-w-0 flex-1 text-left"
                    onClick={() => onSelect(settlement.id)}
                  >
                    <div className="flex items-center gap-1.5">
                      <span className="truncate text-xs font-medium">
                        {carrierName(settlement)}
                      </span>
                      <span className="ml-auto text-xs font-semibold tabular-nums">
                        <AmountDisplay
                          value={settlement.netPayableMinor}
                          currency={settlement.currencyCode}
                        />
                      </span>
                    </div>
                    <div className="mt-0.5 flex items-center gap-1.5">
                      <span className="font-mono text-[10px] text-muted-foreground">
                        {settlement.settlementNumber}
                      </span>
                      <CarrierSettlementStatusBadge
                        status={settlement.status as CarrierSettlementStatus}
                      />
                    </div>
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </ScrollArea>
      {checked.length > 0 && (
        <BulkActionBar
          checked={checked}
          onClear={() => onCheckedChange(new Set())}
          onComplete={() => {
            onCheckedChange(new Set());
            onActionComplete();
          }}
        />
      )}
    </div>
  );
}

function BulkActionBar({
  checked,
  onClear,
  onComplete,
}: {
  checked: CarrierSettlementRow[];
  onClear: () => void;
  onComplete: () => void;
}) {
  const [payDialogOpen, setPayDialogOpen] = useState(false);

  const eligibleRows = (action: CarrierSettlementLifecycleAction) =>
    eligibleCarrierSettlements(checked, action);

  const mutation = useMutation({
    mutationFn: async (input: {
      action: CarrierSettlementLifecycleAction;
      paymentMethod?: string;
      paymentReference?: string;
    }) => {
      const eligible = eligibleRows(input.action);
      await runBulkAction(
        eligible,
        (settlement) => {
          switch (input.action) {
            case "Submit":
              return submitCarrierSettlement({ settlementId: settlement.id });
            case "Approve":
              return approveCarrierSettlement({ settlementId: settlement.id });
            case "Post":
              return postCarrierSettlement({ settlementId: settlement.id });
            case "MarkPaid":
              return markCarrierSettlementPaid({
                settlementId: settlement.id,
                paymentMethod: input.paymentMethod ?? "Check",
                paymentReference: input.paymentReference || undefined,
              });
          }
        },
        { noun: "settlement", verb: carrierLifecycleVerbs[input.action] },
      );
    },
    onSuccess: () => onComplete(),
    onError: (error: Error) => toast.error(error.message || "Bulk action failed"),
  });

  const actionButton = (
    action: CarrierSettlementLifecycleAction,
    label: string,
    icon: React.ReactNode,
    onClick?: () => void,
  ) => {
    const count = eligibleRows(action).length;
    if (count === 0) return null;
    return (
      <Button
        size="sm"
        variant="outline"
        className="h-7 text-xs"
        disabled={mutation.isPending}
        onClick={onClick ?? (() => mutation.mutate({ action }))}
        title={`Applies to the ${count} selected settlement${count === 1 ? "" : "s"} in an eligible status; others are skipped`}
      >
        {icon}
        {label} ({count})
      </Button>
    );
  };

  return (
    <div className="flex flex-wrap items-center gap-1.5 border-t bg-muted/40 p-2">
      {actionButton("Submit", "Submit", <Send className="size-3" />)}
      {actionButton("Approve", "Approve", <CheckCheck className="size-3" />)}
      {actionButton("Post", "Post", <CheckCheck className="size-3" />)}
      {actionButton("MarkPaid", "Mark Paid", <CircleDollarSign className="size-3" />, () =>
        setPayDialogOpen(true),
      )}
      <Button
        size="sm"
        variant="ghost"
        className="ml-auto h-7 text-xs text-muted-foreground"
        onClick={onClear}
      >
        Clear
      </Button>
      <BulkMarkPaidDialog
        open={payDialogOpen}
        count={eligibleRows("MarkPaid").length}
        pending={mutation.isPending}
        methods={CARRIER_MARK_PAID_METHODS}
        onOpenChange={setPayDialogOpen}
        onConfirm={(paymentMethod, paymentReference) => {
          setPayDialogOpen(false);
          mutation.mutate({ action: "MarkPaid", paymentMethod, paymentReference });
        }}
      />
    </div>
  );
}
