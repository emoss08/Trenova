import { useT } from "@trenova/shared/i18n/use-t";
import { BillingDetailUnselected } from "@/components/billing/billing-empty";
import {
  describeBillingSchedule,
  periodRange,
  standaloneShipmentCount,
} from "@/lib/billing-schedule";
import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostBox, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { StatementGroup, StatementShipment } from "@trenova/shared/types/statement";
import { BotIcon, ClockIcon, InfoIcon, RotateCcwIcon } from "lucide-react";
import { Link } from "react-router";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { invalidateStatements, statementDetailQuery } from "../../statement-queries";
import { BillStatementDialog } from "./bill-statement-dialog";
import { StatementGroupCard } from "./statement-group-card";
import { StatementTimeline } from "./statement-timeline";

/** Beyond a handful of invoices, opening every one buries the totals. */
const AUTO_EXPAND_LIMIT = 3;

export function StatementDetail({
  customerId,
  nowSeconds,
}: {
  customerId: string | null;
  nowSeconds: number;
}) {
  const t = useT();

  const queryClient = useQueryClient();
  const { data: statement, isLoading } = useQuery(statementDetailQuery(customerId));

  // Keyed by customer id at the call site, so switching customers remounts this
  // with fresh hold and expansion state rather than carrying one customer's
  // decisions onto another's statement.
  const groups = useMemo(() => statement?.groups ?? [], [statement?.groups]);
  const [heldIds, setHeldIds] = useState<ReadonlySet<string>>(new Set());
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(
    () => new Set(groups.length <= AUTO_EXPAND_LIMIT ? groups.map((group) => group.key) : []),
  );
  const [billOpen, setBillOpen] = useState(false);

  const liveTotal = useMemo(
    () =>
      groups.reduce((total, group) => {
        if (group.belowMinimum) return total;
        return (
          total +
          (group.shipments ?? []).reduce(
            (sum, shipment) =>
              heldIds.has(shipment.billingQueueItemId)
                ? sum
                : sum + Number(shipment.amount ?? 0),
            0,
          )
        );
      }, 0),
    [groups, heldIds],
  );

  const bill = useMutation({
    mutationFn: (reason: string) =>
      apiService.invoiceRunService.billStatementNow(customerId as string, reason, [...heldIds]),
    onSuccess: (result) => {
      setBillOpen(false);
      setHeldIds(new Set());
      if (result.errorCount === 0 && result.skippedCount === 0) {
        toast.success(
          `${result.successCount} invoice${result.successCount === 1 ? "" : "s"} created`,
        );
      } else {
        const firstError = result.results.find((entry) => entry.error)?.error;
        toast.warning(
          `${result.successCount} created, ${result.skippedCount} skipped, ${result.errorCount} failed` +
            (firstError ? ` — ${firstError}` : ""),
        );
      }
      invalidateStatements(queryClient);
    },
    onError: (error: Error) => toast.error(error.message || "Could not bill this statement"),
  });

  if (isLoading) {
    return (
      <div className="bg-card flex min-h-0 flex-col gap-3 overflow-hidden p-4">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-14 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  if (!statement) {
    return (
      <BillingDetailUnselected
        layout="cards"
        title={t("Pick a statement")}
        description={t("See what a customer has accumulated this period, which invoices it becomes, and bill it early if you have to.")}
      />
    );
  }

  const offCycle = statement.periodEnd > nowSeconds;
  const standalone = standaloneShipmentCount(statement);
  const heldCount = heldIds.size;
  const billable = groups.filter((group) => !group.belowMinimum);

  function toggleShipment(shipment: StatementShipment) {
    setHeldIds((previous) => {
      const next = new Set(previous);
      if (next.has(shipment.billingQueueItemId)) next.delete(shipment.billingQueueItemId);
      else next.add(shipment.billingQueueItemId);
      return next;
    });
  }

  function toggleGroup(group: StatementGroup) {
    const shipments = group.shipments ?? [];
    const allIncluded = shipments.every((s) => !heldIds.has(s.billingQueueItemId));
    setHeldIds((previous) => {
      const next = new Set(previous);
      for (const shipment of shipments) {
        if (allIncluded) next.add(shipment.billingQueueItemId);
        else next.delete(shipment.billingQueueItemId);
      }
      return next;
    });
  }

  return (
    <div className="bg-card flex h-full min-h-0 flex-col overflow-hidden">
      <div className="flex flex-wrap items-center gap-2 border-b px-3 py-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold">{statement.customerName}</p>
          <p className="text-muted-foreground truncate text-[11px]">
            {periodRange(statement.periodStart, statement.periodEnd)}
            {statement.customerCode ? ` · ${statement.customerCode}` : ""}
          </p>
        </div>
        {statement.autoBill && (
          <Tooltip>
            <TooltipTrigger
              render={
                <Badge variant="outline" tabIndex={0} className="gap-1">
                  <BotIcon className="size-3" />
                  {t("Auto-bills")}
                </Badge>
              }
            />
            <TooltipContent>
              {t("This statement bills itself when the period closes. Billing it here is only for getting ahead of that.")}
            </TooltipContent>
          </Tooltip>
        )}
        <span className="ml-auto text-base font-semibold tabular-nums">
          {formatCurrency(liveTotal, statement.currencyCode)}
        </span>
      </div>

      <div className="border-b">
        <StatementTimeline statement={statement} nowSeconds={nowSeconds} />
      </div>

      <ScrollArea className="min-h-0 flex-1" viewportClassName="min-h-0" maskVariant="card">
        {groups.length === 0 ? (
          <EmptySheet
            title={t("Nothing on this statement yet")}
            description={`No approved, uninvoiced shipment for ${statement.customerName} was delivered in ${periodRange(statement.periodStart, statement.periodEnd)}. Approving one in the Shipments view puts it here straight away.`}
            sketch={
              <div className="flex flex-col gap-2">
                {[0, 1].map((card) => (
                  <div key={card} className="rounded-lg border p-3">
                    <div className="flex items-center gap-2">
                      <GhostBox className="size-4" />
                      <GhostLine className="w-32" />
                      <GhostBar className="ml-auto w-16" />
                    </div>
                    <div className="mt-2 flex flex-col gap-1.5">
                      <GhostLine className="w-full" />
                      <GhostLine className="w-4/5" />
                    </div>
                  </div>
                ))}
              </div>
            }
          />
        ) : (
          <div className="flex flex-col gap-2 p-2">
            {standalone > 1 && (
              <div className="bg-muted/40 flex items-start gap-2 rounded-lg border p-2.5">
                <InfoIcon className="text-muted-foreground mt-0.5 size-3.5 shrink-0" />
                <p className="text-muted-foreground text-xs">
                  {standalone} shipments aren&apos;t booked as orders, so splitting by order bills
                  each on its own invoice. If {statement.customerName} expects one invoice for the
                  period,{" "}
                  <Link
                    to={`/billing/configuration-files/customers?panelType=edit&panelEntityId=${statement.customerId}`}
                    className="text-foreground font-medium underline underline-offset-2"
                  >
                    change how they&apos;re split
                  </Link>
                  .
                </p>
              </div>
            )}
            {statement.heldCount > 0 && (
              <div className="flex items-start gap-2 rounded-lg border border-amber-300 bg-amber-50/60 p-2.5 dark:border-amber-900 dark:bg-amber-950/30">
                <ClockIcon className="mt-0.5 size-3.5 shrink-0 text-amber-600 dark:text-amber-400" />
                <p className="text-xs text-amber-800 dark:text-amber-200">
                  {statement.heldCount} shipment{statement.heldCount === 1 ? "" : "s"} worth{" "}
                  {formatCurrency(Number(statement.heldAmount ?? 0), statement.currencyCode)} are
                  still in review and will not be on this invoice.{" "}
                  <Link
                    to={`/billing/queue?view=shipments&query=${encodeURIComponent(statement.customerName)}`}
                    className="font-medium underline underline-offset-2"
                  >
                    Review them
                  </Link>
                </p>
              </div>
            )}
            <p className="text-muted-foreground px-1 text-[11px]">
              {describeBillingSchedule({
                invoiceDelivery: "Consolidated",
                billingCycle: statement.cycle,
                billingCycleAnchorDay: statement.billingCycleAnchorDay,
                splitBy: statement.splitBy,
                sectionBy: statement.sectionBy,
                invoiceDetail: statement.detail,
                maxShipmentsPerInvoice: 0,
              })}
            </p>
            {groups.map((group) => (
              <StatementGroupCard
                key={group.key}
                group={group}
                currencyCode={statement.currencyCode}
                expanded={expanded.has(group.key)}
                heldIds={heldIds}
                onToggleExpanded={() =>
                  setExpanded((previous) => {
                    const next = new Set(previous);
                    if (next.has(group.key)) next.delete(group.key);
                    else next.add(group.key);
                    return next;
                  })
                }
                onToggleShipment={toggleShipment}
                onToggleGroup={toggleGroup}
              />
            ))}
          </div>
        )}
      </ScrollArea>

      {groups.length > 0 && (
        <div className="bg-muted/40 flex flex-wrap items-center gap-2 border-t p-2">
          <span className="text-muted-foreground text-[11px]">
            {heldCount > 0
              ? t("{0, plural, one {# shipment} other {# shipments}} held back", heldCount)
              : t("{0, plural, one {# shipment} other {# shipments}} ready", statement.shipmentCount)}
          </span>
          {heldCount > 0 && (
            <Button
              size="sm"
              variant="ghost"
              className="h-7 text-xs"
              onClick={() => setHeldIds(new Set())}
            >
              <RotateCcwIcon className="size-3" />
              {t("Put them back")}
            </Button>
          )}
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="ml-auto inline-flex">
                  <Button
                    size="sm"
                    disabled={billable.length === 0 || statement.shipmentCount === 0}
                    onClick={() => setBillOpen(true)}
                  >
                    {offCycle ? t("Bill early") : t("Bill now")}
                  </Button>
                </span>
              }
            />
            <TooltipContent side="left">
              {statement.shipmentCount === 0
                ? t("Nothing has accrued to this statement yet")
                : billable.length === 0
                  ? t("Every invoice on this statement is under the customer's minimum")
                  : offCycle
                    ? t("Bills this period before it closes, without moving the customer's cycle")
                    : t("This period has closed — bill it")}
            </TooltipContent>
          </Tooltip>
        </div>
      )}

      <BillStatementDialog
        open={billOpen}
        statement={statement}
        groups={groups}
        heldCount={heldCount}
        total={liveTotal}
        offCycle={offCycle}
        pending={bill.isPending}
        onOpenChange={setBillOpen}
        onConfirm={({ reason }) => bill.mutate(reason)}
      />
    </div>
  );
}
