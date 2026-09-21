import { useT } from "@trenova/shared/i18n/use-t";
import AuditTab from "@/components/audit-tab";
import { BillingDetailUnselected } from "@/components/billing/billing-empty";
import { EmptyState } from "@/components/empty-state";
import {
  PlainInvoiceScopeBadge,
  PlainInvoiceSplitBadge,
  PlainInvoiceStatusBadge,
  PlainSettlementStatusBadge,
} from "@trenova/shared/components/status-badge";
import { formatFileSize } from "@/components/documents/document-upload-zone";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { usePostInvoice } from "@/hooks/use-post-invoice";
import type { InvoiceArContext } from "@/lib/graphql/invoice";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { formatUnixDate, formatUnixDateTime } from "@trenova/shared/lib/date";
import { invoiceBillingPeriod, invoiceBillsSingleShipment } from "@/lib/invoice-scope";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import type {
  Invoice,
  InvoiceEmailAttempt,
  InvoiceSendPlan,
  InvoiceSendStatus,
} from "@trenova/shared/types/invoice";
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
import { useQueryStates } from "nuqs";
import { lazy, useEffect, useMemo, useRef } from "react";
import { toast } from "sonner";
import { BillingQueueDocumentsTab } from "../../billing-queue/_components/billing-queue-documents-tab";
import { InvoiceActionsMenu } from "./invoice-actions-menu";
import { InvoiceAdjustmentPanel } from "./invoice-adjustment-panel";
import { InvoiceChargesTab } from "./invoice-charges-tab";
import { InvoiceDisputesTab } from "./invoice-disputes-tab";
import { InvoiceEdiDeliveryCard } from "./invoice-edi-delivery-card";
import { InvoiceOverviewTab } from "./invoice-overview-tab";
import { InvoicePaymentsTab } from "./invoice-payments-tab";
import { InvoiceShareDialog } from "./invoice-share-dialog";
import { invoiceDetailTabSearchParamsParser, isInvoiceDetailTab } from "../use-invoice-state";

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
  const t = useT();

  const [{ tab }, setTabState] = useQueryStates(invoiceDetailTabSearchParamsParser);
  const { data: invoice, isLoading } = useQuery({
    ...queries.invoice.get(selectedInvoiceId ?? ""),
    enabled: !!selectedInvoiceId,
  });
  // The AR side (balance, applications, disputes, EDI plan) is resolver-only
  // and is read beside the REST detail rather than folded into it.
  const arContextQuery = useQuery({
    ...queries.invoice.arContext(selectedInvoiceId ?? ""),
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
      <BillingDetailUnselected
        layout="tabs"
        title={t("Nothing open")}
        description={t(
          "Pick an invoice from the list to review its charges, documents and what has been sent or paid.",
        )}
      />
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
  const billedShipmentCount = invoice.shipmentCount;
  const billingPeriod = invoiceBillingPeriod(invoice);
  const billsSingleShipment = invoiceBillsSingleShipment(invoice.scope);
  const customerName = customer?.name ?? invoice.billToName;
  const isCurrentVersion =
    !invoice.correctionGroupId ||
    lineageQuery.data?.correctionGroup.currentInvoiceId === invoice.id;
  const arContext = arContextQuery.data ?? null;
  const daysPastDue = arContext?.daysPastDue ?? null;
  const isVoided = invoice.status === "Voided";
  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 space-y-4 border-b px-4 py-4">
        <div className="flex flex-wrap items-center justify-between">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold">{invoice.number}</h2>
            <PlainInvoiceScopeBadge scope={invoice.scope} />
            <PlainInvoiceStatusBadge status={invoice.status} />
            <PlainSettlementStatusBadge status={invoice.settlementStatus} />
            <PlainInvoiceSplitBadge isSplitBill={invoice.isSplitBill} />
          </div>
          <div className="flex items-center gap-2">
            {invoice.status === "Posted" ? (
              <span className="text-muted-foreground flex items-center gap-1.5 text-sm">
                <CheckIcon className="size-3.5 text-success-foreground" />
                {t("Posted")}
              </span>
            ) : isVoided ? null : (
              <Button size="sm" onClick={() => postInvoice(invoice.id)} disabled={isPosting}>
                <SendIcon className="size-3.5" />
                {t("Post Invoice")}
              </Button>
            )}
            <InvoiceShareDialog invoice={invoice} />
            {isVoided ? null : <InvoiceAdjustmentPanel invoice={invoice} />}
            <InvoiceActionsMenu invoice={invoice} arContext={arContext} />
          </div>
        </div>

        {isVoided ? <VoidedNotice invoice={invoice} /> : null}

        <div className="flex items-baseline gap-3">
          <span className="text-2xl font-semibold tabular-nums">
            {formatCurrency(totalAmount, invoice.currencyCode)}
          </span>
          <span className="text-muted-foreground text-sm">{customerName}</span>
        </div>

        <div className="grid grid-cols-2 gap-x-6 gap-y-2 sm:grid-cols-3 lg:grid-cols-4">
          <MetadataCell label={t("Invoice Date")} value={formatUnixDate(invoice.invoiceDate)} />
          <MetadataCell label={t("Due Date")} value={formatUnixDate(invoice.dueDate)} />
          <MetadataCell label={t("Payment Terms")} value={invoice.paymentTerm} />
          <MetadataCell label={t("Bill Type")} value={invoice.billType} />
          {daysPastDue !== null && daysPastDue > 0 ? (
            <MetadataCell label={t("Days past due")} value={String(daysPastDue)} />
          ) : null}
          {billingPeriod ? (
            <MetadataCell label={t("Billing Period")} value={billingPeriod} />
          ) : null}
          {billedShipmentCount > 1 || !billsSingleShipment ? (
            <MetadataCell label={t("Shipments")} value={String(billedShipmentCount)} />
          ) : null}
          {billedShipmentCount <= 1 && billsSingleShipment && invoice.shipmentProNumber ? (
            <MetadataCell label={t("PRO Number")} value={invoice.shipmentProNumber} />
          ) : null}
          {billedShipmentCount <= 1 && billsSingleShipment && invoice.shipmentBol ? (
            <MetadataCell label={t("BOL")} value={invoice.shipmentBol} />
          ) : null}
        </div>
      </div>

      {shipment?.moves && shipment.moves.length > 0 ? (
        <div className="h-32 w-full shrink-0 border-b">
          <ShipmentRouteMap moves={shipment.moves} containerClassName="rounded-none border-b" />
        </div>
      ) : null}

      <Tabs
        value={tab}
        onValueChange={(value) => {
          if (isInvoiceDetailTab(value)) {
            void setTabState({ tab: value });
          }
        }}
        className="flex min-h-0 flex-1 flex-col"
      >
        <TabsList variant="underline" className="border-border w-full border-b">
          <TabsTrigger value="overview">{t("Overview")}</TabsTrigger>
          <TabsTrigger value="delivery">{t("Delivery")}</TabsTrigger>
          <TabsTrigger value="charges">{t("Charges")}</TabsTrigger>
          <TabsTrigger value="payments">{t("Payments")}</TabsTrigger>
          <TabsTrigger value="disputes">{t("Disputes")}</TabsTrigger>
          <TabsTrigger value="documents">{t("Documents")}</TabsTrigger>
          <TabsTrigger value="activity">{t("Activity")}</TabsTrigger>
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
          <InvoiceDeliveryTab
            invoice={invoice}
            arContext={arContext}
            arContextLoading={arContextQuery.isLoading}
          />
        </TabsContent>

        <TabsContent value="charges" className="mt-0 min-h-0 flex-1">
          <InvoiceChargesTab invoice={invoice} />
        </TabsContent>
        <TabsContent value="payments" className="mt-0 min-h-0 flex-1">
          <InvoicePaymentsTab
            invoice={invoice}
            arContext={arContext}
            isLoading={arContextQuery.isLoading}
          />
        </TabsContent>
        <TabsContent value="disputes" className="mt-0 min-h-0 flex-1">
          <InvoiceDisputesTab
            invoice={invoice}
            arContext={arContext}
            isLoading={arContextQuery.isLoading}
          />
        </TabsContent>
        <TabsContent value="documents" className="mt-0 min-h-0 flex-1">
          <div className="flex h-full flex-col">
            <div className="min-h-0 flex-1">
              {shipment ? (
                <BillingQueueDocumentsTab
                  shipmentId={invoice.shipmentId ?? ""}
                  selectedDocumentId={selectedDocumentId}
                  onDocumentSelect={onDocumentSelect}
                  isEditable={false}
                  context="invoice"
                />
              ) : (
                <div className="flex h-full items-center justify-center p-6">
                  <EmptyState
                    title={t("No shipment documents available")}
                    description={
                      billsSingleShipment
                        ? t(
                            "This invoice does not currently have shipment context loaded for document review.",
                          )
                        : t(
                            "This invoice bills {0, plural, one {# shipment} other {# shipments}}. Their documents stay on each shipment; the Charges tab lists every shipment it covers.",
                            billedShipmentCount,
                          )
                    }
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

function VoidedNotice({ invoice }: { invoice: Invoice }) {
  const t = useT();

  return (
    <div className="flex gap-3 rounded-md border border-danger-border bg-danger-subtle/60 p-3 dark:border-danger-border dark:bg-danger-subtle/30">
      <AlertTriangleIcon className="mt-0.5 size-4 shrink-0 text-danger-foreground" />
      <div className="min-w-0 text-sm">
        <p className="font-medium text-danger-foreground">
          <span>{t("Voided {0}", formatUnixDateTime(invoice.voidedAt))}</span>
          <span className="mx-1">·</span>
          <span>
            {invoice.voidDisposition === "Rebill"
              ? t("Released for rebilling")
              : t("Freight retired, not rebilled")}
          </span>
        </p>
        {invoice.voidReason ? (
          <p className="mt-0.5 text-danger-foreground/80">{invoice.voidReason}</p>
        ) : null}
      </div>
    </div>
  );
}

function InvoiceDeliveryTab({
  invoice,
  arContext,
  arContextLoading,
}: {
  invoice: Invoice;
  arContext: InvoiceArContext | null;
  arContextLoading: boolean;
}) {
  const t = useT();

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
    onError: () => toast.error(t("Failed to generate invoice PDF")),
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
          <div className="border-border rounded-md border p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-semibold">{t("Email Delivery")}</h3>
                  <Badge variant={SEND_STATUS_VARIANTS[invoice.sendStatus ?? "NotSent"]}>
                    {invoice.sendStatus ?? t("NotSent")}
                  </Badge>
                </div>
                <p className="text-muted-foreground mt-1 text-sm">
                  {invoice.sentAt
                    ? t("Last sent {0}", formatUnixDate(invoice.sentAt))
                    : t("Not sent yet")}
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {hasGeneratedPDF ? (
                  <Button size="sm" variant="outline" onClick={() => void reprintPDF()}>
                    <DownloadIcon className="size-3.5" />
                    {t("Reprint")}
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
                          {invoice.sendStatus === "Sent" ? t("Resend") : t("Send")}
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

          <InvoiceEdiDeliveryCard
            invoice={invoice}
            arContext={arContext}
            isLoading={arContextLoading}
          />

          <div className="border-border rounded-md border p-4">
            <h3 className="text-sm font-semibold">{t("Send Plan")}</h3>
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
              <p className="text-muted-foreground mt-2 text-sm">{t("Send plan unavailable.")}</p>
            )}
          </div>
        </div>
      </ScrollArea>

      <InvoiceSendHistoryPanel invoiceId={invoice.id} />
    </div>
  );
}

function SendPlanSummary({ sendPlan }: { sendPlan: InvoiceSendPlan }) {
  const t = useT();

  const attachmentCount = getSendPlanAttachmentCount(sendPlan);
  const linkCount = getSendPlanLinkCount(sendPlan);

  return (
    <div className="grid grid-cols-2 gap-2 xl:grid-cols-4">
      <SendPlanSummaryCell label={t("To recipients")}>
        <RecipientPreview recipients={sendPlan.recipients.to} />
      </SendPlanSummaryCell>
      <SendPlanSummaryCell label={t("Provider limit")}>
        {formatFileSize(sendPlan.providerLimitBytes)}
      </SendPlanSummaryCell>
      <SendPlanSummaryCell label={t("Body size")}>
        {formatFileSize(sendPlan.estimatedBodyBytes)}
      </SendPlanSummaryCell>
      <SendPlanSummaryCell
        label={t("Email parts")}
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
    <div className="bg-muted/20 min-w-0 rounded-md border p-2.5">
      <p className="text-muted-foreground text-xs">{label}</p>
      <div className="mt-1 min-w-0 text-sm font-medium">{children}</div>
      {detail ? <p className="text-muted-foreground mt-1 truncate text-xs">{detail}</p> : null}
    </div>
  );
}

function RecipientPreview({ recipients }: { recipients: string[] }) {
  const t = useT();

  if (recipients.length === 0) {
    return <span className="text-muted-foreground">{t("No recipients")}</span>;
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
 className="ui-focus-ring shrink-0 rounded-sm text-xs font-medium text-info-foreground underline-offset-2 hover:underline dark:text-info-foreground"
              aria-label={`Show ${recipients.length} To recipients`}
            >
              {t("+{0} more", remainingCount)}
            </button>
          }
        />
        <RecipientHoverList recipients={recipients} />
      </HoverCard>
    </span>
  );
}

function RecipientHoverList({ recipients }: { recipients: string[] }) {
  const t = useT();

  return (
    <HoverCardContent side="top" align="start" className="w-80 max-w-[calc(100vw-2rem)] p-3">
      <p className="text-muted-foreground text-xs font-medium">{t("To recipients")}</p>
      <ul className="mt-2 max-h-60 space-y-1 overflow-y-auto text-sm">
        {recipients.map((recipient, index) => (
          <li
            key={`${recipient}-${index}`}
            className="bg-muted/40 rounded-md px-2 py-1.5 font-mono text-xs break-all"
          >
            {recipient}
          </li>
        ))}
      </ul>
    </HoverCardContent>
  );
}

function MessagePreview({ sendPlan }: { sendPlan: InvoiceSendPlan }) {
  const t = useT();

  return (
    <div className="bg-muted/20 rounded-md border p-3">
      <div className="text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
        <span className="min-w-0">
          {t("From")}{" "}
          <span className="text-foreground font-medium">
            {sendPlan.fromEmail || t("Assigned profile")}
          </span>
        </span>
        <span className="flex items-center gap-1.5">
          {t("Read receipt")}
          <Badge variant={sendPlan.openTracking ? "success" : "neutral"}>
            {sendPlan.openTracking ? t("Enabled") : t("Disabled")}
          </Badge>
        </span>
      </div>
      <div className="mt-3 space-y-2">
        <div className="min-w-0">
          <p className="text-muted-foreground text-xs font-medium">{t("Subject")}</p>
          <p className="truncate text-sm font-semibold">{sendPlan.subject || t("No subject")}</p>
        </div>
        <div>
          <p className="text-muted-foreground text-xs font-medium">{t("Body")}</p>
          <p className="text-muted-foreground line-clamp-4 text-sm whitespace-pre-line">
            {sendPlan.body || t("No body content")}
          </p>
        </div>
      </div>
    </div>
  );
}

function DeliveryPackageList({ parts }: { parts: InvoiceSendPlan["parts"] }) {
  const t = useT();

  if (parts.length === 0) {
    return (
      <div className="bg-muted/20 text-muted-foreground rounded-md border p-3 text-sm">
        {t("No delivery package parts.")}
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {parts.map((part) => {
        const documentCount = part.attachments.length + part.links.length;

        return (
          <div key={part.partNumber} className="bg-muted/20 rounded-md border p-3">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div className="flex min-w-0 items-center gap-2">
                <span className="text-sm font-semibold">{t("Part {0}", part.partNumber)}</span>
                <Badge variant="neutral" appearance="outline">{formatFileSize(part.estimatedSizeBytes)}</Badge>
              </div>
              <div className="flex shrink-0 flex-wrap items-center justify-end gap-1.5">
                <Badge variant="neutral" appearance="outline">
                  {formatCount(part.attachments.length, "attachment")}
                </Badge>
                <Badge variant="neutral" appearance="outline">{formatCount(part.links.length, "link")}</Badge>
              </div>
            </div>

            {part.warnings.length > 0 ? (
              <div className="mt-2 space-y-1">
                {part.warnings.map((warning) => (
                  <div
                    key={warning}
                    className="flex gap-1.5 text-xs text-warning-foreground"
                  >
                    <AlertTriangleIcon className="mt-0.5 size-3 shrink-0" />
                    <span>{warning}</span>
                  </div>
                ))}
              </div>
            ) : null}

            {documentCount > 0 ? (
              <div className="bg-background/60 mt-3 divide-y overflow-hidden rounded-md border">
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
              <p className="text-muted-foreground mt-3 text-xs">
                {t("No attachments or links in this part.")}
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
        <p className="text-muted-foreground text-xs">{label}</p>
      </div>
      <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
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
  const t = useT();

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
    <div className="border-border flex min-h-[20rem] flex-col rounded-md border p-4 lg:h-full lg:min-h-0">
      <h3 className="text-sm font-semibold">{t("Send History")}</h3>
      <ScrollArea className="mt-3 min-h-0 flex-1" viewportClassName="pr-2">
        {query.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </div>
        ) : query.isError ? (
          <p className="rounded-md border border-danger/30 bg-danger/10 p-3 text-sm text-danger-foreground">
            {t("Send history could not be loaded.")}
          </p>
        ) : attempts.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t("No email attempts recorded.")}</p>
        ) : (
          <div className="space-y-3">
            {attempts.map((attempt) => (
              <InvoiceSendHistoryCard key={attempt.id} attempt={attempt} />
            ))}
            {query.isFetchingNextPage ? (
              <div className="flex items-center justify-center py-4">
                <TextShimmer className="font-mono text-sm" duration={1}>
                  {t("Loading more...")}
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
  const t = useT();

  const status = invoiceAttemptDisplayStatus(attempt);
  const error = invoiceAttemptDisplayError(attempt);
  const sentAt = attempt.email?.sentAt ?? attempt.sentAt;
  const failedAt = attempt.email?.failedAt;
  const providerMessageId = attempt.email?.providerMessageId ?? attempt.providerMessageId;

  return (
    <div className="bg-muted/20 rounded-md border p-3">
      <div className="flex items-center justify-between gap-2">
        <Badge variant={SEND_STATUS_VARIANTS[status]}>{status}</Badge>
        <span className="text-muted-foreground text-xs">
          {t("Part {0} of {1}", attempt.partNumber, attempt.totalParts)}
        </span>
      </div>
      <p className="mt-2 truncate text-sm font-medium">{attempt.subject}</p>
      <p className="text-muted-foreground mt-1 text-xs">
        {sentAt
          ? formatUnixDate(sentAt)
          : failedAt
            ? t("Failed {0}", formatUnixDate(failedAt))
            : t("Not sent")}
      </p>
      {providerMessageId ? (
        <p className="text-muted-foreground mt-1 truncate text-xs">
          {t("Provider ID: {0}", providerMessageId)}
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
  if (invoice.status === "Voided") {
    return "A voided invoice is never sent.";
  }
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
          ? "mt-3 flex gap-2 rounded-md border border-danger/30 bg-danger/10 p-2 text-sm text-danger-foreground"
          : "mt-3 flex gap-2 rounded-md border border-warning/30 bg-warning/10 p-2 text-sm text-warning-foreground"
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
      <p className="text-muted-foreground text-xs">{label}</p>
      <p className="text-sm font-medium">{value}</p>
    </div>
  );
}

const SEND_STATUS_VARIANTS: Record<InvoiceSendStatus, BadgeVariant> = {
  NotSent: "neutral",
  Sending: "info",
  Sent: "success",
  PartiallySent: "warning",
  Failed: "danger",
};
