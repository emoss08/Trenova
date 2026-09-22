import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptyTable, type EmptyTableColumn } from "@trenova/shared/components/ui/empty-table";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { invoiceBillsSingleShipment } from "@/lib/invoice-scope";
import { shipmentPanelPath } from "@/lib/shipment-utils";
import {
  ORDER_CHARGES_GROUP_KEY,
  chargeComposition,
  describeAllocationShare,
  describeChargeCalculation,
  groupInvoiceLinesByShipment,
  type InvoiceLineGroup,
} from "@trenova/shared/lib/invoice-lines";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import type { Invoice, InvoiceLine, InvoiceLineType } from "@trenova/shared/types/invoice";
import {
  ChevronRightIcon,
  ChevronsDownUpIcon,
  ChevronsUpDownIcon,
  ExternalLinkIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";

/**
 * Beyond this many shipments the list is long enough that opening every section
 * buries the one a biller is looking for, so all but the first start collapsed.
 */
const COLLAPSE_THRESHOLD = 8;

const LINE_TYPE_DOTS: Record<InvoiceLineType, string> = {
  Freight: "bg-info",
  Accessorial: "bg-accent-violet",
  Memo: "bg-accent-amber",
};

const EMPTY_COLUMNS: readonly EmptyTableColumn[] = [
  { label: "Charge" },
  { label: "Calculation" },
  { label: "Amount", numeric: true },
];

function sectionHeading(group: InvoiceLineGroup, t: TranslateFn): string {
  if (group.key === ORDER_CHARGES_GROUP_KEY) return t("Order charges");
  return group.proNumber ?? group.key;
}

function initialCollapsedKeys(
  groups: readonly InvoiceLineGroup[],
  isSummary: boolean,
): ReadonlySet<string> {
  if (groups.length <= 1) return new Set<string>();
  // The customer's copy lists one line per shipment, so the tab opens on that
  // same view; the breakdown is a click away.
  if (isSummary) return new Set(groups.map((group) => group.key));
  if (groups.length > COLLAPSE_THRESHOLD) {
    return new Set(groups.slice(1).map((group) => group.key));
  }
  return new Set<string>();
}

export function InvoiceChargesTab({ invoice }: { invoice: Invoice }) {
  const t = useT();

  const lines = invoice.lines ?? [];
  const groups = useMemo(() => groupInvoiceLinesByShipment(invoice.lines ?? []), [invoice.lines]);

  const isGrouped = groups.length > 1;
  const isSummary = invoice.detail === "Summary" && isGrouped;

  const [collapse, setCollapse] = useState(() => ({
    invoiceId: invoice.id,
    keys: initialCollapsedKeys(groups, isSummary),
  }));

  // The pane swaps invoices without remounting the tab; each invoice opens on
  // its own default rather than inheriting the previous invoice's sections.
  if (collapse.invoiceId !== invoice.id) {
    setCollapse({ invoiceId: invoice.id, keys: initialCollapsedKeys(groups, isSummary) });
  }

  if (lines.length === 0) {
    return <InvoiceChargesEmpty invoice={invoice} />;
  }

  const collapsed = collapse.keys;
  const anyCollapsed = groups.some((group) => collapsed.has(group.key));
  const shipmentCount = groups.filter((group) => group.shipmentId !== null).length;

  function toggleSection(key: string) {
    setCollapse((previous) => {
      const next = new Set(previous.keys);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return { ...previous, keys: next };
    });
  }

  function toggleAll() {
    setCollapse((previous) => ({
      ...previous,
      keys: anyCollapsed ? new Set<string>() : new Set(groups.map((group) => group.key)),
    }));
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-1 border-b px-4 py-2">
        <p className="text-sm font-medium">
          {isGrouped
            ? t(
                "{0, plural, one {# charge} other {# charges}} across {1, plural, one {# shipment} other {# shipments}}",
                lines.length,
                shipmentCount,
              )
            : t("{0, plural, one {# charge} other {# charges}}", lines.length)}
        </p>
        {isGrouped ? (
          <p className="text-muted-foreground text-xs">
            {isSummary
              ? t("Customer copy lists one line per shipment")
              : t("Customer copy lists every charge")}
          </p>
        ) : null}
        {isGrouped ? (
          <Button size="xs" variant="ghost" className="ml-auto" onClick={toggleAll}>
            {anyCollapsed ? (
              <ChevronsUpDownIcon className="size-3.5" />
            ) : (
              <ChevronsDownUpIcon className="size-3.5" />
            )}
            {anyCollapsed ? t("Expand all") : t("Collapse all")}
          </Button>
        ) : null}
      </div>

      <ScrollArea className="min-h-0 flex-1" maskHeight={0}>
        <div className="flex flex-col gap-3 p-4">
          {isGrouped ? (
            groups.map((group) => (
              <ChargeSection
                key={group.key}
                group={group}
                currencyCode={invoice.currencyCode}
                isCollapsed={collapsed.has(group.key)}
                onToggle={() => toggleSection(group.key)}
              />
            ))
          ) : (
            <ChargeBreakdown group={groups[0]} currencyCode={invoice.currencyCode} />
          )}
        </div>
      </ScrollArea>

      <InvoiceChargeTotals invoice={invoice} />
    </div>
  );
}

function ChargeSection({
  group,
  currencyCode,
  isCollapsed,
  onToggle,
}: {
  group: InvoiceLineGroup;
  currencyCode: string;
  isCollapsed: boolean;
  onToggle: () => void;
}) {
  const t = useT();

  const heading = sectionHeading(group, t);
  const bodyId = `invoice-charge-section-${group.key}`;

  return (
    <section className="overflow-hidden rounded-lg border" aria-label={heading}>
      <div
        data-section-heading
        className={cn(
          "bg-muted/40 flex min-w-0 items-center gap-1 py-1 pr-3 pl-1",
          !isCollapsed && "border-b",
        )}
      >
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={!isCollapsed}
          aria-controls={bodyId}
          className="ui-focus-ring hover:text-foreground text-muted-foreground flex min-w-0 items-center gap-2 rounded-sm px-2 py-1 text-left transition-colors"
        >
          <ChevronRightIcon
            className={cn(
              "size-3.5 shrink-0 transition-transform motion-reduce:transition-none",
              !isCollapsed && "rotate-90",
            )}
          />
          <span
            className={cn(
              "text-foreground truncate text-xs font-medium",
              group.shipmentId !== null && "font-mono",
            )}
          >
            {heading}
          </span>
          {group.bol ? <span className="truncate text-xs">{t("BOL {0}", group.bol)}</span> : null}
        </button>
        {group.shipmentId !== null ? (
          <Tooltip>
            <TooltipTrigger
              render={
                <Link
                  to={shipmentPanelPath(group.shipmentId)}
                  aria-label={t("Open shipment {0}", heading)}
                  className={buttonVariants({ variant: "ghost", size: "icon-xs" })}
                />
              }
            >
              <ExternalLinkIcon className="size-3" />
            </TooltipTrigger>
            <TooltipContent side="top" sideOffset={8}>
              {t("Open shipment")}
            </TooltipContent>
          </Tooltip>
        ) : null}
        <span className="text-muted-foreground ml-auto shrink-0 pl-3 text-xs tabular-nums">
          {t("{0, plural, one {# charge} other {# charges}}", group.lines.length)}
        </span>
        <span className="shrink-0 pl-3 text-sm font-semibold tabular-nums">
          {formatCurrency(group.subtotal, currencyCode)}
        </span>
      </div>

      <div id={bodyId}>
        {isCollapsed ? null : <ChargeBreakdown group={group} currencyCode={currencyCode} />}
      </div>
    </section>
  );
}

/**
 * One shipment's charges laid out the way the billing queue shows them before
 * they are invoiced: the rating, the line haul, then each accessorial by name
 * with how its amount was reached.
 */
function ChargeBreakdown({
  group,
  currencyCode,
}: {
  group: InvoiceLineGroup;
  currencyCode: string;
}) {
  const t = useT();

  if (group.key === ORDER_CHARGES_GROUP_KEY) {
    return (
      <ul aria-label={t("Order charges")} className="flex flex-col p-2">
        {group.lines.map((line) => (
          <ChargeRow key={line.id} line={line} currencyCode={currencyCode} />
        ))}
      </ul>
    );
  }

  const freightLines = group.lines.filter((line) => line.type === "Freight");
  const accessorialLines = group.lines.filter((line) => line.type === "Accessorial");
  const formulaName = freightLines.find((line) => line.formulaTemplateName)?.formulaTemplateName;
  const accessorialSubtotal = accessorialLines.reduce((sum, line) => sum + (line.amount ?? 0), 0);

  return (
    <div className="flex flex-col gap-1 p-2">
      {formulaName ? (
        <div className="border-border bg-muted mx-2 mb-1 flex items-center gap-2 rounded-md border px-3 py-2 text-xs">
          <span className="text-muted-foreground">{t("Rating:")}</span>
          <span className="font-medium">{formulaName}</span>
        </div>
      ) : null}

      <ul aria-label={t("Line haul")} className="flex flex-col">
        {freightLines.map((line) => (
          <FreightRows key={line.id} line={line} currencyCode={currencyCode} />
        ))}
      </ul>

      <Separator className="my-1" />

      <div className="px-2 pt-1">
        <span className="text-muted-foreground text-xs font-medium">{t("Accessorials")}</span>
      </div>

      {accessorialLines.length === 0 ? (
        <p className="text-muted-foreground px-2 pb-2 text-xs">{t("No accessorial charges")}</p>
      ) : (
        <>
          <ul aria-label={t("Accessorials")} className="flex flex-col">
            {accessorialLines.map((line) => (
              <ChargeRow key={line.id} line={line} currencyCode={currencyCode} />
            ))}
          </ul>
          <div className="text-muted-foreground flex items-center justify-between px-2 py-1">
            <span className="text-xs">{t("Subtotal")}</span>
            <span className="text-xs font-medium tabular-nums">
              {formatCurrency(accessorialSubtotal, currencyCode)}
            </span>
          </div>
        </>
      )}
    </div>
  );
}

function FreightRows({ line, currencyCode }: { line: InvoiceLine; currencyCode: string }) {
  const t = useT();

  return (
    <>
      {line.rate != null ? (
        <ChargeRowLayout
          label={t("Base rate")}
          details={[t("Per-unit rate before formula")]}
          value={formatCurrency(line.rate, currencyCode)}
          muted
        />
      ) : null}
      <ChargeRowLayout
        label={t("Line haul")}
        details={[t("Line {0}", line.lineNumber)]}
        value={formatCurrency(line.amount ?? 0, currencyCode)}
      />
    </>
  );
}

function ChargeRow({ line, currencyCode }: { line: InvoiceLine; currencyCode: string }) {
  const t = useT();

  const calculation = describeChargeCalculation(line, currencyCode, t);
  const share = describeAllocationShare(line, t);
  const details = [calculation, share, t("Line {0}", line.lineNumber)].filter(
    (detail): detail is string => Boolean(detail),
  );

  return (
    <ChargeRowLayout
      label={t(line.description)}
      code={line.chargeCode}
      details={details}
      value={formatCurrency(line.amount ?? 0, currencyCode)}
    />
  );
}

function ChargeRowLayout({
  label,
  code,
  details,
  value,
  muted = false,
}: {
  label: string;
  code?: string | null;
  details: string[];
  value: string;
  muted?: boolean;
}) {
  return (
    <li className="hover:bg-muted flex items-center justify-between gap-3 rounded-md p-2 transition-colors">
      <div className="flex min-w-0 flex-col">
        <span className="flex min-w-0 items-center gap-1.5">
          <span className="truncate text-sm">{label}</span>
          {code ? (
            <span className="bg-muted text-muted-foreground shrink-0 rounded-md border px-1 font-mono text-2xs leading-4">
              {code}
            </span>
          ) : null}
        </span>
        <span className="text-muted-foreground truncate text-xs">{details.join(" · ")}</span>
      </div>
      <span
        className={cn(
          "shrink-0 text-sm tabular-nums",
          muted ? "text-muted-foreground" : "font-medium",
        )}
      >
        {value}
      </span>
    </li>
  );
}

/**
 * The invoice's own totals, read from the header the PDF prints: freight is the
 * sum of Freight lines and accessorials the sum of Accessorial lines.
 */
function InvoiceChargeTotals({ invoice }: { invoice: Invoice }) {
  const t = useT();

  const freight = Number(invoice.subtotalAmount ?? 0);
  const accessorial = Number(invoice.otherAmount ?? 0);
  const composition = chargeComposition(freight, accessorial);

  return (
    <section
      aria-label={t("Invoice totals")}
      className="bg-muted/30 flex shrink-0 flex-wrap items-end gap-x-8 gap-y-3 border-t px-4 py-3"
    >
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        {composition ? (
          <div
            aria-hidden
            className="bg-muted flex h-1.5 w-full max-w-md gap-0.5 overflow-hidden rounded-full"
          >
            {composition.freightShare > 0 ? (
              <span
                className={LINE_TYPE_DOTS.Freight}
                style={{ width: `${composition.freightShare * 100}%` }}
              />
            ) : null}
            {composition.accessorialShare > 0 ? (
              <span
                className={LINE_TYPE_DOTS.Accessorial}
                style={{ width: `${composition.accessorialShare * 100}%` }}
              />
            ) : null}
          </div>
        ) : null}
        <dl className="flex flex-wrap gap-x-6 gap-y-1 text-xs">
          <TotalsLegendItem
            label={t("Freight")}
            dotClassName={LINE_TYPE_DOTS.Freight}
            value={formatCurrency(freight, invoice.currencyCode)}
          />
          <TotalsLegendItem
            label={t("Accessorials")}
            dotClassName={LINE_TYPE_DOTS.Accessorial}
            value={formatCurrency(accessorial, invoice.currencyCode)}
          />
        </dl>
      </div>
      <dl>
        <div className="text-right">
          <dt className="text-muted-foreground text-xs">{t("Total")}</dt>
          <dd className="text-lg font-semibold tabular-nums">
            {formatCurrency(Number(invoice.totalAmount ?? 0), invoice.currencyCode)}
          </dd>
        </div>
      </dl>
    </section>
  );
}

function TotalsLegendItem({
  label,
  dotClassName,
  value,
}: {
  label: string;
  dotClassName: string;
  value: string;
}) {
  return (
    <div className="flex items-center gap-1.5">
      <span aria-hidden className={cn("size-1.5 rounded-full", dotClassName)} />
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}

function InvoiceChargesEmpty({ invoice }: { invoice: Invoice }) {
  const t = useT();

  // An order or consolidated invoice's queue item is only its anchor leg, so
  // linking to it would send the biller to one shipment of many.
  const queueItemAction = invoiceBillsSingleShipment(invoice.scope) ? (
    <Link
      to={`/billing/queue?item=${invoice.billingQueueItemId}&includePosted=true`}
      className={buttonVariants({ variant: "outline", size: "sm" })}
    >
      {t("Open queue item")}
    </Link>
  ) : undefined;

  return (
    <div className="flex h-full items-center justify-center p-4">
      <EmptyTable
        title={t("No charges on this invoice")}
        description={t(
          "Charges are copied from each shipment's rating when the invoice is generated. Review the rating in the billing queue.",
        )}
        columns={EMPTY_COLUMNS}
        action={queueItemAction}
      />
    </div>
  );
}
