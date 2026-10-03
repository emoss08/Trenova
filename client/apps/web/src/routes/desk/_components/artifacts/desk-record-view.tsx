import { formatDisplayValue, statusPhase } from "@/components/assistant/readable-values";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Link } from "react-router";
import type { RecordFact, RecordView, RouteEnd } from "./artifact-payloads";
import { ArtIcon } from "./desk-art-kinds";

const PHASE_PILL: Record<string, string> = {
  failed: "dk-warn",
  attention: "dk-warn",
  awaiting: "dk-blue",
  active: "dk-blue",
  queued: "dk-blue",
  complete: "dk-green",
  closed: "dk-ink",
};

/** Kinds whose title is a name rather than a number, set in the reading face. */
const NAMED = new Set(["customer", "worker"]);

function money(amount: string | number, currency = "USD"): string {
  const figure = Number(amount);
  if (amount === "" || !Number.isFinite(figure)) return String(amount);
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
    minimumFractionDigits: 2,
  }).format(figure);
}

/** "2026-10-02T09:12" as the card says it: the time today, or the day and time otherwise. */
export function localMoment(at: string, now = new Date()): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})[T ](\d{2}):(\d{2})/u.exec(at);
  if (!match) return at;
  const [, year, month, day, hour, minute] = match.map(Number);
  const moment = new Date(year, month - 1, day, hour, minute);
  const time = moment.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  const sameDay =
    moment.getFullYear() === now.getFullYear() &&
    moment.getMonth() === now.getMonth() &&
    moment.getDate() === now.getDate();
  if (sameDay) return time;
  return `${moment.toLocaleDateString(undefined, { month: "short", day: "numeric" })}, ${time}`;
}

function routeTime(end: RouteEnd, origin: boolean, t: TranslateFn): string {
  if (end.at === "") return "";
  const at = localMoment(end.at);
  switch (end.when) {
    case "departed":
      return origin ? t("Picked up {0}", at) : t("Left {0}", at);
    case "arrived":
      return origin ? t("Arrived {0}", at) : t("Delivered {0}", at);
    default:
      return t("Appt {0}", at);
  }
}

function factLabel(key: string, t: TranslateFn): string {
  switch (key) {
    case "driver":
      return t("Driver");
    case "tractor":
      return t("Tractor");
    case "trailer":
      return t("Trailer");
    case "carrier":
      return t("Carrier");
    case "weight":
      return t("Weight");
    case "miles":
      return t("Miles");
    case "rate":
      return t("Rate");
    case "bol":
      return t("BOL");
    case "serviceType":
      return t("Service");
    case "shipmentType":
      return t("Type");
    case "settlement":
      return t("Payment");
    case "dueDate":
      return t("Due");
    case "invoiceDate":
      return t("Invoiced");
    case "paymentTerm":
      return t("Terms");
    case "proNumber":
      return t("Pro number");
    case "sent":
      return t("Sent");
    case "lines":
      return t("Lines");
    case "biller":
      return t("Biller");
    case "billType":
      return t("Bill type");
    case "ageDays":
      return t("Waiting");
    case "exception":
      return t("Exception");
    case "billingCycle":
      return t("Billing cycle");
    case "invoiceDelivery":
      return t("Invoices");
    case "creditStatus":
      return t("Credit");
    case "creditLimit":
      return t("Credit limit");
    case "creditBalance":
      return t("Credit used");
    case "creditHold":
      return t("On hold for");
    case "dotNumber":
      return t("DOT number");
    case "mcNumber":
      return t("MC number");
    case "fleet":
      return t("Fleet");
    case "cdlClass":
      return t("CDL class");
    case "endorsement":
      return t("Endorsements");
    case "compliance":
      return t("Compliance");
    case "physicalDueDate":
      return t("Physical due");
    case "mvrDueDate":
      return t("MVR due");
    case "twicExpiry":
      return t("TWIC expires");
    case "nextTrainingDue":
      return t("Training due");
    case "hireDate":
      return t("Hired");
    case "leave":
      return t("On leave");
    default:
      return humanizeToolName(key);
  }
}

function factValue(fact: RecordFact, currency: string, t: TranslateFn): ReactNode {
  const { key, value } = fact;
  switch (key) {
    case "weight":
      return t("{0} lb", Number(value).toLocaleString());
    case "miles":
      return t("{0} loaded", Math.round(Number(value)).toLocaleString());
    case "rate":
    case "creditLimit":
    case "creditBalance":
      return money(value, currency);
    case "ageDays":
      return t("{0, plural, one {# day} other {# days}}", Number(value));
    case "dueDate":
    case "invoiceDate":
    case "physicalDueDate":
    case "mvrDueDate":
    case "twicExpiry":
    case "nextTrainingDue":
    case "hireDate":
      return typeof value === "number"
        ? new Date(value * 1000).toLocaleDateString(undefined, {
            month: "short",
            day: "numeric",
            year: "numeric",
          })
        : value;
    case "settlement":
    case "sent":
    case "billType":
    case "creditStatus":
    case "compliance":
    case "billingCycle":
    case "invoiceDelivery":
    case "paymentTerm":
      return typeof value === "string" ? formatDisplayValue("status", value, t) : value;
    default:
      return value;
  }
}

/** What the readiness line says: a driver can take a load or not, anything else can be approved or not. */
function readyText(view: RecordView, t: TranslateFn): string {
  const ready = view.ready;
  if (!ready) return "";
  if (view.type === "worker") {
    return ready.canApprove
      ? t("Can take a load")
      : ready.blockedBy || t("Can't be assigned right now");
  }
  return ready.canApprove
    ? t("Ready to approve")
    : ready.blockedBy || t("Not ready to approve yet");
}

/** The route as the design draws it: where it started, where it is going, and the truck between. */
function Route({ from, to, progress }: { from: RouteEnd; to: RouteEnd; progress: number }) {
  const t = useT();
  return (
    <div className="dk-ax-route">
      <div className="dk-ax-stop">
        <i />
        <b>{from.city || from.place}</b>
        {from.city && from.place && <span>{from.place}</span>}
        <em>{routeTime(from, true, t)}</em>
      </div>
      <div className="dk-ax-line" aria-label={t("{0}% of the way", Math.round(progress * 100))}>
        <span className="dk-ax-done" style={{ width: `${progress * 100}%` }} />
        <span className="dk-ax-trk" style={{ left: `${progress * 100}%` }}>
          <ArtIcon name="truck" size={13} />
        </span>
      </div>
      <div className="dk-ax-stop dk-end">
        <i />
        <b>{to.city || to.place}</b>
        {to.city && to.place && <span>{to.place}</span>}
        <em>{routeTime(to, false, t)}</em>
      </div>
    </div>
  );
}

/**
 * A record laid out for its kind. The name, who it is for and its status
 * lead; a shipment shows its route, an invoice or a billing item its amount;
 * then the few facts a person reads that kind of record for, and the way in.
 */
export function DeskRecordView({
  title,
  view,
  entity,
  path,
}: {
  title: string;
  view: RecordView;
  entity: string;
  path: string;
}) {
  const t = useT();
  const statusText = view.status ? formatDisplayValue("status", view.status, t) : "";
  const phase = view.status ? statusPhase(view.status) : null;
  const currency = view.amount?.currency || "USD";

  return (
    <div className="dk-ax-pad dk-ax-rec">
      <div className="dk-ax-rec-h">
        <div>
          <div className={cn("dk-ax-big", NAMED.has(view.type) && "dk-named")}>{title}</div>
          <div className="dk-ax-sub">{view.subtitle || humanizeToolName(entity)}</div>
        </div>
        {statusText !== "" && (
          <span className={cn("dk-ax-pill dk-lg", phase ? PHASE_PILL[phase] : "")}>
            <i />
            {statusText}
          </span>
        )}
      </div>

      {view.from &&
        view.to &&
        (view.from.city || view.from.place) &&
        (view.to.city || view.to.place) && (
          <Route from={view.from} to={view.to} progress={view.progress ?? 0} />
        )}

      {view.amount && view.amount.total !== "" && (
        <div className="dk-ax-amt">
          <b className="dk-ax-num">{money(view.amount.total, currency)}</b>
          {view.amount.balance !== "" && Number(view.amount.balance) > 0 && (
            <span>{t("{0} still due", money(view.amount.balance, currency))}</span>
          )}
        </div>
      )}

      {view.ready && (
        <div className={cn("dk-ax-ready", view.ready.canApprove ? "dk-ok" : "dk-no")}>
          <ArtIcon name={view.ready.canApprove ? "check" : "warn"} size={12} stroke={2.4} />
          {readyText(view, t)}
        </div>
      )}

      {view.facts.length > 0 && (
        <dl className="dk-ax-fields">
          {view.facts.map((fact) => (
            <div key={fact.key}>
              <dt>{factLabel(fact.key, t)}</dt>
              <dd>{factValue(fact, currency, t)}</dd>
            </div>
          ))}
        </dl>
      )}

      {path !== "" && (
        <Link className="dk-ax-btn" to={path}>
          <ArtIcon name="ext" size={13} />
          {t("Open {0}", humanizeToolName(entity).toLowerCase())}
        </Link>
      )}
    </div>
  );
}
