import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type {
  BillingQueueItem,
  BillingQueueStatus,
  PayerShareLine,
  ReassignChargeResult,
} from "@trenova/shared/types/billing-queue";
import type { ChargeAllocation, Shipment } from "@trenova/shared/types/shipment";

/**
 * The field the server reports an amount split on when a charge edit leaves it
 * no longer adding up. Retrying with this flag set converts the split to
 * percentages.
 */
export const STALE_AMOUNT_SPLIT_FIELD = "convertAmountSplitsToPercent";

/** Statuses in which no invoice can exist yet, so a charge may still change hands. */
const REASSIGNABLE_STATUSES: ReadonlySet<BillingQueueStatus> = new Set([
  "ReadyForReview",
  "InReview",
  "OnHold",
]);

export function isChargeReassignable(status: BillingQueueStatus): boolean {
  return REASSIGNABLE_STATUSES.has(status);
}

/** The server's messages for each amount split an edit left stale; empty for any other failure. */
export function staleAmountSplitMessages(error: unknown): string[] {
  if (!(error instanceof ApiRequestError)) return [];
  return error
    .getFieldErrors(STALE_AMOUNT_SPLIT_FIELD)
    .map((fieldError) => fieldError.message)
    .filter((message): message is string => Boolean(message));
}

/** A payer's share of a charge as people read it: "47.37%". */
export function formatSharePercent(percent: number | null | undefined): string {
  if (percent == null) return "";
  return `${Number(percent.toFixed(2))}%`;
}

/** Who pays a charge, for a line billed to someone else: "Acme Manufacturing" or "Peak 52.63% · Acme 47.37%". */
export function describeLinePayers(line: PayerShareLine): string {
  if (line.payers.length === 1) return line.payers[0].payerName;
  return line.payers
    .map((party) => `${party.payerName} ${formatSharePercent(party.percent)}`.trim())
    .join(" · ");
}

/**
 * The allocation rows that divide one charge today, carrying payer names from
 * the resolved share so the split editor can label them without a lookup.
 */
export function allocationsForCharge(
  shipment: Pick<Shipment, "chargeAllocations"> | null | undefined,
  line: Pick<PayerShareLine, "kind" | "additionalChargeId" | "payers">,
): ChargeAllocation[] {
  const names = new Map(line.payers.map((party) => [party.payerId, party]));
  return [...(shipment?.chargeAllocations ?? [])]
    .filter((row) =>
      line.kind === "Freight"
        ? row.chargeKind === "Freight"
        : row.chargeKind === "Accessorial" && row.additionalChargeId === line.additionalChargeId,
    )
    .sort((a, b) => (a.sequence ?? 0) - (b.sequence ?? 0))
    .map((row) => {
      const party = names.get(row.billToCustomerId);
      return {
        ...row,
        billToCustomer:
          row.billToCustomer ??
          (party
            ? { id: party.payerId, name: party.payerName, code: party.payerCode ?? null }
            : null),
      };
    });
}

/** What the reassignment did to the queue, for the confirmation toast. */
export function reassignSummary(
  t: TranslateFn,
  chargeName: string,
  result: Pick<ReassignChargeResult, "createdItemIds" | "canceledItemIds">,
): string {
  const parts = [t("{0} moved.", chargeName)];
  if (result.createdItemIds.length > 0) {
    parts.push(
      t(
        "{0, plural, one {# queue item created} other {# queue items created}}.",
        result.createdItemIds.length,
      ),
    );
  }
  if (result.canceledItemIds.length > 0) {
    parts.push(
      t(
        "{0, plural, one {# queue item canceled} other {# queue items canceled}}.",
        result.canceledItemIds.length,
      ),
    );
  }
  return parts.join(" ");
}

/**
 * Where the queue should land after a reassignment canceled the item on screen:
 * another active item for the same shipment, or nothing.
 */
export function nextItemAfterReassignment(
  current: Pick<BillingQueueItem, "id" | "shipmentId">,
  result: Pick<ReassignChargeResult, "items" | "canceledItemIds">,
): string | null | undefined {
  if (!result.canceledItemIds.includes(current.id)) return undefined;
  const next = result.items.find(
    (row) => row.id !== current.id && row.shipmentId === current.shipmentId,
  );
  return next?.id ?? null;
}
