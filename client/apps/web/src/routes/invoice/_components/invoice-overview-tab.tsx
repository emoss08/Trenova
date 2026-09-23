import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatUnixDate, formatUnixDateTime } from "@trenova/shared/lib/date";
import { invoicePanelPath } from "@/lib/invoice-links";
import { invoiceBillingPeriod, invoiceBillsSingleShipment } from "@/lib/invoice-scope";
import { getDestinationLocation, getOriginLocation, shipmentPanelPath } from "@/lib/shipment-utils";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { Invoice } from "@trenova/shared/types/invoice";
import type { InvoiceAdjustment, InvoiceAdjustmentLineage } from "@/types/invoice-adjustment";
import { ExternalLinkIcon, TriangleAlertIcon } from "lucide-react";
import { Link } from "react-router";
import { InvoiceAdjustmentRuntimeSection } from "./invoice-adjustment-runtime-section";
import { InvoiceArContextSection } from "./invoice-ar-context-section";
import { recordPath } from "@/config/record-links";

export function InvoiceOverviewTab({
  invoice,
  isCurrentVersion,
  correctionSummary,
  latestAdjustment,
  latestAdjustmentDetail,
}: {
  invoice: Invoice;
  isCurrentVersion: boolean;
  correctionSummary: InvoiceAdjustmentLineage | undefined;
  latestAdjustment: InvoiceAdjustment | null;
  latestAdjustmentDetail: InvoiceAdjustment | null | undefined;
}) {
  const t = useT();

  const shipment = invoice.shipment;
  const originLocation = shipment ? getOriginLocation(shipment) : null;
  const destinationLocation = shipment ? getDestinationLocation(shipment) : null;
  const billsSingleShipment = invoiceBillsSingleShipment(invoice.scope);
  const billingPeriod = invoiceBillingPeriod(invoice);

  return (
    <ScrollArea className="h-full">
      <div className="flex flex-col gap-5 px-4 py-2">
        <InvoiceAdjustmentRuntimeSection
          invoice={invoice}
          correctionSummary={correctionSummary}
          latestAdjustment={latestAdjustment}
          latestAdjustmentDetail={latestAdjustmentDetail}
        />
        {invoice.offCycleReason ? (
          <Alert variant="warning" size="sm">
            <TriangleAlertIcon />
            <AlertTitle>{t("Billed outside this customer's statement")}</AlertTitle>
            <AlertDescription>{invoice.offCycleReason}</AlertDescription>
          </Alert>
        ) : null}
        <div className="grid gap-5 xl:grid-cols-2">
          <div className="flex flex-col gap-5">
            <div className="bg-card rounded-lg border p-3">
              <SectionLabel>{t("Bill-To")}</SectionLabel>
              <div className="mt-1.5">
                <p className="text-sm font-medium">{invoice.billToName}</p>
                {invoice.billToCode ? (
                  <p className="text-2xs text-muted-foreground mt-0.5">{invoice.billToCode}</p>
                ) : null}
                <div className="text-muted-foreground mt-1 text-xs">
                  {invoice.billToAddressLine1 ? <p>{invoice.billToAddressLine1}</p> : null}
                  {invoice.billToAddressLine2 ? <p>{invoice.billToAddressLine2}</p> : null}
                  <p>
                    {[invoice.billToCity, invoice.billToState, invoice.billToPostalCode]
                      .filter(Boolean)
                      .join(", ")}
                  </p>
                  {invoice.billToCountry ? <p>{invoice.billToCountry}</p> : null}
                </div>
              </div>
            </div>

            <InvoiceArContextSection invoice={invoice} />

            {invoice.scope === "Memo" ? <MemoDetailsCard invoice={invoice} /> : null}

            <div className="bg-card rounded-lg border p-3">
              <SectionLabel>{t("Charge summary")}</SectionLabel>
              <DescriptionList layout="split" className="mt-1">
                <DescriptionItem label={t("Freight charges")} numeric>
                  {formatCurrency(Number(invoice.subtotalAmount ?? 0), invoice.currencyCode)}
                </DescriptionItem>
                <DescriptionItem label={t("Other charges")} numeric>
                  {formatCurrency(Number(invoice.otherAmount ?? 0), invoice.currencyCode)}
                </DescriptionItem>
                <DescriptionItem
                  label={t("Total")}
                  numeric
                  valueClassName="text-base font-semibold"
                >
                  {formatCurrency(Number(invoice.totalAmount ?? 0), invoice.currencyCode)}
                </DescriptionItem>
              </DescriptionList>
            </div>

            <div className="bg-card rounded-lg border p-3">
              <SectionLabel>{t("References")}</SectionLabel>
              <DescriptionList className="mt-2">
                {billsSingleShipment && invoice.shipmentId ? (
                  <DescriptionItem label={t("Shipment")}>
                    <Link
                      to={shipmentPanelPath(invoice.shipmentId)}
                      className="inline-flex items-center gap-1 hover:underline"
                    >
                      {invoice.shipmentProNumber || invoice.shipmentId.slice(0, 12)}
                      <ExternalLinkIcon className="size-2.5" />
                    </Link>
                  </DescriptionItem>
                ) : null}
                {invoice.orderId ? (
                  <DescriptionItem label={t("Order")}>
                    <Link
                      to={recordPath("order", invoice.orderId)}
                      className="inline-flex items-center gap-1 hover:underline"
                    >
                      {invoice.orderNumber || invoice.orderId.slice(0, 12)}
                      <ExternalLinkIcon className="size-2.5" />
                    </Link>
                  </DescriptionItem>
                ) : null}
                {billsSingleShipment ? (
                  <DescriptionItem label={t("Billing queue")}>
                    <Link
                      to={`/billing/queue?item=${invoice.billingQueueItemId}&includePosted=true`}
                      className="inline-flex items-center gap-1 hover:underline"
                    >
                      {t("Queue item")}
                      <ExternalLinkIcon className="size-2.5" />
                    </Link>
                  </DescriptionItem>
                ) : (
                  <DescriptionItem label={t("Shipments")}>
                    {t("{0, plural, one {# shipment} other {# shipments}}", invoice.shipmentCount)}
                  </DescriptionItem>
                )}
                {billsSingleShipment && invoice.shipmentBol ? (
                  <DescriptionItem label={t("BOL")}>{invoice.shipmentBol}</DescriptionItem>
                ) : null}
                {originLocation && destinationLocation ? (
                  <DescriptionItem label={t("Route")}>
                    {originLocation.city}, {originLocation.state?.abbreviation} →{" "}
                    {destinationLocation.city}, {destinationLocation.state?.abbreviation}
                  </DescriptionItem>
                ) : null}
              </DescriptionList>
            </div>
          </div>

          <div className="flex flex-col gap-5">
            <div className="bg-card rounded-lg border p-3">
              <SectionLabel>{t("Invoice details")}</SectionLabel>
              <DescriptionList className="mt-2">
                {billingPeriod ? (
                  <DescriptionItem label={t("Billing period")}>{billingPeriod}</DescriptionItem>
                ) : (
                  <DescriptionItem label={t("Service date")}>
                    {formatUnixDate(invoice.serviceDate)}
                  </DescriptionItem>
                )}
                <DescriptionItem label={t("Currency")}>{invoice.currencyCode}</DescriptionItem>
                <DescriptionItem label={t("Posted")}>
                  {invoice.status === "Posted"
                    ? formatUnixDateTime(invoice.postedAt)
                    : t("Not yet")}
                </DescriptionItem>
                <DescriptionItem label={t("Lineage")}>
                  {invoice.isAdjustmentArtifact
                    ? isCurrentVersion
                      ? t("Current artifact")
                      : t("Historical artifact")
                    : t("Root invoice")}
                </DescriptionItem>
              </DescriptionList>
            </div>

            <div className="bg-card rounded-lg border p-3">
              <SectionLabel>{t("Lifecycle")}</SectionLabel>
              <div className="mt-2">
                <LifecycleStep
                  label={
                    invoice.scope === "Consolidated"
                      ? t("Generated from statement")
                      : invoice.scope === "Memo"
                        ? t("Generated as a memo")
                        : t("Generated from billing queue")
                  }
                  active
                  timestamp={formatUnixDateTime(invoice.createdAt)}
                />
                <LifecycleStep
                  label={t("Ready for posting")}
                  active
                  timestamp={formatUnixDate(invoice.invoiceDate)}
                />
                <LifecycleStep
                  label={t("Posted to invoice history")}
                  active={invoice.status === "Posted" || Boolean(invoice.postedAt)}
                  timestamp={formatUnixDateTime(invoice.postedAt)}
                  isLast={invoice.status !== "Voided"}
                />
                {invoice.status === "Voided" ? (
                  <LifecycleStep
                    label={t("Voided")}
                    active
                    tone="danger"
                    timestamp={formatUnixDateTime(invoice.voidedAt)}
                    details={[
                      invoice.voidReason ?? "",
                      invoice.voidDisposition === "Rebill"
                        ? t("Released for rebilling")
                        : t("Freight retired, not rebilled"),
                    ].filter(Boolean)}
                    isLast
                  />
                ) : null}
              </div>
            </div>

            {correctionSummary?.invoices.length ? (
              <div className="bg-card rounded-lg border p-3">
                <SectionLabel>{t("Correction group")}</SectionLabel>
                <div className="mt-2 flex flex-col gap-1.5">
                  {correctionSummary.invoices.map((lineageInvoice) => {
                    const current =
                      correctionSummary.correctionGroup.currentInvoiceId === lineageInvoice.id;
                    return (
                      <div
                        key={lineageInvoice.id}
                        className="bg-background flex items-center justify-between gap-3 rounded-md border px-3 py-2"
                      >
                        <div className="min-w-0">
                          <p className="truncate text-xs font-medium">{lineageInvoice.number}</p>
                          <p className="text-2xs text-muted-foreground">
                            {lineageInvoice.billType} · {lineageInvoice.status}
                          </p>
                        </div>
                        <Badge variant={current ? "success" : "neutral"} className="shrink-0">
                          {current ? t("Current") : t("Superseded")}
                        </Badge>
                      </div>
                    );
                  })}
                </div>
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </ScrollArea>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return <p className="text-muted-foreground text-xs font-medium">{children}</p>;
}

function LifecycleStep({
  label,
  active,
  timestamp,
  details,
  tone = "default",
  isLast = false,
}: {
  label: string;
  active: boolean;
  timestamp: string;
  details?: string[];
  tone?: "default" | "danger";
  isLast?: boolean;
}) {
  const dot = tone === "danger" ? "bg-danger" : "bg-success";
  const line = tone === "danger" ? "bg-danger-border" : "bg-success-border";
  return (
    <div className="relative flex gap-3">
      <div className="flex flex-col items-center">
        <div className={cn("mt-1 size-2 rounded-full", active ? dot : "bg-muted-foreground/30")} />
        {!isLast ? <div className={cn("my-0.5 w-px flex-1", active ? line : "bg-border")} /> : null}
      </div>
      <div className={cn("pb-3", isLast && "pb-0")}>
        <p
          className={cn(
            "text-xs font-medium",
            !active && "text-muted-foreground",
            tone === "danger" && active && "text-danger-foreground",
          )}
        >
          {label}
        </p>
        <p className="text-2xs text-muted-foreground">{timestamp}</p>
        {details?.map((detail) => (
          <p key={detail} className="text-2xs mt-0.5">
            {detail}
          </p>
        ))}
      </div>
    </div>
  );
}

/**
 * What a standalone memo is about. A memo has no shipment to explain it, so
 * the reason and the invoice it corrects are the whole story.
 */
function MemoDetailsCard({ invoice }: { invoice: Invoice }) {
  const t = useT();

  return (
    <div className="bg-card rounded-lg border p-3" data-testid="invoice-memo-details">
      <SectionLabel>{t("Memo")}</SectionLabel>
      <DescriptionList className="mt-2">
        <DescriptionItem label={t("Kind")}>
          {invoice.memoKind === "LateCharge" ? t("Late charge") : t("Manual")}
        </DescriptionItem>
        {invoice.referenceInvoiceId ? (
          <DescriptionItem label={t("Referenced invoice")}>
            <Link
              to={invoicePanelPath(invoice.referenceInvoiceId)}
              className="inline-flex items-center gap-1 hover:underline"
              aria-label={t("Referenced invoice")}
            >
              {t("Open invoice")}
              <ExternalLinkIcon className="size-2.5" />
            </Link>
          </DescriptionItem>
        ) : null}
        <DescriptionItem label={t("Reason")} span="full" valueClassName="whitespace-pre-line">
          {invoice.memoReason || <DescriptionEmpty />}
        </DescriptionItem>
      </DescriptionList>
    </div>
  );
}
