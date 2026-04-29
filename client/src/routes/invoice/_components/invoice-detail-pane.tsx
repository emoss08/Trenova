import AuditTab from "@/components/audit-tab";
import { EmptyState } from "@/components/empty-state";
import { PlainInvoiceStatusBadge, PlainSettlementStatusBadge } from "@/components/status-badge";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { usePostInvoice } from "@/hooks/use-post-invoice";
import { formatUnixDate } from "@/lib/date";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import type { InvoiceLineType } from "@/types/invoice";
import { useQuery } from "@tanstack/react-query";
import { CheckIcon, FileTextIcon, PackageCheckIcon, ReceiptTextIcon, SendIcon } from "lucide-react";
import { lazy, useMemo } from "react";
import { BillingQueueDocumentsTab } from "../../billing-queue/_components/billing-queue-documents-tab";
import { InvoiceAdjustmentPanel } from "./invoice-adjustment-panel";
import { InvoiceOverviewTab } from "./invoice-overview-tab";

const ShipmentRouteMap = lazy(() =>
  import("@/components/command-palette/_components/shipment/shipment-preview-map").then((m) => ({
    default: m.ShipmentRouteMap,
  })),
);

export default function InvoiceDetailPane({
  selectedInvoiceId,
  selectedDocumentId,
  onDocumentSelect,
}: {
  selectedInvoiceId: string | null;
  selectedDocumentId: string | null;
  onDocumentSelect: (docId: string, fileName: string) => void;
}) {
  const { data: invoice, isLoading } = useQuery({
    ...queries.invoice.get(selectedInvoiceId ?? ""),
    enabled: !!selectedInvoiceId,
  });
  const lineageQuery = useQuery({
    ...queries["invoice-adjustment"].lineage(invoice?.correctionGroupId ?? ""),
    enabled: Boolean(invoice?.correctionGroupId),
  });
  const latestAdjustment = useMemo(() => {
    if (!lineageQuery.data) {
      return null;
    }

    return (
      [...lineageQuery.data.adjustments].sort(
        (left, right) => right.createdAt - left.createdAt,
      )[0] ?? null
    );
  }, [lineageQuery.data]);
  const latestAdjustmentDetailQuery = useQuery({
    ...queries["invoice-adjustment"].get(latestAdjustment?.id ?? ""),
    enabled: Boolean(latestAdjustment?.id),
  });

  const { mutate: postInvoice, isPending: isPosting } = usePostInvoice();

  if (!selectedInvoiceId) {
    return (
      <div className="flex h-full items-center justify-center">
        <EmptyState
          title="No invoice selected"
          description="Select an invoice to review the posted billing record, shipment references, and receivable details."
          icons={[ReceiptTextIcon, FileTextIcon, PackageCheckIcon]}
          className="flex h-full max-w-none flex-col items-center justify-center rounded-none border-none p-6 shadow-none"
        />
      </div>
    );
  }

  if (isLoading || !invoice) {
    return (
      <div className="flex flex-col gap-4 p-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  const shipment = invoice.shipment;
  const customer = invoice.customer;
  const totalAmount = Number(invoice.totalAmount ?? 0);
  const customerName = customer?.name ?? invoice.billToName;
  const isCurrentVersion =
    !invoice.correctionGroupId ||
    lineageQuery.data?.correctionGroup.currentInvoiceId === invoice.id;
  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 space-y-4 border-b px-4 py-4">
        <div className="flex flex-wrap items-center justify-between">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold">{invoice.number}</h2>
            <PlainInvoiceStatusBadge status={invoice.status} />
            <PlainSettlementStatusBadge status={invoice.settlementStatus} />
          </div>
          <div className="flex items-center gap-2">
            {invoice.status === "Posted" ? (
              <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
                <CheckIcon className="size-3.5 text-green-600" />
                Posted
              </span>
            ) : (
              <Button size="sm" onClick={() => postInvoice(invoice.id)} disabled={isPosting}>
                <SendIcon className="size-3.5" />
                Post Invoice
              </Button>
            )}
            <InvoiceAdjustmentPanel invoice={invoice} />
          </div>
        </div>

        <div className="flex items-baseline gap-3">
          <span className="text-2xl font-bold tabular-nums">
            {formatCurrency(totalAmount, invoice.currencyCode)}
          </span>
          <span className="text-sm text-muted-foreground">{customerName}</span>
        </div>

        <div className="grid grid-cols-2 gap-x-6 gap-y-2 sm:grid-cols-3 lg:grid-cols-4">
          <MetadataCell label="Invoice Date" value={formatUnixDate(invoice.invoiceDate)} />
          <MetadataCell label="Due Date" value={formatUnixDate(invoice.dueDate)} />
          <MetadataCell label="Payment Terms" value={invoice.paymentTerm} />
          <MetadataCell label="Bill Type" value={invoice.billType} />
          {invoice.shipmentProNumber ? (
            <MetadataCell label="PRO Number" value={invoice.shipmentProNumber} />
          ) : null}
          {invoice.shipmentBol ? <MetadataCell label="BOL" value={invoice.shipmentBol} /> : null}
        </div>
      </div>

      {shipment?.moves && shipment.moves.length > 0 ? (
        <div className="h-32 w-full shrink-0 border-b">
          <ShipmentRouteMap moves={shipment.moves} containerClassName="rounded-none border-b" />
        </div>
      ) : null}

      <Tabs defaultValue="overview" className="flex min-h-0 flex-1 flex-col">
        <TabsList variant="underline" className="w-full border-b border-border">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="charges">Charges</TabsTrigger>
          <TabsTrigger value="documents">Documents</TabsTrigger>
          <TabsTrigger value="activity">Activity</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-0 min-h-0 flex-1">
          <InvoiceOverviewTab
            invoice={invoice}
            isCurrentVersion={isCurrentVersion}
            correctionSummary={lineageQuery.data}
            latestAdjustment={latestAdjustment}
            latestAdjustmentDetail={latestAdjustmentDetailQuery.data}
          />
        </TabsContent>

        <TabsContent value="charges" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="p-4">
              <div className="overflow-hidden rounded-xl border border-border">
                <table className="w-full text-sm">
                  <thead className="bg-muted/50 text-left text-muted-foreground">
                    <tr>
                      <th className="px-4 py-3 font-medium">Line</th>
                      <th className="px-4 py-3 font-medium">Description</th>
                      <th className="px-4 py-3 font-medium">Type</th>
                      <th className="px-4 py-3 text-right font-medium">Quantity</th>
                      <th className="px-4 py-3 text-right font-medium">Unit Price</th>
                      <th className="px-4 py-3 text-right font-medium">Amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {invoice.lines.map((line) => (
                      <tr key={line.id} className="border-t transition-colors hover:bg-muted/50">
                        <td className="px-4 py-3 font-mono text-xs">{line.lineNumber}</td>
                        <td className="px-4 py-3">{line.description}</td>
                        <td className="px-4 py-3">
                          <Badge variant={LINE_TYPE_VARIANTS[line.type]}>{line.type}</Badge>
                        </td>
                        <td className="px-4 py-3 text-right tabular-nums">
                          {Number(line.quantity ?? 0)}
                        </td>
                        <td className="px-4 py-3 text-right tabular-nums">
                          {formatCurrency(Number(line.unitPrice ?? 0), invoice.currencyCode)}
                        </td>
                        <td className="px-4 py-3 text-right font-medium tabular-nums">
                          {formatCurrency(Number(line.amount ?? 0), invoice.currencyCode)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot className="border-t bg-muted/30">
                    <tr>
                      <td
                        colSpan={5}
                        className="px-4 py-2.5 text-right text-sm text-muted-foreground"
                      >
                        Subtotal
                      </td>
                      <td className="px-4 py-2.5 text-right text-sm tabular-nums">
                        {formatCurrency(Number(invoice.subtotalAmount ?? 0), invoice.currencyCode)}
                      </td>
                    </tr>
                    <tr>
                      <td
                        colSpan={5}
                        className="px-4 py-2.5 text-right text-sm text-muted-foreground"
                      >
                        Other Charges
                      </td>
                      <td className="px-4 py-2.5 text-right text-sm tabular-nums">
                        {formatCurrency(Number(invoice.otherAmount ?? 0), invoice.currencyCode)}
                      </td>
                    </tr>
                    <tr className="border-t">
                      <td colSpan={5} className="px-4 py-3 text-right text-sm font-semibold">
                        Total
                      </td>
                      <td className="px-4 py-3 text-right text-sm font-bold tabular-nums">
                        {formatCurrency(totalAmount, invoice.currencyCode)}
                      </td>
                    </tr>
                  </tfoot>
                </table>
              </div>
            </div>
          </ScrollArea>
        </TabsContent>
        <TabsContent value="documents" className="mt-0 min-h-0 flex-1">
          {shipment ? (
            <BillingQueueDocumentsTab
              shipmentId={invoice.shipmentId}
              selectedDocumentId={selectedDocumentId}
              onDocumentSelect={onDocumentSelect}
              isEditable={false}
              context="invoice"
            />
          ) : (
            <div className="flex h-full items-center justify-center p-6">
              <EmptyState
                title="No shipment documents available"
                description="This invoice does not currently have shipment context loaded for document review."
                icons={[FileTextIcon, ReceiptTextIcon, PackageCheckIcon]}
                className="max-w-xl border-none p-8 shadow-none"
              />
            </div>
          )}
        </TabsContent>
        <TabsContent value="activity" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="px-4 py-3">
              <AuditTab resourceId={invoice.id} />
            </div>
          </ScrollArea>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function MetadataCell({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-medium">{value}</p>
    </div>
  );
}

const LINE_TYPE_VARIANTS: Record<InvoiceLineType, BadgeVariant> = {
  Freight: "info",
  Accessorial: "purple",
};
