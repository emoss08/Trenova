import AuditTab from "@/components/audit-tab";
import { EmptyState } from "@/components/empty-state";
import { PlainInvoiceStatusBadge, PlainSettlementStatusBadge } from "@/components/status-badge";
import { formatFileSize } from "@/components/documents/document-upload-zone";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePostInvoice } from "@/hooks/use-post-invoice";
import { ApiRequestError } from "@/lib/api";
import { formatUnixDate } from "@/lib/date";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  Invoice,
  InvoiceEmailAttempt,
  InvoiceLineType,
  InvoiceSendPlan,
  InvoiceSendStatus,
} from "@/types/invoice";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  DownloadIcon,
  FileTextIcon,
  MailIcon,
  PackageCheckIcon,
  ReceiptTextIcon,
  SendIcon,
} from "lucide-react";
import { lazy, useEffect, useMemo, useRef } from "react";
import { toast } from "sonner";
import { BillingQueueDocumentsTab } from "../../billing-queue/_components/billing-queue-documents-tab";
import { InvoiceAdjustmentPanel } from "./invoice-adjustment-panel";
import { InvoiceOverviewTab } from "./invoice-overview-tab";

const ShipmentRouteMap = lazy(() =>
  import("@/components/command-palette/_components/shipment/shipment-preview-map").then((m) => ({
    default: m.ShipmentRouteMap,
  })),
);

const INVOICE_SEND_HISTORY_PAGE_SIZE = 20;

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
          <TabsTrigger value="delivery">Delivery</TabsTrigger>
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

        <TabsContent value="delivery" className="mt-0 min-h-0 flex-1">
          <InvoiceDeliveryTab invoice={invoice} />
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
          <div className="flex h-full flex-col">
            <div className="min-h-0 flex-1">
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
            </div>
          </div>
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

function InvoiceDeliveryTab({ invoice }: { invoice: Invoice }) {
  const queryClient = useQueryClient();
  const sendPlanQuery = useQuery(queries.invoice.sendPlan(invoice.id));

  const invalidateInvoiceDelivery = () => {
    void queryClient.invalidateQueries({ queryKey: ["invoice"] });
    void queryClient.invalidateQueries({ queryKey: ["invoice-list"] });
    void queryClient.invalidateQueries({
      queryKey: ["documents", "shipment", invoice.shipmentId],
    });
  };

  const generateMutation = useMutation({
    mutationFn: () => apiService.invoiceService.generatePdf(invoice.id),
    onSuccess: () => {
      invalidateInvoiceDelivery();
      toast.success(`${invoice.number} PDF generation started`);
    },
    onError: () => toast.error("Failed to generate invoice PDF"),
  });

  const sendMutation = useMutation({
    mutationFn: () => apiService.invoiceService.send(invoice.id),
    onSuccess: () => {
      invalidateInvoiceDelivery();
      void queryClient.invalidateQueries(queries.invoice.sendPlan(invoice.id));
      void queryClient.invalidateQueries({
        queryKey: queries.invoice.emailAttempts(invoice.id).queryKey,
      });
      toast.success(`${invoice.number} send attempted`);
    },
    onError: (error) =>
      toast.error(error instanceof ApiRequestError ? error.message : "Failed to send invoice"),
  });

  const sendPlan = sendPlanQuery.data;
  const sendDisabledReason = getInvoiceSendDisabledReason(
    invoice,
    sendPlan,
    sendPlanQuery.isLoading,
  );
  const canSend = !sendDisabledReason;
  const hasGeneratedPDF = Boolean(invoice.pdfDocumentId);
  const pdfActionLabel = hasGeneratedPDF ? "Regenerate" : "Generate PDF";

  const reprintPDF = async () => {
    if (!invoice.pdfDocumentId) {
      return;
    }
    const downloadUrl = await apiService.documentService.getDownloadUrl(invoice.pdfDocumentId);
    window.open(downloadUrl, "_blank", "noopener,noreferrer");
  };

  return (
    <div className="grid h-full min-h-0 gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <ScrollArea className="min-h-[20rem] lg:h-full lg:min-h-0" viewportClassName="pr-2">
        <div className="space-y-4">
          <div className="rounded-md border border-border p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-semibold">Email Delivery</h3>
                  <Badge variant={SEND_STATUS_VARIANTS[invoice.sendStatus ?? "NotSent"]}>
                    {invoice.sendStatus ?? "NotSent"}
                  </Badge>
                </div>
                <p className="mt-1 text-sm text-muted-foreground">
                  {invoice.sentAt ? `Last sent ${formatUnixDate(invoice.sentAt)}` : "Not sent yet"}
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {hasGeneratedPDF ? (
                  <Button size="sm" variant="outline" onClick={() => void reprintPDF()}>
                    <DownloadIcon className="size-3.5" />
                    Reprint
                  </Button>
                ) : null}
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => generateMutation.mutate()}
                  disabled={generateMutation.isPending}
                >
                  <FileTextIcon className="size-3.5" />
                  {pdfActionLabel}
                </Button>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <span className="inline-flex" tabIndex={!canSend ? 0 : undefined}>
                        <Button
                          size="sm"
                          onClick={() => sendMutation.mutate()}
                          disabled={!canSend || sendMutation.isPending}
                        >
                          <MailIcon className="size-3.5" />
                          {invoice.sendStatus === "Sent" ? "Resend" : "Send"}
                        </Button>
                      </span>
                    }
                  />
                  {!canSend ? (
                    <TooltipContent className="max-w-72" side="top" sideOffset={8}>
                      {sendDisabledReason}
                    </TooltipContent>
                  ) : null}
                </Tooltip>
              </div>
            </div>
            {invoice.lastSendError ? (
              <DeliveryNotice tone="error" message={invoice.lastSendError} />
            ) : null}
            {invoice.lastSendWarning ? (
              <DeliveryNotice tone="warning" message={invoice.lastSendWarning} />
            ) : null}
          </div>

          <div className="rounded-md border border-border p-4">
            <h3 className="text-sm font-semibold">Send Plan</h3>
            {sendPlanQuery.isLoading ? (
              <Skeleton className="mt-3 h-24 w-full" />
            ) : sendPlan ? (
              <div className="mt-3 space-y-3">
                {sendPlan.errors.map((error) => (
                  <DeliveryNotice key={error} tone="error" message={error} />
                ))}
                {sendPlan.warnings.map((warning) => (
                  <DeliveryNotice key={warning} tone="warning" message={warning} />
                ))}
                <SendPlanSummary sendPlan={sendPlan} />
                <MessagePreview sendPlan={sendPlan} />
                <DeliveryPackageList parts={sendPlan.parts} />
              </div>
            ) : (
              <p className="mt-2 text-sm text-muted-foreground">Send plan unavailable.</p>
            )}
          </div>
        </div>
      </ScrollArea>

      <InvoiceSendHistoryPanel invoiceId={invoice.id} />
    </div>
  );
}

function SendPlanSummary({ sendPlan }: { sendPlan: InvoiceSendPlan }) {
  const attachmentCount = getSendPlanAttachmentCount(sendPlan);
  const linkCount = getSendPlanLinkCount(sendPlan);

  return (
    <div className="grid grid-cols-2 gap-2 xl:grid-cols-4">
      <SendPlanSummaryCell label="To recipients">
        <RecipientPreview recipients={sendPlan.recipients.to} />
      </SendPlanSummaryCell>
      <SendPlanSummaryCell label="Provider limit">
        {formatFileSize(sendPlan.providerLimitBytes)}
      </SendPlanSummaryCell>
      <SendPlanSummaryCell label="Body size">
        {formatFileSize(sendPlan.estimatedBodyBytes)}
      </SendPlanSummaryCell>
      <SendPlanSummaryCell
        label="Email parts"
        detail={formatPackageBreakdown(attachmentCount, linkCount)}
      >
        {formatCount(sendPlan.parts.length, "part")}
      </SendPlanSummaryCell>
    </div>
  );
}

function SendPlanSummaryCell({
  label,
  detail,
  children,
}: {
  label: string;
  detail?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="min-w-0 rounded-md border bg-muted/20 p-2.5">
      <p className="text-xs text-muted-foreground">{label}</p>
      <div className="mt-1 min-w-0 text-sm font-medium">{children}</div>
      {detail ? <p className="mt-1 truncate text-xs text-muted-foreground">{detail}</p> : null}
    </div>
  );
}

function RecipientPreview({ recipients }: { recipients: string[] }) {
  if (recipients.length === 0) {
    return <span className="text-muted-foreground">No recipients</span>;
  }

  if (recipients.length === 1) {
    return <span className="block truncate">{recipients[0]}</span>;
  }

  const remainingCount = recipients.length - 1;

  return (
    <span className="flex w-full min-w-0 items-center gap-1.5">
      <span className="min-w-0 flex-1 truncate">{recipients[0]}</span>
      <HoverCard>
        <HoverCardTrigger
          render={
            <button
              type="button"
              className="shrink-0 rounded-sm text-xs font-medium text-blue-600 underline-offset-2 hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none dark:text-blue-400"
              aria-label={`Show ${recipients.length} To recipients`}
            >
              +{remainingCount} more
            </button>
          }
        />
        <RecipientHoverList recipients={recipients} />
      </HoverCard>
    </span>
  );
}

function RecipientHoverList({ recipients }: { recipients: string[] }) {
  return (
    <HoverCardContent side="top" align="start" className="w-80 max-w-[calc(100vw-2rem)] p-3">
      <p className="text-xs font-medium text-muted-foreground">To recipients</p>
      <ul className="mt-2 max-h-60 space-y-1 overflow-y-auto text-sm">
        {recipients.map((recipient, index) => (
          <li
            key={`${recipient}-${index}`}
            className="rounded-md bg-muted/40 px-2 py-1.5 font-mono text-xs break-all"
          >
            {recipient}
          </li>
        ))}
      </ul>
    </HoverCardContent>
  );
}

function MessagePreview({ sendPlan }: { sendPlan: InvoiceSendPlan }) {
  return (
    <div className="rounded-md border bg-muted/20 p-3">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span className="min-w-0">
          From{" "}
          <span className="font-medium text-foreground">
            {sendPlan.fromEmail || "Assigned profile"}
          </span>
        </span>
        <span className="flex items-center gap-1.5">
          Read receipt
          <Badge variant={sendPlan.openTracking ? "active" : "outline"}>
            {sendPlan.openTracking ? "Enabled" : "Disabled"}
          </Badge>
        </span>
      </div>
      <div className="mt-3 space-y-2">
        <div className="min-w-0">
          <p className="text-xs font-medium text-muted-foreground">Subject</p>
          <p className="truncate text-sm font-semibold">{sendPlan.subject || "No subject"}</p>
        </div>
        <div>
          <p className="text-xs font-medium text-muted-foreground">Body</p>
          <p className="line-clamp-4 text-sm whitespace-pre-line text-muted-foreground">
            {sendPlan.body || "No body content"}
          </p>
        </div>
      </div>
    </div>
  );
}

function DeliveryPackageList({ parts }: { parts: InvoiceSendPlan["parts"] }) {
  if (parts.length === 0) {
    return (
      <div className="rounded-md border bg-muted/20 p-3 text-sm text-muted-foreground">
        No delivery package parts.
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {parts.map((part) => {
        const documentCount = part.attachments.length + part.links.length;

        return (
          <div key={part.partNumber} className="rounded-md border bg-muted/20 p-3">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div className="flex min-w-0 items-center gap-2">
                <span className="text-sm font-semibold">Part {part.partNumber}</span>
                <Badge variant="outline">{formatFileSize(part.estimatedSizeBytes)}</Badge>
              </div>
              <div className="flex shrink-0 flex-wrap items-center justify-end gap-1.5">
                <Badge variant="outline">
                  {formatCount(part.attachments.length, "attachment")}
                </Badge>
                <Badge variant="outline">{formatCount(part.links.length, "link")}</Badge>
              </div>
            </div>

            {part.warnings.length > 0 ? (
              <div className="mt-2 space-y-1">
                {part.warnings.map((warning) => (
                  <div
                    key={warning}
                    className="flex gap-1.5 text-xs text-yellow-700 dark:text-yellow-400"
                  >
                    <AlertTriangleIcon className="mt-0.5 size-3 shrink-0" />
                    <span>{warning}</span>
                  </div>
                ))}
              </div>
            ) : null}

            {documentCount > 0 ? (
              <div className="mt-3 divide-y overflow-hidden rounded-md border bg-background/60">
                {part.attachments.map((attachment) => (
                  <DeliveryPackageDocument
                    key={attachment.documentId}
                    label={attachment.invoicePdf ? "Invoice PDF" : "Attachment"}
                    fileName={attachment.fileName}
                    sizeBytes={attachment.sizeBytes}
                  />
                ))}
                {part.links.map((link) => (
                  <DeliveryPackageDocument
                    key={link.documentId}
                    label={link.reason ? `Link - ${link.reason}` : "Link"}
                    fileName={link.fileName}
                    sizeBytes={link.sizeBytes}
                  />
                ))}
              </div>
            ) : (
              <p className="mt-3 text-xs text-muted-foreground">
                No attachments or links in this part.
              </p>
            )}
          </div>
        );
      })}
    </div>
  );
}

function DeliveryPackageDocument({
  label,
  fileName,
  sizeBytes,
}: {
  label: string;
  fileName: string;
  sizeBytes: number;
}) {
  return (
    <div className="flex min-w-0 items-center justify-between gap-3 px-2.5 py-2 text-sm">
      <div className="min-w-0">
        <p className="truncate font-medium">{fileName}</p>
        <p className="text-xs text-muted-foreground">{label}</p>
      </div>
      <span className="shrink-0 text-xs text-muted-foreground tabular-nums">
        {formatFileSize(sizeBytes)}
      </span>
    </div>
  );
}

function getSendPlanAttachmentCount(sendPlan: InvoiceSendPlan): number {
  return sendPlan.parts.reduce((count, part) => count + part.attachments.length, 0);
}

function getSendPlanLinkCount(sendPlan: InvoiceSendPlan): number {
  return sendPlan.parts.reduce((count, part) => count + part.links.length, 0);
}

function formatPackageBreakdown(attachmentCount: number, linkCount: number): string {
  const packageCounts: string[] = [];
  if (attachmentCount > 0) {
    packageCounts.push(formatCount(attachmentCount, "attachment"));
  }
  if (linkCount > 0) {
    packageCounts.push(formatCount(linkCount, "link"));
  }

  return packageCounts.length > 0 ? packageCounts.join(" / ") : "No attachments or links";
}

function formatCount(count: number, singular: string): string {
  return `${count} ${count === 1 ? singular : `${singular}s`}`;
}

function InvoiceSendHistoryPanel({ invoiceId }: { invoiceId: string }) {
  const queryKey = queries.invoice.emailAttempts(invoiceId).queryKey;
  const observerTarget = useRef<HTMLDivElement>(null);

  const query = useInfiniteQuery({
    queryKey,
    queryFn: async ({ pageParam }) =>
      apiService.invoiceService.listEmailAttempts(invoiceId, {
        limit: INVOICE_SEND_HISTORY_PAGE_SIZE,
        offset: pageParam,
      }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.next || lastPage.results.length === INVOICE_SEND_HISTORY_PAGE_SIZE) {
        return lastPageParam + INVOICE_SEND_HISTORY_PAGE_SIZE;
      }
      return undefined;
    },
  });

  const attempts = useMemo(
    () => query.data?.pages.flatMap((page) => page.results) ?? [],
    [query.data?.pages],
  );
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <div className="flex min-h-[20rem] flex-col rounded-md border border-border p-4 lg:h-full lg:min-h-0">
      <h3 className="text-sm font-semibold">Send History</h3>
      <ScrollArea className="mt-3 min-h-0 flex-1" viewportClassName="pr-2">
        {query.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </div>
        ) : query.isError ? (
          <p className="rounded-md border border-red-600/30 bg-red-600/10 p-3 text-sm text-red-700 dark:text-red-400">
            Send history could not be loaded.
          </p>
        ) : attempts.length === 0 ? (
          <p className="text-sm text-muted-foreground">No email attempts recorded.</p>
        ) : (
          <div className="space-y-3">
            {attempts.map((attempt) => (
              <InvoiceSendHistoryCard key={attempt.id} attempt={attempt} />
            ))}
            {query.isFetchingNextPage ? (
              <div className="flex items-center justify-center py-4">
                <TextShimmer className="font-mono text-sm" duration={1}>
                  Loading more...
                </TextShimmer>
              </div>
            ) : null}
            <div ref={observerTarget} className="h-px" />
          </div>
        )}
      </ScrollArea>
    </div>
  );
}

function InvoiceSendHistoryCard({ attempt }: { attempt: InvoiceEmailAttempt }) {
  const status = invoiceAttemptDisplayStatus(attempt);
  const error = invoiceAttemptDisplayError(attempt);
  const sentAt = attempt.email?.sentAt ?? attempt.sentAt;
  const failedAt = attempt.email?.failedAt;
  const providerMessageId = attempt.email?.providerMessageId ?? attempt.providerMessageId;

  return (
    <div className="rounded-md border bg-muted/20 p-3">
      <div className="flex items-center justify-between gap-2">
        <Badge variant={SEND_STATUS_VARIANTS[status]}>{status}</Badge>
        <span className="text-xs text-muted-foreground">
          Part {attempt.partNumber} of {attempt.totalParts}
        </span>
      </div>
      <p className="mt-2 truncate text-sm font-medium">{attempt.subject}</p>
      <p className="mt-1 text-xs text-muted-foreground">
        {sentAt
          ? formatUnixDate(sentAt)
          : failedAt
            ? `Failed ${formatUnixDate(failedAt)}`
            : "Not sent"}
      </p>
      {providerMessageId ? (
        <p className="mt-1 truncate text-xs text-muted-foreground">
          Provider ID: {providerMessageId}
        </p>
      ) : null}
      {error ? <DeliveryNotice tone="error" message={error} /> : null}
    </div>
  );
}

function getInvoiceSendDisabledReason(
  invoice: Invoice,
  sendPlan: InvoiceSendPlan | undefined,
  sendPlanLoading: boolean,
): string | null {
  if (sendPlanLoading) {
    return "Checking invoice email configuration.";
  }
  if (!invoice.pdfDocumentId) {
    return "Generate the invoice PDF before sending.";
  }
  if (!sendPlan) {
    return "Send plan is unavailable.";
  }
  if (sendPlan.errors.length > 0) {
    return sendPlan.errors.join("; ");
  }
  return null;
}

function invoiceAttemptDisplayStatus(attempt: InvoiceEmailAttempt): InvoiceSendStatus {
  switch (attempt.email?.status) {
    case "Sent":
    case "Delivered":
    case "Opened":
    case "Clicked":
      return "Sent";
    case "Failed":
    case "Bounced":
    case "Complained":
    case "Suppressed":
      return "Failed";
    case "Queued":
    case "Sending":
      return "Sending";
    default:
      return attempt.status;
  }
}

function invoiceAttemptDisplayError(attempt: InvoiceEmailAttempt): string | null {
  if (attempt.email?.lastError) {
    return attempt.email.lastError;
  }
  return attempt.error || null;
}

function DeliveryNotice({ tone, message }: { tone: "error" | "warning"; message: string }) {
  return (
    <div
      className={
        tone === "error"
          ? "mt-3 flex gap-2 rounded-md border border-red-600/30 bg-red-600/10 p-2 text-sm text-red-700 dark:text-red-400"
          : "mt-3 flex gap-2 rounded-md border border-yellow-600/30 bg-yellow-600/10 p-2 text-sm text-yellow-700 dark:text-yellow-400"
      }
    >
      <AlertTriangleIcon className="mt-0.5 size-3.5 shrink-0" />
      <span>{message}</span>
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

const SEND_STATUS_VARIANTS: Record<InvoiceSendStatus, BadgeVariant> = {
  NotSent: "outline",
  Sending: "info",
  Sent: "active",
  PartiallySent: "warning",
  Failed: "inactive",
};
