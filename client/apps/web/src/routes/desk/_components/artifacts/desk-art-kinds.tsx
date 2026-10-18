import type { AssistantArtifact } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { defineLabels } from "@trenova/shared/i18n/labels";

const ICONS = {
  shipment: (
    <>
      <path d="M3.5 6.5h10v9h-10zM13.5 9.5h4l3 3v3h-7" />
      <circle cx="7" cy="17" r="1.6" />
      <circle cx="17" cy="17" r="1.6" />
    </>
  ),
  tractor: (
    <>
      <path d="M4 16.5V9.5h6.5l2.5-3.5h3.5l3.5 4.5v6h-2" />
      <path d="M13 6v5.5h7M4 16.5h3M11 16.5h3.5" />
      <circle cx="9" cy="17" r="1.8" />
      <circle cx="16.5" cy="17" r="1.8" />
    </>
  ),
  trailer: (
    <>
      <path d="M3.5 6.5h14v9.5h-14zM17.5 13h3" />
      <path d="M3.5 16h2M10.5 16h3" />
      <circle cx="8" cy="17" r="1.8" />
      <circle cx="13" cy="17" r="1.8" />
    </>
  ),
  worker: (
    <>
      <circle cx="12" cy="8" r="3.5" />
      <path d="M5 20.5c.6-3.9 3.3-6 7-6s6.4 2.1 7 6" />
    </>
  ),
  location: (
    <>
      <path d="M12 20.5s-6-5.4-6-10.3a6 6 0 0 1 12 0c0 4.9-6 10.3-6 10.3z" />
      <circle cx="12" cy="10" r="2.2" />
    </>
  ),
  carrier: (
    <>
      <path d="M2.5 7.5h11v8h-11zM13.5 10h3.5l3 3v2.5h-6.5" />
      <path d="M6 4.5h4.5M8.25 2.5v4" />
      <circle cx="6.5" cy="17" r="1.6" />
      <circle cx="16.5" cy="17" r="1.6" />
    </>
  ),
  order: (
    <>
      <path d="M6.5 4.5h11v16h-11z" />
      <path d="M9.5 3v3M14.5 3v3M9.5 10h5M9.5 13.5h5M9.5 17h3" />
    </>
  ),
  failure: (
    <>
      <circle cx="10.5" cy="12.5" r="7.5" />
      <path d="M10.5 8.5v4.2l2.8 1.8M19.5 4v5M19.5 12v.01" />
    </>
  ),
  edi: (
    <>
      <path d="M3.5 6.5h17v11h-17z" />
      <path d="M3.5 7.5l8.5 5.5 8.5-5.5M7 15h2.5M14.5 15H17" />
    </>
  ),
  inbound: (
    <>
      <path d="M6.5 3.5h7.5l4 4v13h-11.5z" />
      <path d="M14 3.5v4h4M12.25 10v6.5M9.5 14l2.75 2.75L15 14" />
    </>
  ),
  invoice: (
    <>
      <path d="M6 3.5h12v17l-2-1.4-2 1.4-2-1.4-2 1.4-2-1.4-2 1.4z" />
      <path d="M9 8h6M9 11.5h6M9 15h3.5" />
    </>
  ),
  pay: (
    <>
      <path d="M3.5 6h17v12h-17z" />
      <path d="M7.5 6v12" strokeDasharray="1.6 1.9" />
      <path d="M11 10h6M11 14h4" />
    </>
  ),
  payable: (
    <>
      <path d="M3.5 6h17v12h-17z" />
      <path d="M7.5 6v12" strokeDasharray="1.6 1.9" />
      <path d="M11 12h6M15 10l2 2-2 2" />
    </>
  ),
  detention: (
    <>
      <circle cx="12" cy="13" r="7.5" />
      <path d="M12 9v4l2.75 1.75M10 3.5h4" />
    </>
  ),
  journal: (
    <>
      <path d="M4.5 4.5h15v15h-15z" />
      <path d="M12 4.5v15M7 8.5h3M7 12h3M14 12h3M14 15.5h3" />
    </>
  ),
  manual: (
    <>
      <path d="M5 4.5h11v15H5z" />
      <path d="M8 8.5h5M8 12h3M13.5 17.25l5.25-5.25 1.75 1.75-5.25 5.25H13.5z" />
    </>
  ),
  partner: (
    <>
      <path d="M4 5.5h16v13H4z" />
      <path d="M9 9v6.5M6.75 13.25 9 15.5l2.25-2.25M15 15.5V9M12.75 11.25 15 9l2.25 2.25" />
    </>
  ),
  tender: (
    <>
      <path d="M4.5 6.5h15v3a2.5 2.5 0 0 0 0 5v3h-15v-3a2.5 2.5 0 0 0 0-5z" />
      <path d="M14.5 6.5v11" strokeDasharray="1.6 1.9" />
      <path d="M8 12h3.5M10 10l1.8 2-1.8 2" />
    </>
  ),
  agreement: (
    <>
      <path d="M6 3.5h12v17H6z" />
      <path d="M9 8h6M9 11.5h6M8.8 16.8c.9-1.4 1.8-1.4 2.3 0s1.5 1.3 3.9-.6" />
    </>
  ),
  customer: (
    <>
      <path d="M4.5 20.5V7l7.5-3.5L19.5 7v13.5" />
      <path d="M3 20.5h18M9 9.5h1.5M13.5 9.5H15M9 13h1.5M13.5 13H15M10.5 20.5V17h3v3.5" />
    </>
  ),
  formula: (
    <>
      <rect x="3.5" y="4.5" width="17" height="15" rx="2.5" />
      <path d="M11 8.5c-1.3 0-1.9.7-2.1 2l-.8 4.8c-.2 1.3-.8 1.9-2.1 1.9M7.2 12h3.6M13.2 11l3.6 4.5M16.8 11l-3.6 4.5" />
    </>
  ),
  bill: (
    <>
      <path d="M6 3.5h12v17l-2-1.3-2 1.3-2-1.3-2 1.3-2-1.3-2 1.3z" />
      <path d="M9.5 8.5h5M9.5 12h5M12 14.5v2" />
    </>
  ),
  table: (
    <>
      <rect x="3.5" y="5" width="17" height="14" rx="2" />
      <path d="M3.5 10h17M9 10v9" />
    </>
  ),
  record: (
    <>
      <rect x="3.5" y="5" width="17" height="14" rx="2" />
      <circle cx="9" cy="11" r="2" />
      <path d="M6 16c.6-1.4 1.7-2 3-2s2.4.6 3 2M14.5 10h3M14.5 13.5h3" />
    </>
  ),
  rate: (
    <>
      <path d="M6 3.5h12v17l-3-1.8-3 1.8-3-1.8-3 1.8z" />
      <path d="M9.5 9.5h5M9.5 13h5" />
    </>
  ),
  email: (
    <>
      <rect x="3.5" y="5.5" width="17" height="13" rx="2" />
      <path d="M4 7l8 6 8-6" />
    </>
  ),
  plan: (
    <>
      <path d="M9 6.5h11M9 12h11M9 17.5h11" />
      <path d="M3.8 6.5l1.2 1.2 2-2.2M3.8 12l1.2 1.2 2-2.2" />
      <circle cx="5.2" cy="17.5" r="1.3" />
    </>
  ),
  report: (
    <>
      <path d="M4 20V4M4 20h16" />
      <path d="M8 16v-4M12 16V8M16 16v-6" />
    </>
  ),
  analysis: (
    <>
      <path d="M4 20V4M4 20h9" />
      <path d="M8 16v-3M12 16V9" />
      <circle cx="16.5" cy="13.5" r="3" />
      <path d="M18.7 15.7l2.3 2.3" />
    </>
  ),
  portal: (
    <>
      <rect x="3.5" y="4.5" width="17" height="15" rx="2" />
      <path d="M3.5 8.5h17" />
      <circle cx="6.2" cy="6.5" r="0.6" />
      <path d="M11 11.5l5.5 2.2-2.4 0.9-0.9 2.4z" />
    </>
  ),
  diff: (
    <>
      <path d="M7 4v10M3.5 9.5L7 14l3.5-4.5" />
      <path d="M17 20V10M13.5 14.5L17 10l3.5 4.5" />
    </>
  ),
  doc: (
    <>
      <path d="M6 3.5h8l4 4v13H6z" />
      <path d="M14 3.5v4h4M9 12h6M9 15.5h6" />
    </>
  ),
  view: <path d="M4 5h16l-6 7.5V19l-4-2v-4.5z" />,
  decision: (
    <>
      <path d="M12 3.5l7 2.5v5.5c0 4.4-3 7.7-7 9-4-1.3-7-4.6-7-9V6z" />
      <path d="M9 12l2 2 4-4" />
    </>
  ),
  extract: (
    <>
      <path d="M4 8V5.5A1.5 1.5 0 0 1 5.5 4H8M16 4h2.5A1.5 1.5 0 0 1 20 5.5V8M20 16v2.5a1.5 1.5 0 0 1-1.5 1.5H16M8 20H5.5A1.5 1.5 0 0 1 4 18.5V16" />
      <path d="M8 10h8M8 14h5" />
    </>
  ),
  alert: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M12 8v4.5M12 15.8v.01" />
    </>
  ),
  mail: (
    <>
      <rect x="3.5" y="5.5" width="17" height="13" rx="2" />
      <path d="M4 7l8 6 8-6" />
    </>
  ),
  pin: <path d="M9 4h6l-1 5 3 3H7l3-3zM12 12v8" />,
  dl: <path d="M12 4v11M7.5 10.5L12 15l4.5-4.5M5 19.5h14" />,
  ext: (
    <path d="M14 4.5h5.5V10M19.5 4.5L11 13M17 14v4.5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h4.5" />
  ),
  x: <path d="M6.5 6.5l11 11M17.5 6.5l-11 11" />,
  pause: <path d="M9 6v12M15 6v12" />,
  up: <path d="M6 15l6-6 6 6" />,
  down: <path d="M6 9l6 6 6-6" />,
  search: (
    <>
      <circle cx="11" cy="11" r="6" />
      <path d="M20 20l-4.5-4.5" />
    </>
  ),
  check: <path d="M5 12.5l4.5 4.5L19 7" />,
  truck: (
    <>
      <path d="M3.5 6.5h10v9h-10zM13.5 9.5h4l3 3v3h-7" />
      <circle cx="7" cy="17" r="1.6" />
      <circle cx="17" cy="17" r="1.6" />
    </>
  ),
  warn: (
    <>
      <path d="M12 4l9 15.5H3z" />
      <path d="M12 10v4M12 17v.01" />
    </>
  ),
  copy: (
    <>
      <rect x="8" y="8" width="11" height="11" rx="2" />
      <path d="M5 15V6a1 1 0 0 1 1-1h9" />
    </>
  ),
} satisfies Record<string, ReactNode>;

export type ArtIconName = keyof typeof ICONS;

/** The workspace's own line icons, drawn on one 24-unit grid. */
export function ArtIcon({
  name,
  size = 14,
  stroke = 1.8,
}: {
  name: ArtIconName;
  size?: number;
  stroke?: number;
}) {
  return (
    <svg
      className="size-auto"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={stroke}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      {ICONS[name]}
    </svg>
  );
}

/** How the workspace groups the kinds it draws: one mark and one colour per family. */
export type DeskArtKind =
  | "table"
  | "record"
  | "rate"
  | "email"
  | "plan"
  | "report"
  | "analysis"
  | "portal"
  | "diff"
  | "doc"
  | "view"
  | "decision"
  | "extract"
  | "bill"
  | "formula"
  | "shipment"
  | "customer"
  | "tractor"
  | "trailer"
  | "worker"
  | "location"
  | "carrier"
  | "order"
  | "failure"
  | "agreement"
  | "edi"
  | "inbound"
  | "tender"
  | "partner"
  | "invoice"
  | "pay"
  | "payable"
  | "detention"
  | "journal"
  | "manual";

const KIND_LABELS: Record<DeskArtKind, string> = defineLabels({
  table: "Table",
  record: "Record",
  rate: "Rate explanation",
  email: "Email draft",
  plan: "Plan",
  report: "Report",
  analysis: "Analysis",
  portal: "Portal task",
  diff: "Run diff",
  doc: "Document",
  view: "View",
  decision: "Decision",
  extract: "Extraction",
  bill: "Billing item",
  formula: "Formula template",
  shipment: "Shipment",
  customer: "Customer",
  tractor: "Tractor",
  trailer: "Trailer",
  worker: "Worker",
  location: "Location",
  carrier: "Carrier",
  order: "Order",
  failure: "Service failure",
  agreement: "Rate agreement",
  edi: "EDI message",
  inbound: "EDI inbound file",
  tender: "Load tender",
  partner: "EDI partner",
  invoice: "Invoice",
  pay: "Worker settlement",
  payable: "Carrier settlement",
  detention: "Detention charge",
  journal: "Journal entry",
  manual: "Manual journal",
});

/**
 * Records the Desk draws as a card of their own, each with its own mark: a
 * billing queue item as the item a biller reviews, a formula template as the
 * formula a biller reads, a shipment as the load a dispatcher follows, a
 * customer as the account an account manager or a biller keeps, a tractor or a
 * trailer as the unit a dispatcher or fleet manager runs, a worker as the
 * driver a dispatcher or safety manager puts on loads, a location as the dock
 * a planner sends trucks to, a carrier as the partner a carrier manager
 * qualifies and pays, an order as the business a customer service rep and a
 * biller follow from taken to billed, a service failure as the miss a customer
 * service rep explains and settles, a rate agreement as the terms a pricing
 * analyst holds freight to, an EDI message as the document an EDI coordinator
 * follows to and from a trading partner, an EDI inbound file as the delivery
 * a trading partner sent that the coordinator traces into messages, an EDI
 * transfer as the load tender a coordinator accepts or declines, an EDI partner
 * as the trading partner a coordinator keeps trading, an invoice as
 * the bill a biller sends and a collector follows to paid, a worker settlement
 * as the pay stub a payroll clerk approves and pays, a carrier settlement as
 * the payable an AP clerk checks against the carrier's invoices and pays, a
 * detention occurrence as the charge a biller approves for a stop that ran
 * past its free time, a journal entry as the posting an accountant reviews in
 * the general ledger, a manual journal as the entry a controller approves
 * before it is booked by hand.
 */
const CARD_KINDS: Readonly<Record<string, DeskArtKind>> = {
  billing_queue_item: "bill",
  formula_template: "formula",
  shipment: "shipment",
  customer: "customer",
  tractor: "tractor",
  trailer: "trailer",
  worker: "worker",
  location: "location",
  carrier: "carrier",
  order: "order",
  service_failure: "failure",
  rate_agreement: "agreement",
  edi_message: "edi",
  edi_inbound_file: "inbound",
  edi_transfer: "tender",
  edi_partner: "partner",
  invoice: "invoice",
  driver_settlement: "pay",
  carrier_settlement: "payable",
  detention_occurrence: "detention",
  journal_entry: "journal",
  manual_journal: "manual",
};

export function deskArtKind(artifact: Pick<AssistantArtifact, "kind" | "payload">): DeskArtKind {
  switch (artifact.kind) {
    case "table_view":
      return "path" in artifact.payload ? "view" : "table";
    case "report_preview":
    case "report_run":
      return "report";
    case "analysis":
      return "analysis";
    case "portal_task":
      return "portal";
    case "entity_card": {
      const entity = artifact.payload.entity;
      return typeof entity === "string" && Object.hasOwn(CARD_KINDS, entity)
        ? CARD_KINDS[entity]
        : "record";
    }
    case "rate_explanation":
      return "rate";
    case "email_draft":
    case "inbound_message":
      return "email";
    case "plan":
      return "plan";
    case "run_diff":
      return "diff";
    case "navigation":
    case "dashboard_ref":
      return "view";
    case "decision_request":
      return "decision";
    case "extraction":
    case "edi_connection_draft":
      return "extract";
    case "document":
    case "briefing":
    case "draft_edit":
      return "doc";
    default:
      return "doc";
  }
}

export function deskArtKindLabel(kind: DeskArtKind): string {
  return KIND_LABELS[kind];
}

/** The kind's name in the reader's language. */
export function deskArtKindName(kind: DeskArtKind, t: TranslateFn): string {
  switch (kind) {
    case "table":
      return t("Table");
    case "record":
      return t("Record");
    case "rate":
      return t("Rate explanation");
    case "email":
      return t("Email draft");
    case "plan":
      return t("Plan");
    case "report":
      return t("Report");
    case "analysis":
      return t("Analysis");
    case "portal":
      return t("Portal task");
    case "diff":
      return t("Run diff");
    case "doc":
      return t("Document");
    case "view":
      return t("View");
    case "decision":
      return t("Decision");
    case "extract":
      return t("Extraction");
    case "bill":
      return t("Billing item");
    case "formula":
      return t("Formula template");
    case "shipment":
      return t("Shipment");
    case "customer":
      return t("Customer");
    case "tractor":
      return t("Tractor");
    case "trailer":
      return t("Trailer");
    case "worker":
      return t("Worker");
    case "location":
      return t("Location");
    case "carrier":
      return t("Carrier");
    case "order":
      return t("Order");
    case "failure":
      return t("Service failure");
    case "agreement":
      return t("Rate agreement");
    case "edi":
      return t("EDI message");
    case "inbound":
      return t("EDI inbound file");
    case "tender":
      return t("Load tender");
    case "partner":
      return t("EDI partner");
    case "invoice":
      return t("Invoice");
    case "pay":
      return t("Worker settlement");
    case "payable":
      return t("Carrier settlement");
    case "detention":
      return t("Detention charge");
    case "journal":
      return t("Journal entry");
    case "manual":
      return t("Manual journal");
  }
}

/** The kinds in the order the browser's filters list them. */
export const DESK_ART_KINDS = Object.keys(KIND_LABELS) as DeskArtKind[];

export function DeskArtKindIcon({ kind, size = 15 }: { kind: DeskArtKind; size?: number }) {
  return <ArtIcon name={kind} size={size} stroke={size <= 11 ? 2 : 1.8} />;
}
