import { leadFacts } from "./reading-lead";
import { FactoringRemittanceStatusBadge } from "@/components/factoring/factoring-badges";
import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import type { InboundMessageDetail, InboundShipmentRef } from "@/lib/graphql/inbox";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { inbox } from "@/lib/queries/inbox";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery } from "@tanstack/react-query";
import { useContextMenuTrigger } from "@trenova/shared/components/context-menu/context-menu-host";
import {
  AlertTriangleIcon,
  ArrowRightIcon,
  Building07Icon,
  CoinsHandIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  Copy01Icon,
  EyeIcon,
  LinkExternal01Icon,
  Truck01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort, formatUnixTime } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { useNavigate } from "react-router";
import { useTriageActions } from "../use-triage-actions";

const DAY = 86_400;

function lane(t: TranslateFn, shipment: InboundShipmentRef): string {
  return t("{0} → {1}", shipment.originCity, shipment.destinationCity);
}

function RowIcon({ children, open }: { children: React.ReactNode; open?: boolean }) {
  return (
    <span
      aria-hidden
      className={cn(
        "flex size-6.5 shrink-0 items-center justify-center rounded-md text-foreground-muted",
        open ? "bg-canvas" : "bg-surface-hover",
      )}
    >
      {children}
    </span>
  );
}

function ShipmentRow({
  message,
  shipment,
}: {
  message: InboundMessageDetail;
  shipment: InboundShipmentRef;
}) {
  const t = useT();
  const actions = useTriageActions();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const peek = useQuery(inbox.shipmentPeek(shipment.id)).data;
  const now = useNowSeconds();
  const openPeek = () =>
    useTriageStore
      .getState()
      .openOverlay({ kind: "shipmentPeek", messageId: message.id, shipmentId: shipment.id });

  let meta: string | null = null;
  if (peek?.deliveredAt != null) {
    meta = t("Delivered {0}", formatUnixDateTimeShort(peek.deliveredAt));
  } else if (peek?.eta != null) {
    const when =
      peek.eta - now < DAY ? formatUnixTime(peek.eta) : formatUnixDateTimeShort(peek.eta);
    meta = peek.late ? t("ETA {0} · running late", when) : t("ETA {0}", when);
  }

  const trigger = useContextMenuTrigger(() => ({
    title: shipment.proNumber,
    subtitle: lane(t, shipment),
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
    [
      t("On file"),
      peek === undefined || peek.documentsOnFile.length === 0
        ? "—"
        : peek.documentsOnFile.join(", "),
    ],
  ];

  return (
    <div className={cn("rounded-lg transition-colors", open && "bg-surface-hover pb-2")}>
      <button
        type="button"
        aria-expanded={open}
        {...trigger}
        onClick={() => setOpen((value) => !value)}
        className={cn(
          "flex w-full items-center gap-2.5 rounded-lg ui-focus-ring p-2 text-left",
          !open && "hover:bg-surface-hover",
        )}
      >
        <RowIcon open={open}>
          <Truck01Icon className="size-3.25" />
        </RowIcon>
        <span className="flex min-w-0 flex-1 flex-col leading-snug">
          <span className="truncate font-mono text-base font-medium">{shipment.proNumber}</span>
          <span className="truncate text-xs text-foreground-subtle">
            {lane(t, shipment)}
            {meta !== null && (
              <span className={cn(peek?.late === true && "text-warning")}>{t(" · {0}", meta)}</span>
            )}
          </span>
        </span>
        {open ? (
          <ChevronDownIcon aria-hidden className="size-3 text-foreground-subtle" />
        ) : (
          <ChevronRightIcon aria-hidden className="size-3 text-foreground-subtle" />
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
            className="mt-1 ml-11 inline-flex items-center gap-1 rounded-sm ui-focus-ring text-xs text-foreground-subtle hover:text-foreground"
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
      className="flex w-full items-center gap-2.5 rounded-lg ui-focus-ring p-2 text-left hover:bg-surface-hover"
    >
      <RowIcon>
        <Building07Icon className="size-3.25" />
      </RowIcon>
      <span className="flex min-w-0 flex-1 flex-col leading-snug">
        <span className="truncate text-base font-medium">{name}</span>
        <span className="truncate text-xs text-foreground-subtle">
          {code === "" ? t("Customer") : t("Customer · {0}", code)}
        </span>
      </span>
      <LinkExternal01Icon aria-hidden className="size-3 text-foreground-subtle" />
    </button>
  );
}

/** The factor remittances this email's attachments were read into, each opening its review. */
function RemittanceRows({ messageId }: { messageId: string }) {
  const t = useT();
  const navigate = useNavigate();
  const { data } = useQuery(factoringQueries.remittances({ inboundMessageId: messageId }));
  if (data === undefined) {
    return null;
  }

  return data.remittances.map((remittance) => (
    <button
      key={remittance.id}
      type="button"
      onClick={() => void navigate(recordPath("factoring_remittance", remittance.id))}
      className="flex w-full items-center gap-2.5 rounded-lg ui-focus-ring p-2 text-left hover:bg-surface-hover"
    >
      <RowIcon>
        <CoinsHandIcon className="size-3.25" />
      </RowIcon>
      <span className="flex min-w-0 flex-1 flex-col leading-snug">
        <span className="truncate text-base font-medium">
          {remittance.fileName || t("Factor remittance")}
        </span>
        <span className="truncate text-xs text-foreground-subtle">
          {t(
            "Remittance · {0}",
            remittance.factoringCompany?.name ?? remittance.factoringCompanyId,
          )}
        </span>
      </span>
      <FactoringRemittanceStatusBadge status={remittance.status} />
      <LinkExternal01Icon aria-hidden className="size-3 text-foreground-subtle" />
    </button>
  ));
}

type FactCell = {
  key: string | null;
  label: string;
  value: string;
  warning: boolean;
  flagged: boolean;
};

function Fact({
  messageId,
  fact,
  lead = false,
}: {
  messageId: string;
  fact: FactCell;
  /** One of the facts this kind of message leads with: drawn larger, on its own row. */
  lead?: boolean;
}) {
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
        onSelect: () => actions.copyText(t("{0}: {1}", fact.label, fact.value)),
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
      className={cn(
        "flex min-w-0 flex-col gap-px rounded-md ui-focus-ring bg-surface-hover px-2.25 text-left transition-colors hover:bg-surface-active",
        lead ? "py-2.5" : "py-1.75",
      )}
    >
      <span className="truncate text-2xs text-foreground-subtle">{fact.label}</span>
      <span
        className={cn(
          "truncate font-medium",
          lead ? "text-lg/tight font-semibold" : "text-sm",
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
    const read = message.attachments.find(
      (attachment) => (attachment.reading?.fields.length ?? 0) > 0,
    );
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

  const led = leadFacts(message.classification, facts);

  return (
    <div className="mt-2.5 flex flex-col gap-2">
      <div className="flex items-center gap-1.5 pl-0.5 text-xs text-foreground-subtle">
        <AssistMark aria-hidden className="size-3 text-accent-violet" />
        <span className="truncate">{title}</span>
      </div>
      {/* What this kind of message is read for comes first and larger: a tender's
          lane, an invoice's amount, the money in a dispute. */}
      {led.lead.length > 0 && (
        <div className="grid grid-cols-2 gap-1">
          {led.lead.map((fact) => (
            <Fact key={`${fact.key ?? fact.label}`} messageId={message.id} fact={fact} lead />
          ))}
        </div>
      )}
      {led.rest.length > 0 && (
        <div className="grid grid-cols-2 gap-1">
          {led.rest.map((fact) => (
            <Fact key={`${fact.key ?? fact.label}`} messageId={message.id} fact={fact} />
          ))}
        </div>
      )}
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
  const { allowed: readsRemittances } = usePermission(Resource.FactoringRemittance, Operation.Read);
  const remittance = readsRemittances && message.classification === "FactorRemittance";
  if (shipment === null && message.matchedCustomer === null && !hasFacts && !remittance) {
    return null;
  }

  return (
    <section aria-label={t("Information")}>
      <h2 className="mb-2 text-xs font-medium text-foreground-subtle">{t("Information")}</h2>
      <div className="flex flex-col gap-0.5">
        {shipment !== null && <ShipmentRow message={message} shipment={shipment} />}
        <CustomerRow message={message} />
        {remittance && <RemittanceRows messageId={message.id} />}
      </div>
      <Facts message={message} />
    </section>
  );
}
