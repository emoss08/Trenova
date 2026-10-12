import { recordPath } from "@/config/record-links";
import type { InboundMessageDetail, InboundShipmentRef } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery } from "@tanstack/react-query";
import { useContextMenuTrigger } from "@trenova/shared/components/context-menu/context-menu-host";
import {
  AlertTriangleIcon,
  ArrowRightIcon,
  Building07Icon,
  ChevronDownIcon,
  ChevronRightIcon,
  Copy01Icon,
  EyeIcon,
  LinkExternal01Icon,
  Truck01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatUnixDateTimeShort,
  formatUnixTime,
} from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { useNavigate } from "react-router";
import { useTriageActions } from "../use-triage-actions";

const DAY = 86_400;

function lane(shipment: InboundShipmentRef): string {
  return `${shipment.originCity} → ${shipment.destinationCity}`;
}

function RowIcon({ children, open }: { children: React.ReactNode; open?: boolean }) {
  return (
    <span
      aria-hidden
      className={cn(
        "text-foreground-muted flex size-6.5 shrink-0 items-center justify-center rounded-md",
        open ? "bg-canvas" : "bg-surface-hover",
      )}
    >
      {children}
    </span>
  );
}

function ShipmentRow({ message, shipment }: { message: InboundMessageDetail; shipment: InboundShipmentRef }) {
  const t = useT();
  const actions = useTriageActions();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const peek = useQuery(queries.inbox.shipmentPeek(shipment.id)).data;
  const now = Math.floor(Date.now() / 1000);
  const openPeek = () =>
    useTriageStore
      .getState()
      .openOverlay({ kind: "shipmentPeek", messageId: message.id, shipmentId: shipment.id });

  let meta: string | null = null;
  if (peek?.deliveredAt != null) {
    meta = t("Delivered {0}", formatUnixDateTimeShort(peek.deliveredAt));
  } else if (peek?.eta != null) {
    const when = peek.eta - now < DAY ? formatUnixTime(peek.eta) : formatUnixDateTimeShort(peek.eta);
    meta = peek.late ? t("ETA {0} · running late", when) : t("ETA {0}", when);
  }

  const trigger = useContextMenuTrigger(() => ({
    title: shipment.proNumber,
    subtitle: lane(shipment),
    entries: [
      {
        id: "open",
        label: t("Open"),
        icon: LinkExternal01Icon,
        onSelect: () => navigate(recordPath("shipment", shipment.id)),
      },
      { id: "peek", label: t("Peek"), icon: EyeIcon, onSelect: openPeek },
      { type: "separator" },
      {
        id: "pro",
        label: t("Copy PRO"),
        icon: Copy01Icon,
        hint: shipment.proNumber,
        onSelect: () => actions.copyText(shipment.proNumber),
      },
      shipment.bol !== "" && {
        id: "bol",
        label: t("Copy BOL"),
        icon: Copy01Icon,
        hint: shipment.bol,
        onSelect: () => actions.copyText(shipment.bol),
      },
      shipment.poNumber !== "" && {
        id: "po",
        label: t("Copy PO"),
        icon: Copy01Icon,
        hint: shipment.poNumber,
        onSelect: () => actions.copyText(shipment.poNumber),
      },
      {
        id: "unlink",
        label: t("Unlink"),
        icon: XCloseIcon,
        destructive: true,
        onSelect: () => actions.unlink(message.id, "Shipment"),
      },
    ],
  }));

  const details: [string, string][] = [
    [t("Driver"), peek?.driverName || "—"],
    [t("Equipment"), peek?.equipment || "—"],
    [t("BOL"), shipment.bol || "—"],
    [t("PO"), shipment.poNumber || "—"],
    [t("On file"), peek === undefined || peek.documentsOnFile.length === 0 ? "—" : peek.documentsOnFile.join(", ")],
  ];

  return (
    <div className={cn("rounded-lg transition-colors", open && "bg-surface-hover pb-2")}>
      <button
        type="button"
        aria-expanded={open}
        {...trigger}
        onClick={() => setOpen((value) => !value)}
        className={cn(
          "ui-focus-ring flex w-full items-center gap-2.5 rounded-lg p-2 text-left",
          !open && "hover:bg-surface-hover",
        )}
      >
        <RowIcon open={open}>
          <Truck01Icon className="size-3.25" />
        </RowIcon>
        <span className="flex min-w-0 flex-1 flex-col leading-snug">
          <span className="truncate font-mono text-base font-medium">{shipment.proNumber}</span>
          <span className="text-foreground-subtle truncate text-xs">
            {lane(shipment)}
            {meta !== null && (
              <span className={cn(peek?.late === true && "text-warning")}>{t(" · {0}", meta)}</span>
            )}
          </span>
        </span>
        {open ? (
          <ChevronDownIcon aria-hidden className="text-foreground-subtle size-3" />
        ) : (
          <ChevronRightIcon aria-hidden className="text-foreground-subtle size-3" />
        )}
      </button>
      {open && (
        <div className="animate-rise">
          <dl className="flex flex-col pr-2.5 pl-11">
            {details.map(([label, value]) => (
              <div key={label} className="flex justify-between gap-2.5 py-1 text-sm">
                <dt className="text-foreground-subtle">{label}</dt>
                <dd className="truncate text-right">{value}</dd>
              </div>
            ))}
          </dl>
          <button
            type="button"
            onClick={openPeek}
            className="text-foreground-subtle hover:text-foreground ui-focus-ring mt-1 ml-11 inline-flex items-center gap-1 rounded-sm text-xs"
          >
            {t("Open shipment")}
            <ArrowRightIcon aria-hidden className="size-2.75" />
          </button>
        </div>
      )}
    </div>
  );
}

function CustomerRow({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const actions = useTriageActions();
  const navigate = useNavigate();
  const customer = message.matchedCustomer;
  if (customer === null) {
    return null;
  }
  const openPeek = () =>
    useTriageStore
      .getState()
      .openOverlay({ kind: "customerPeek", messageId: message.id, customerId: customer.id });

  return (
    <CustomerButton
      name={customer.name}
      code={customer.code}
      onOpen={openPeek}
      menu={() => ({
        title: customer.name,
        subtitle: t("Customer"),
        entries: [
          {
            id: "open",
            label: t("Open"),
            icon: LinkExternal01Icon,
            onSelect: () => navigate(recordPath("customer", customer.id)),
          },
          { id: "peek", label: t("Peek"), icon: EyeIcon, onSelect: openPeek },
          {
            id: "unlink",
            label: t("Unlink"),
            icon: XCloseIcon,
            destructive: true,
            onSelect: () => actions.unlink(message.id, "Customer"),
          },
        ],
      })}
    />
  );
}

function CustomerButton({
  name,
  code,
  onOpen,
  menu,
}: {
  name: string;
  code: string;
  onOpen: () => void;
  menu: Parameters<typeof useContextMenuTrigger>[0];
}) {
  const t = useT();
  const trigger = useContextMenuTrigger(menu);

  return (
    <button
      type="button"
      {...trigger}
      onClick={onOpen}
      className="hover:bg-surface-hover ui-focus-ring flex w-full items-center gap-2.5 rounded-lg p-2 text-left"
    >
      <RowIcon>
        <Building07Icon className="size-3.25" />
      </RowIcon>
      <span className="flex min-w-0 flex-1 flex-col leading-snug">
        <span className="truncate text-base font-medium">{name}</span>
        <span className="text-foreground-subtle truncate text-xs">
          {code === "" ? t("Customer") : t("Customer · {0}", code)}
        </span>
      </span>
      <LinkExternal01Icon aria-hidden className="text-foreground-subtle size-3" />
    </button>
  );
}

type FactCell = { key: string | null; label: string; value: string; warning: boolean; flagged: boolean };

function Fact({ messageId, fact }: { messageId: string; fact: FactCell }) {
  const t = useT();
  const actions = useTriageActions();
  const trigger = useContextMenuTrigger(() => ({
    title: fact.label,
    subtitle: fact.value,
    entries: [
      {
        id: "copy",
        label: t("Copy value"),
        icon: Copy01Icon,
        onSelect: () => actions.copyText(fact.value),
      },
      {
        id: "copy-labelled",
        label: t("Copy as “{0}: …”", fact.label),
        icon: Copy01Icon,
        onSelect: () => actions.copyText(`${fact.label}: ${fact.value}`),
      },
      fact.key !== null && { type: "separator" },
      fact.key !== null && {
        id: "wrong",
        label: fact.flagged ? t("Flagged as wrong") : t("This is wrong"),
        icon: AlertTriangleIcon,
        hint: fact.flagged ? undefined : t("Nova learns from it"),
        disabled: fact.flagged,
        onSelect: () => {
          if (fact.key !== null) {
            actions.flagFact(messageId, fact.key, fact.label);
          }
        },
      },
    ],
  }));

  return (
    <button
      type="button"
      {...trigger}
      title={t("Copy")}
      onClick={() => actions.copyText(fact.value)}
      className="bg-surface-hover hover:bg-surface-active ui-focus-ring flex min-w-0 flex-col gap-px rounded-md px-2.25 py-1.75 text-left transition-colors"
    >
      <span className="text-foreground-subtle truncate text-2xs">{fact.label}</span>
      <span
        className={cn(
          "truncate text-sm font-medium",
          fact.warning && "text-warning",
          fact.flagged && "line-through decoration-1",
        )}
      >
        {fact.value}
      </span>
    </button>
  );
}

/** What Nova read: from the message's words, or else from its first read document. */
function Facts({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  let title: string;
  let facts: FactCell[];
  if (message.facts.length > 0) {
    title = t("Read from the message");
    facts = message.facts.map((fact) => ({
      key: fact.key,
      label: fact.label,
      value: fact.value,
      warning: fact.tone === "Warning",
      flagged: fact.flagged,
    }));
  } else {
    const read = message.attachments.find((attachment) => (attachment.reading?.fields.length ?? 0) > 0);
    if (read === undefined || read.reading === null) {
      return null;
    }
    title = t("Read from {0}", read.fileName);
    facts = read.reading.fields.slice(0, 4).map((field) => ({
      key: null,
      label: field.label,
      value: field.value,
      warning: field.tone === "Warning",
      flagged: false,
    }));
  }

  return (
    <div className="mt-2.5 flex flex-col gap-2">
      <div className="text-foreground-subtle flex items-center gap-1.5 pl-0.5 text-xs">
        <AssistMark aria-hidden className="text-accent-violet size-3" />
        <span className="truncate">{title}</span>
      </div>
      <div className="grid grid-cols-2 gap-1">
        {facts.map((fact) => (
          <Fact key={`${fact.key ?? fact.label}`} messageId={message.id} fact={fact} />
        ))}
      </div>
    </div>
  );
}

/** What the message tells you: its shipment and customer, and the facts Nova read. */
export function Information({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const shipment = message.matchedShipment;
  const hasFacts =
    message.facts.length > 0 ||
    message.attachments.some((attachment) => (attachment.reading?.fields.length ?? 0) > 0);
  if (shipment === null && message.matchedCustomer === null && !hasFacts) {
    return null;
  }

  return (
    <section aria-label={t("Information")}>
      <h2 className="text-foreground-subtle mb-2 text-xs font-medium">{t("Information")}</h2>
      <div className="flex flex-col gap-0.5">
        {shipment !== null && <ShipmentRow message={message} shipment={shipment} />}
        <CustomerRow message={message} />
      </div>
      <Facts message={message} />
    </section>
  );
}
