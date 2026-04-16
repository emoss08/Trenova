import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { formatUnixDate, formatUnixDateTime } from "@/lib/date";
import { getDestinationLocation, getOriginLocation } from "@/lib/shipment-utils";
import { cn, formatCurrency } from "@/lib/utils";
import type { Invoice } from "@/types/invoice";
import type { InvoiceAdjustment, InvoiceAdjustmentLineage } from "@/types/invoice-adjustment";
import { ExternalLinkIcon } from "lucide-react";
import { Link } from "react-router";
import { InvoiceAdjustmentRuntimeSection } from "./invoice-adjustment-runtime-section";

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
  const shipment = invoice.shipment;
  const originLocation = shipment ? getOriginLocation(shipment) : null;
  const destinationLocation = shipment ? getDestinationLocation(shipment) : null;

  return (
    <ScrollArea className="h-full">
      <div className="flex flex-col gap-5 px-4 py-2">
        <InvoiceAdjustmentRuntimeSection
          invoice={invoice}
          correctionSummary={correctionSummary}
          latestAdjustment={latestAdjustment}
          latestAdjustmentDetail={latestAdjustmentDetail}
        />
        <div className="grid gap-5 xl:grid-cols-2">
          <div className="flex flex-col gap-5">
            <div className="rounded-lg border bg-card p-3">
              <SectionLabel>Bill-To</SectionLabel>
              <div className="mt-1.5">
                <p className="text-sm font-medium">{invoice.billToName}</p>
                {invoice.billToCode ? (
                  <p className="mt-0.5 text-2xs text-muted-foreground">{invoice.billToCode}</p>
                ) : null}
                <div className="mt-1 text-xs text-muted-foreground">
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

            <div className="rounded-lg border bg-card p-3">
              <SectionLabel>Charge Summary</SectionLabel>
              <div className="mt-2 space-y-2">
                <ChargeSummaryRow
                  label="Freight Charges"
                  value={formatCurrency(Number(invoice.subtotalAmount ?? 0), invoice.currencyCode)}
                />
                <ChargeSummaryRow
                  label="Other Charges"
                  value={formatCurrency(Number(invoice.otherAmount ?? 0), invoice.currencyCode)}
                />
                <Separator />
                <ChargeSummaryRow
                  label="Total"
                  value={formatCurrency(Number(invoice.totalAmount ?? 0), invoice.currencyCode)}
                  bold
                />
              </div>
            </div>

            <div className="rounded-lg border bg-card p-3">
              <SectionLabel>References</SectionLabel>
              <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
                <PropertyCell label="Shipment">
                  <Link
                    to={`/shipment-management/shipments?item=${invoice.shipmentId}`}
                    className="inline-flex items-center gap-1 text-xs font-medium hover:underline"
                  >
                    {invoice.shipmentProNumber || invoice.shipmentId.slice(0, 12)}
                    <ExternalLinkIcon className="size-2.5" />
                  </Link>
                </PropertyCell>
                <PropertyCell label="Billing Queue">
                  <Link
                    to={`/billing/queue?item=${invoice.billingQueueItemId}&includePosted=true`}
                    className="inline-flex items-center gap-1 text-xs font-medium hover:underline"
                  >
                    Queue Item
                    <ExternalLinkIcon className="size-2.5" />
                  </Link>
                </PropertyCell>
                {invoice.shipmentBol ? (
                  <PropertyCell label="BOL">
                    <span className="text-xs font-medium">{invoice.shipmentBol}</span>
                  </PropertyCell>
                ) : null}
                {originLocation && destinationLocation ? (
                  <PropertyCell label="Route">
                    <span className="text-xs font-medium">
                      {originLocation.city}, {originLocation.state?.abbreviation} →{" "}
                      {destinationLocation.city}, {destinationLocation.state?.abbreviation}
                    </span>
                  </PropertyCell>
                ) : null}
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-5">
            <div className="rounded-lg border bg-card p-3">
              <SectionLabel>Invoice Details</SectionLabel>
              <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-2">
                <PropertyCell label="Service Date">
                  <span className="text-xs font-medium">{formatUnixDate(invoice.serviceDate)}</span>
                </PropertyCell>
                <PropertyCell label="Currency">
                  <span className="text-xs font-medium">{invoice.currencyCode}</span>
                </PropertyCell>
                <PropertyCell label="Posted">
                  <span className="text-xs font-medium">
                    {invoice.status === "Posted" ? formatUnixDateTime(invoice.postedAt) : "Not yet"}
                  </span>
                </PropertyCell>
                <PropertyCell label="Lineage">
                  <span className="text-xs font-medium">
                    {invoice.isAdjustmentArtifact
                      ? isCurrentVersion
                        ? "Current artifact"
                        : "Historical artifact"
                      : "Root invoice"}
                  </span>
                </PropertyCell>
              </div>
            </div>

            <div className="rounded-lg border bg-card p-3">
              <SectionLabel>Lifecycle</SectionLabel>
              <div className="mt-2">
                <LifecycleStep
                  label="Generated from Billing Queue"
                  active
                  timestamp={formatUnixDateTime(invoice.createdAt)}
                />
                <LifecycleStep
                  label="Ready for Posting"
                  active
                  timestamp={formatUnixDate(invoice.invoiceDate)}
                />
                <LifecycleStep
                  label="Posted to Invoice History"
                  active={invoice.status === "Posted"}
                  timestamp={formatUnixDateTime(invoice.postedAt)}
                  isLast
                />
              </div>
            </div>

            {correctionSummary?.invoices.length ? (
              <div className="rounded-lg border bg-card p-3">
                <SectionLabel>Correction Group</SectionLabel>
                <div className="mt-2 flex flex-col gap-1.5">
                  {correctionSummary.invoices.map((lineageInvoice) => {
                    const current =
                      correctionSummary.correctionGroup.currentInvoiceId === lineageInvoice.id;
                    return (
                      <div
                        key={lineageInvoice.id}
                        className="flex items-center justify-between gap-3 rounded-md border bg-background px-3 py-2"
                      >
                        <div className="min-w-0">
                          <p className="truncate text-xs font-medium">{lineageInvoice.number}</p>
                          <p className="text-2xs text-muted-foreground">
                            {lineageInvoice.billType} · {lineageInvoice.status}
                          </p>
                        </div>
                        <Badge variant={current ? "active" : "secondary"} className="shrink-0">
                          {current ? "Current" : "Superseded"}
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
  return <p className="text-xs font-medium text-muted-foreground">{children}</p>;
}

function PropertyCell({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-2xs text-muted-foreground">{label}</p>
      {children}
    </div>
  );
}

function ChargeSummaryRow({
  label,
  value,
  bold = false,
}: {
  label: string;
  value: string;
  bold?: boolean;
}) {
  return (
    <div className="flex items-center justify-between">
      <span
        className={cn("text-sm", bold ? "font-medium text-foreground" : "text-muted-foreground")}
      >
        {label}
      </span>
      <span
        className={cn(
          "tracking-tight tabular-nums",
          bold ? "text-base font-semibold text-foreground" : "text-sm text-muted-foreground",
        )}
      >
        {value}
      </span>
    </div>
  );
}

function LifecycleStep({
  label,
  active,
  timestamp,
  isLast = false,
}: {
  label: string;
  active: boolean;
  timestamp: string;
  isLast?: boolean;
}) {
  return (
    <div className="relative flex gap-3">
      <div className="flex flex-col items-center">
        <div
          className={cn(
            "mt-1 size-2 rounded-full",
            active ? "bg-green-600" : "bg-muted-foreground/30",
          )}
        />
        {!isLast ? (
          <div className={cn("my-0.5 w-px flex-1", active ? "bg-green-600/30" : "bg-border")} />
        ) : null}
      </div>
      <div className={cn("pb-3", isLast && "pb-0")}>
        <p className={cn("text-xs font-medium", !active && "text-muted-foreground")}>{label}</p>
        <p className="text-2xs text-muted-foreground">{timestamp}</p>
      </div>
    </div>
  );
}
