import { useT } from "@trenova/shared/i18n/use-t";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { StatementGroup, StatementShipment } from "@trenova/shared/types/statement";
import { ChevronDownIcon, ChevronRightIcon, PauseCircleIcon, ReceiptTextIcon } from "lucide-react";

/**
 * One invoice this statement will produce, with the shipments that will be on it.
 *
 * Held shipments are unchecked rather than removed: a biller deciding what does
 * not go on this invoice should be able to see what they decided, and change
 * their mind, right up until they bill.
 */
export function StatementGroupCard({
  group,
  currencyCode,
  expanded,
  heldIds,
  onToggleExpanded,
  onToggleShipment,
  onToggleGroup,
}: {
  group: StatementGroup;
  currencyCode: string;
  expanded: boolean;
  heldIds: ReadonlySet<string>;
  onToggleExpanded: () => void;
  onToggleShipment: (shipment: StatementShipment) => void;
  onToggleGroup: (group: StatementGroup) => void;
}) {
  const t = useT();

  const shipments = group.shipments ?? [];
  const included = shipments.filter((s) => !heldIds.has(s.billingQueueItemId));
  const heldCount = shipments.length - included.length;

  // The live total, not the server's: it has to move the instant a biller holds
  // a shipment back, or the number they bill against is not the number they saw.
  const liveTotal = included.reduce((total, s) => total + Number(s.amount ?? 0), 0);
  const allIncluded = shipments.length > 0 && included.length === shipments.length;

  return (
    <div
      className={cn(
        "bg-card overflow-hidden rounded-lg border transition-opacity",
        included.length === 0 && shipments.length > 0 && "opacity-60",
      )}
    >
      <div className="flex items-center gap-2 px-3 py-2">
        {shipments.length > 0 && (
          <Checkbox
            checked={allIncluded}
            indeterminate={!allIncluded && included.length > 0}
            onCheckedChange={() => onToggleGroup(group)}
            aria-label={`Include every shipment on ${group.label}`}
          />
        )}
        <button
          type="button"
          onClick={onToggleExpanded}
          aria-expanded={expanded}
          className="text-muted-foreground hover:text-foreground flex min-w-0 flex-1 items-center gap-2 text-left transition-colors"
        >
          {expanded ? (
            <ChevronDownIcon className="size-3.5 shrink-0" />
          ) : (
            <ChevronRightIcon className="size-3.5 shrink-0" />
          )}
          <ReceiptTextIcon className="size-3.5 shrink-0" />
          <span className="text-foreground truncate text-sm font-medium">{t(group.label)}</span>
        </button>

        <span className="text-muted-foreground shrink-0 text-[11px] tabular-nums">
          {t("{0}{1} shp", included.length, heldCount > 0 ? ` ${t("of {0}", shipments.length)}` : "")}
            </span>
        <span className="shrink-0 text-sm font-semibold tabular-nums">
          {formatCurrency(liveTotal, currencyCode)}
        </span>
        {group.belowMinimum && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span
                  tabIndex={0}
                  className="inline-flex shrink-0 text-amber-600 dark:text-amber-400"
                  aria-label={t("Under the customer's invoice minimum")}
                >
                  <PauseCircleIcon className="size-3.5" />
                </span>
              }
            />
            <TooltipContent side="left">
              {t("Under the customer's invoice minimum. Billing the statement skips this invoice and its shipments stay on next period.")}
            </TooltipContent>
          </Tooltip>
        )}
      </div>

      {expanded && shipments.length > 0 && (
        <div className="divide-y border-t">
          {shipments.map((shipment) => {
            const held = heldIds.has(shipment.billingQueueItemId);
            return (
              <div
                key={shipment.billingQueueItemId}
                className={cn(
                  "flex items-center gap-2 px-3 py-1.5 text-xs",
                  held && "text-muted-foreground",
                )}
              >
                <Checkbox
                  checked={!held}
                  onCheckedChange={() => onToggleShipment(shipment)}
                  aria-label={`Include ${shipment.proNumber ?? "shipment"} on this invoice`}
                />
                <span className={cn("w-28 shrink-0 truncate font-mono", held && "line-through")}>
                  {shipment.proNumber || "—"}
                </span>
                <span className="text-muted-foreground hidden w-28 shrink-0 truncate sm:inline">
                  {shipment.bol ? t("BOL {0}", shipment.bol) : ""}
                </span>
                <span className="text-muted-foreground hidden w-24 shrink-0 truncate md:inline">
                  {shipment.poNumber ? t("PO {0}", shipment.poNumber) : ""}
                </span>
                <span className="text-muted-foreground hidden shrink-0 lg:inline">
                  {shipment.serviceDate ? formatUnixDateMedium(shipment.serviceDate) : ""}
                </span>
                <span
                  className={cn("ml-auto shrink-0 tabular-nums", held && "line-through")}
                >
                  {formatCurrency(Number(shipment.amount ?? 0), currencyCode)}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
