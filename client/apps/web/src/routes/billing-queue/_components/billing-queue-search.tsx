import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { billingQueue } from "@/lib/queries/billing-queue";
import { billingQueueService } from "@/services/billing-queue";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Input } from "@trenova/shared/components/ui/input";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { SearchLgIcon, XCloseIcon } from "@trenova/shared/components/icons";
import {
  dateWindowFromSearch,
  dateWindowLabel,
  todayInUserCalendar,
  type QueueDateField,
} from "@/lib/billing-queue-date-window";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { useNow } from "@trenova/shared/hooks/use-now";
import type { BillingQueueSearchIntent } from "@trenova/shared/types/billing-queue";
import { BILLING_QUEUE_OPEN_LANES } from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useState } from "react";
import { queueSidebarSearchParamsParser } from "../use-billing-queue-state";
import { CLEARED_QUEUE_FILTERS } from "./billing-queue-filters";
import { laneLabel } from "./billing-queue-lanes";

/** Three words or more reads as a question for Nova rather than a number to look up. */
export function isSentence(text: string): boolean {
  return text.trim().split(/\s+/u).filter(Boolean).length >= 3;
}

type Chip = { key: string; label: string; clear: Partial<typeof CLEARED_QUEUE_FILTERS> };

/**
 * The queue's search. A PRO, BOL or name filters as it is typed; with Nova on, a
 * sentence ("Acme over $5k delivered last week") is read into filters on Enter.
 * Every filter in force shows as a chip that removes itself, so what Nova read is
 * always visible and correctable.
 */
export function BillingQueueSearch() {
  const t = useT();
  const now = useNow("minute");
  const [params, setParams] = useQueryStates(queueSidebarSearchParamsParser);
  const [payerName, setPayerName] = useState<string | null>(null);
  const [customerName, setCustomerName] = useState<string | null>(null);
  const [unresolved, setUnresolved] = useState<string | null>(null);
  const { data: assist } = useQuery(billingQueue.assist());
  const canAsk = assist?.search === true;
  const asking = canAsk && isSentence(params.query);

  const compose = useApiMutation({
    mutationFn: (text: string) => billingQueueService.composeSearch(text, todayInUserCalendar()),
    resourceName: "BillingQueueItem",
    onSuccess: (intent: BillingQueueSearchIntent) => {
      setPayerName(intent.payerName ?? null);
      setCustomerName(intent.shipmentCustomerName ?? null);
      setUnresolved(intent.unresolved || null);
      void setParams({
        ...CLEARED_QUEUE_FILTERS,
        includeClosed: params.includeClosed,
        query: intent.search,
        lanes: intent.lanes.filter((lane) =>
          (BILLING_QUEUE_OPEN_LANES as readonly string[]).includes(lane),
        ) as (typeof BILLING_QUEUE_OPEN_LANES)[number][],
        payer: intent.payerId ?? null,
        minAmount: intent.minAmount != null ? String(intent.minAmount) : null,
        maxAmount: intent.maxAmount != null ? String(intent.maxAmount) : null,
        createdBefore: intent.createdBefore ?? null,
        shipmentCustomer: intent.shipmentCustomerId ?? null,
        delivered: dateWindowFromSearch(intent.delivered),
        pickedUp: dateWindowFromSearch(intent.pickedUp),
        queued: dateWindowFromSearch(intent.queued),
      });
    },
  });

  const chips: Chip[] = [];
  for (const lane of params.lanes) {
    chips.push({
      key: `lane-${lane}`,
      label: laneLabel(lane, t),
      clear: { lanes: params.lanes.filter((value) => value !== lane) },
    });
  }
  if (params.payer) {
    chips.push({ key: "payer", label: payerName ?? t("One payer"), clear: { payer: null } });
  }
  if (params.minAmount && params.maxAmount) {
    chips.push({
      key: "amount",
      label: t(
        "{0} to {1}",
        formatCurrency(Number(params.minAmount)),
        formatCurrency(Number(params.maxAmount)),
      ),
      clear: { minAmount: null, maxAmount: null },
    });
  } else if (params.minAmount) {
    chips.push({
      key: "amount",
      label: t("Over {0}", formatCurrency(Number(params.minAmount))),
      clear: { minAmount: null },
    });
  } else if (params.maxAmount) {
    chips.push({
      key: "amount",
      label: t("Under {0}", formatCurrency(Number(params.maxAmount))),
      clear: { maxAmount: null },
    });
  }
  if (params.shipmentCustomer) {
    chips.push({
      key: "customer",
      label: customerName ? t("Shipments for {0}", customerName) : t("One customer"),
      clear: { shipmentCustomer: null },
    });
  }
  const dateClears = {
    delivered: { delivered: null },
    pickedUp: { pickedUp: null },
    queued: { queued: null },
  } satisfies Record<QueueDateField, Chip["clear"]>;
  for (const field of ["delivered", "pickedUp", "queued"] as const) {
    const span = params[field];
    if (span) {
      chips.push({ key: field, label: dateWindowLabel(field, span, t), clear: dateClears[field] });
    }
  }
  if (params.createdBefore) {
    chips.push({
      key: "age",
      label: t(
        "{0, plural, one {Waiting over # day} other {Waiting over # days}}",
        Math.max(1, Math.round((now - params.createdBefore) / 86400)),
      ),
      clear: { createdBefore: null },
    });
  }

  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1.5">
      <form
        onSubmit={(event) => {
          event.preventDefault();
          if (asking && !compose.isPending) {
            compose.mutate(params.query.trim());
          }
        }}
      >
        <Input
          aria-label={canAsk ? t("Search or ask Nova") : t("Search the queue")}
          placeholder={
            canAsk
              ? t("Search, or ask: “Acme over $5k delivered this week”")
              : t("Search PRO, BOL, PO, customer…")
          }
          leftElement={
            canAsk ? (
              <NovaBadge size="xs" spin={compose.isPending ? "always" : "none"} />
            ) : (
              <SearchLgIcon className="text-muted-foreground size-3.5" />
            )
          }
          value={params.query}
          onChange={(event) => {
            setUnresolved(null);
            void setParams({ query: event.target.value, preset: null });
          }}
          className="h-7.5 text-sm"
          inputContainerClassName="w-full"
        />
      </form>
      {asking ? (
        <p className="text-foreground-muted px-1 text-xs">
          {compose.isPending ? (
            <TextShimmer as="span">{t("Nova is reading that…")}</TextShimmer>
          ) : (
            t("Press Enter to ask Nova")
          )}
        </p>
      ) : null}
      {chips.length > 0 || unresolved ? (
        <div className="flex flex-wrap items-center gap-1">
          {chips.map((chip) => (
            <button
              key={chip.key}
              type="button"
              onClick={() => void setParams({ ...chip.clear, preset: null })}
              aria-label={t("Remove filter {0}", chip.label)}
              className="ui-focus-ring bg-muted text-foreground hover:bg-surface-hover rounded-badge inline-flex items-center gap-1 px-2 py-0.5 text-xs"
            >
              {chip.label}
              <XCloseIcon className="text-foreground-subtle size-3" />
            </button>
          ))}
          {unresolved ? (
            <span className="text-foreground-muted text-xs">
              {t("Nova couldn't use: {0}", unresolved)}
            </span>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
