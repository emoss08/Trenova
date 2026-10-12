import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { recordPath } from "@/config/record-links";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { TriageCustomerPeek } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery } from "@tanstack/react-query";
import {
  Inbox01Icon,
  LinkExternal01Icon,
  Truck01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { useNavigate } from "react-router";
import { inlineErrorText } from "../../inline-error";
import { peekPhase } from "./peek-parts";

/**
 * A customer the message is about: how busy they are this week, what of
 * theirs waits in Triage, their shipments moving now and their latest mail.
 * A shipment opens its own peek; a message opens in the reader.
 */
export function CustomerPeek({
  customerId,
  messageId,
  onClose,
  onOpenMessage,
  onOpenWaiting,
}: {
  customerId: string;
  messageId: string | null;
  onClose: () => void;
  onOpenMessage: (id: string) => void;
  /** Opens what of this customer's waits in Triage, searched by their name. */
  onOpenWaiting: (customerName: string) => void;
}) {
  const t = useT();
  const query = useQuery(queries.inbox.customerPeek(customerId));
  const customer = query.data;

  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent size="sm" showCloseButton={false} className="gap-3.5">
        {customer === undefined ? (
          query.isError ? (
            <>
              <DialogTitle>{t("Customer")}</DialogTitle>
              <p role="alert" className="text-danger text-sm">
                {inlineErrorText(t, query.error)}
              </p>
            </>
          ) : (
            <>
              <DialogTitle className="sr-only">{t("Customer")}</DialogTitle>
              <Skeleton className="h-6 w-1/2" />
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-24 w-full" />
            </>
          )
        ) : (
          <CustomerBody
            customer={customer}
            messageId={messageId}
            onClose={onClose}
            onOpenMessage={onOpenMessage}
            onOpenWaiting={onOpenWaiting}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

const ROW =
  "ui-focus-ring hover:bg-surface-hover -mx-2 flex h-8 w-[calc(100%+1rem)] items-center gap-2.25 rounded-md px-2 text-left text-sm";

function CustomerBody({
  customer,
  messageId,
  onClose,
  onOpenMessage,
  onOpenWaiting,
}: {
  customer: TriageCustomerPeek;
  messageId: string | null;
  onClose: () => void;
  onOpenMessage: (id: string) => void;
  onOpenWaiting: (customerName: string) => void;
}) {
  const t = useT();
  const navigate = useNavigate();
  const now = useNowSeconds();

  return (
    <>
      <header className="flex items-start gap-2.5">
        <div className="flex min-w-0 flex-1 flex-col">
          <DialogTitle className="truncate text-xl font-semibold">{customer.name}</DialogTitle>
          <span className="text-foreground-subtle text-sm">
            {t(
              "Customer since {0}",
              formatUnixInUserTimezone(customer.since, { month: "short", year: "numeric" }, ""),
            )}
          </span>
        </div>
        <Button variant="ghost" size="icon-sm" aria-label={t("Close")} onClick={onClose}>
          <XCloseIcon className="size-3.5" />
        </Button>
      </header>

      <KpiStrip aria-label={t("This week")}>
        <KpiStripItem label={t("Shipments this week")} value={customer.shipmentsThisWeek} />
        <KpiStripItem
          label={t("Waiting in Triage")}
          value={customer.waitingInTriage}
          tone={customer.waitingInTriage > 0 ? "warning" : undefined}
          onClick={
            customer.waitingInTriage > 0
              ? () => {
                  onClose();
                  onOpenWaiting(customer.name);
                }
              : undefined
          }
        />
      </KpiStrip>

      {customer.shipments.length > 0 && (
        <section className="flex flex-col">
          <h3 className="text-foreground-subtle pb-1 text-xs">{t("Shipments")}</h3>
          {customer.shipments.map((shipment) => {
            const origin = shipment.stops[0]?.city ?? "";
            const destination = shipment.stops.at(-1)?.city ?? "";
            const delivered = peekPhase(shipment) === "delivered";
            return (
              <button
                key={shipment.id}
                type="button"
                onClick={() =>
                  useTriageStore.getState().openOverlay({
                    kind: "shipmentPeek",
                    shipmentId: shipment.id,
                    messageId: messageId ?? "",
                  })
                }
                className={ROW}
              >
                <Truck01Icon aria-hidden className="text-foreground-subtle size-3.25 shrink-0" />
                <span className="min-w-0 flex-1 truncate">
                  {t("{0} · {1} → {2}", shipment.proNumber, origin, destination)}
                </span>
                <span className="text-foreground-subtle font-mono text-2xs">
                  {delivered ? t("done") : t("moving")}
                </span>
              </button>
            );
          })}
        </section>
      )}

      <section className="flex flex-col">
        <h3 className="text-foreground-subtle pb-1 text-xs">{t("Messages")}</h3>
        {customer.messages.length === 0 ? (
          <p className="text-foreground-subtle text-sm">{t("No mail from them yet")}</p>
        ) : (
          customer.messages.map((message) => (
            <button
              key={message.messageId}
              type="button"
              onClick={() => {
                onClose();
                onOpenMessage(message.messageId);
              }}
              className={ROW}
            >
              <Inbox01Icon aria-hidden className="text-foreground-subtle size-3.25 shrink-0" />
              <span className="min-w-0 flex-1 truncate">
                {message.subject.trim() === "" ? t("(no subject)") : message.subject}
              </span>
              <span className="text-foreground-subtle font-mono text-2xs">
                {formatSecondsAgo(now - message.receivedAt)}
              </span>
            </button>
          ))
        )}
      </section>

      <footer className="border-border-subtle flex items-center gap-1.5 border-t pt-3">
        <span className="flex-1" />
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            onClose();
            void navigate(recordPath("customer", customer.id));
          }}
        >
          {t("Open customer")}
          <LinkExternal01Icon className="size-3.5" />
        </Button>
      </footer>
    </>
  );
}
