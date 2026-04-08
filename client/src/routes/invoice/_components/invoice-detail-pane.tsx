import AuditTab from "@/components/audit-tab";
import { EmptyState } from "@/components/empty-state";
import { PlainInvoiceStatusBadge } from "@/components/status-badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { queries } from "@/lib/queries";
import { getDestinationLocation, getOriginLocation } from "@/lib/shipment-utils";
import { cn, formatCurrency } from "@/lib/utils";
import { buttonVariants } from "@/lib/variants/button";
import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CircleDollarSignIcon,
  ClipboardListIcon,
  Clock3Icon,
  FileTextIcon,
  PackageCheckIcon,
  ReceiptTextIcon,
  SendIcon,
  TimerResetIcon,
  TruckIcon,
} from "lucide-react";
import { lazy } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { BillingQueueDocumentsTab } from "../../billing-queue/_components/billing-queue-documents-tab";

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
  const queryClient = useQueryClient();

  const { data: invoice, isLoading } = useQuery({
    ...queries.invoice.get(selectedInvoiceId ?? ""),
    enabled: !!selectedInvoiceId,
  });

  const { mutate: postInvoice, isPending: isPosting } = useMutation({
    mutationFn: (invoiceId: string) => apiService.invoiceService.post(invoiceId),
    onSuccess: (updated) => {
      void queryClient.invalidateQueries({ queryKey: ["invoice"] });
      void queryClient.invalidateQueries({ queryKey: ["invoice-list"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      toast.success(`${updated.number} posted`);
    },
    onError: () => {
      toast.error("Failed to post invoice");
    },
  });

  if (!selectedInvoiceId) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <EmptyState
          title="No invoice selected"
          description="Select an invoice to review the posted billing record, shipment references, and receivable details."
          icons={[ReceiptTextIcon, FileTextIcon, PackageCheckIcon]}
          className="max-w-xl border-none p-8 shadow-none"
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
  const lineCount = invoice.lines.length;
  const totalAmount = Number(invoice.totalAmount ?? 0);
  const originLocation = shipment ? getOriginLocation(shipment) : null;
  const destinationLocation = shipment ? getDestinationLocation(shipment) : null;
  const customerName = customer?.name ?? invoice.billToName;
  const topSummary = [
    {
      label: "Invoice Date",
      value: formatUnixDate(invoice.invoiceDate),
      icon: <ReceiptTextIcon className="size-4" />,
    },
    {
      label: "Due Date",
      value: formatUnixDate(invoice.dueDate),
      icon: <Clock3Icon className="size-4" />,
    },
    {
      label: "Posted At",
      value: invoice.status === "Posted" ? formatUnixDateTime(invoice.postedAt) : "Not posted",
      icon: <SendIcon className="size-4" />,
    },
    {
      label: "Charge Lines",
      value: String(lineCount),
      icon: <ClipboardListIcon className="size-4" />,
    },
  ];

  return (
    <div className="flex h-full flex-col">
      <div className="shrink-0 border-b bg-gradient-to-b from-muted/50 to-background px-4 py-4">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-xl font-semibold tracking-tight">{invoice.number}</h2>
                <PlainInvoiceStatusBadge status={invoice.status} />
                <span className="rounded-full border bg-background px-2.5 py-1 font-mono text-[10px] tracking-[0.18em] text-muted-foreground uppercase">
                  {invoice.billType}
                </span>
              </div>

              <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
                <span>{customerName}</span>
                {invoice.shipmentProNumber ? (
                  <>
                    <Separator orientation="vertical" className="h-5" />
                    <span>PRO: {invoice.shipmentProNumber}</span>
                  </>
                ) : null}
                {invoice.shipmentBol ? (
                  <>
                    <Separator orientation="vertical" className="h-5" />
                    <span>PRO: {invoice.shipmentProNumber}</span>
                    <span>BOL: {invoice.shipmentBol}</span>
                  </>
                ) : null}
                <Separator orientation="vertical" className="h-5" />
                <span>Terms: {invoice.paymentTerm}</span>
              </div>

              {originLocation && destinationLocation ? (
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <TruckIcon className="size-3.5" />
                  <span>
                    {originLocation.city}, {originLocation.state?.abbreviation}
                  </span>
                  <span className="text-muted-foreground/50">→</span>
                  <span>
                    {destinationLocation.city}, {destinationLocation.state?.abbreviation}
                  </span>
                </div>
              ) : null}
            </div>

            <div className="rounded-2xl border border-border bg-background/90 px-4 py-3">
              <p className="text-[11px] tracking-[0.16em] text-muted-foreground uppercase">
                Invoice Total
              </p>
              <p className="mt-1 text-2xl font-semibold tabular-nums">
                {formatCurrency(totalAmount, invoice.currencyCode)}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                Historical receivable generated from billing review
              </p>
            </div>
          </div>

          <StatusMessage
            status={invoice.status}
            invoiceDate={invoice.invoiceDate}
            postedAt={invoice.postedAt}
          />

          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            {topSummary.map((item) => (
              <SummaryTile
                key={item.label}
                icon={item.icon}
                label={item.label}
                value={item.value}
              />
            ))}
          </div>
        </div>
      </div>

      {shipment?.moves && shipment.moves.length > 0 ? (
        <div className="h-32 w-full shrink-0 border-b">
          <ShipmentRouteMap moves={shipment.moves} containerClassName="rounded-none border-b" />
        </div>
      ) : null}

      <div className="flex flex-wrap items-center gap-2 border-b px-4 py-2">
        <Button
          size="sm"
          onClick={() => postInvoice(invoice.id)}
          disabled={invoice.status === "Posted" || isPosting}
        >
          <SendIcon className="size-3.5" />
          {invoice.status === "Posted" ? "Posted" : "Post Invoice"}
        </Button>
        <Link
          to={`/shipment-management/shipments?item=${invoice.shipmentId}`}
          className={buttonVariants({ variant: "outline", size: "sm" })}
        >
          <PackageCheckIcon className="size-3.5" />
          View Shipment
        </Link>
        <Link
          to={`/billing/queue?item=${invoice.billingQueueItemId}&includePosted=true`}
          className={buttonVariants({ variant: "outline", size: "sm" })}
        >
          <FileTextIcon className="size-3.5" />
          View Billing Queue Item
        </Link>
      </div>

      <Tabs defaultValue="overview" className="flex min-h-0 flex-1 flex-col">
        <TabsList variant="underline" className="w-full border-b border-border">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="charges">Charges</TabsTrigger>
          <TabsTrigger value="documents">Documents</TabsTrigger>
          <TabsTrigger value="activity">Activity</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="grid gap-4 p-4 xl:grid-cols-[1.15fr_0.85fr]">
              <div className="space-y-4">
                <Card className="border border-border shadow-none">
                  <CardHeader className="border-b">
                    <CardTitle>Invoice Summary</CardTitle>
                    <CardDescription>
                      Snapshot of the receivable document as it exists after billing review.
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="grid gap-3 pt-4">
                    <DataRow label="Invoice Number" value={invoice.number} mono />
                    <DataRow label="Invoice Status" value={invoice.status} />
                    <DataRow label="Bill Type" value={invoice.billType} />
                    <DataRow label="Invoice Date" value={formatUnixDate(invoice.invoiceDate)} />
                    <DataRow label="Due Date" value={formatUnixDate(invoice.dueDate)} />
                    <DataRow label="Service Date" value={formatUnixDate(invoice.serviceDate)} />
                    <DataRow label="Payment Terms" value={invoice.paymentTerm} />
                    <DataRow label="Currency" value={invoice.currencyCode} />
                    <DataRow label="Posted At" value={formatUnixDateTime(invoice.postedAt)} />
                  </CardContent>
                </Card>

                <Card className="border border-border shadow-none">
                  <CardHeader className="border-b">
                    <CardTitle>Shipment and Source Context</CardTitle>
                    <CardDescription>
                      Operational references used to create this invoice from the billing queue.
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="grid gap-3 pt-4">
                    <DataRow label="Billing Queue Item" value={invoice.billingQueueItemId} mono />
                    <DataRow label="Shipment ID" value={invoice.shipmentId} mono />
                    <DataRow label="Shipment PRO" value={invoice.shipmentProNumber || "N/A"} />
                    <DataRow label="Shipment BOL" value={invoice.shipmentBol || "N/A"} />
                    <DataRow label="Customer" value={customerName || "Unknown customer"} />
                    <DataRow
                      label="Customer Code"
                      value={customer?.code || invoice.billToCode || "N/A"}
                    />
                  </CardContent>
                </Card>

                <Card className="border border-border shadow-none">
                  <CardHeader className="border-b">
                    <CardTitle>Bill-To Snapshot</CardTitle>
                    <CardDescription>
                      Historical billing destination captured at invoice creation.
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="grid gap-3 pt-4">
                    <DataRow label="Bill-To Name" value={invoice.billToName} />
                    <DataRow label="Bill-To Code" value={invoice.billToCode || "N/A"} />
                    <DataRow
                      label="Billing Address"
                      value={formatAddress([
                        invoice.billToAddressLine1,
                        invoice.billToAddressLine2,
                        invoice.billToCity,
                        invoice.billToState,
                        invoice.billToPostalCode,
                        invoice.billToCountry,
                      ])}
                    />
                  </CardContent>
                </Card>
              </div>

              <div className="space-y-4">
                <Card className="border border-border shadow-none">
                  <CardHeader className="border-b">
                    <CardTitle>Charge Summary</CardTitle>
                    <CardDescription>
                      Final charge values retained on the invoice record.
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3 pt-4">
                    <AmountRow
                      label="Freight Charges"
                      value={formatCurrency(
                        Number(invoice.subtotalAmount ?? 0),
                        invoice.currencyCode,
                      )}
                    />
                    <AmountRow
                      label="Other Charges"
                      value={formatCurrency(Number(invoice.otherAmount ?? 0), invoice.currencyCode)}
                    />
                    <Separator />
                    <AmountRow
                      label="Invoice Total"
                      value={formatCurrency(Number(invoice.totalAmount ?? 0), invoice.currencyCode)}
                      emphasized
                    />
                  </CardContent>
                </Card>

                <Card className="border border-border shadow-none">
                  <CardHeader className="border-b">
                    <CardTitle>Document State</CardTitle>
                    <CardDescription>
                      Where this invoice sits in the receivable lifecycle.
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3 pt-4">
                    <LifecycleRow
                      label="Generated from Billing Queue"
                      active
                      timestamp={formatUnixDateTime(invoice.createdAt)}
                    />
                    <LifecycleRow
                      label="Ready for Posting"
                      active
                      timestamp={formatUnixDate(invoice.invoiceDate)}
                    />
                    <LifecycleRow
                      label="Posted to Invoice History"
                      active={invoice.status === "Posted"}
                      timestamp={formatUnixDateTime(invoice.postedAt)}
                    />
                  </CardContent>
                </Card>

                <Card className="border border-border shadow-none">
                  <CardHeader className="border-b">
                    <CardTitle>Quick Context</CardTitle>
                    <CardDescription>
                      Fast reference details for support, billing, and dispute review.
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="grid gap-3 pt-4">
                    <MetricPill
                      label="Lines"
                      value={String(lineCount)}
                      icon={<ClipboardListIcon className="size-4" />}
                    />
                    <MetricPill
                      label="Status"
                      value={invoice.status}
                      icon={<TimerResetIcon className="size-4" />}
                    />
                    <MetricPill
                      label="Terms"
                      value={invoice.paymentTerm}
                      icon={<Clock3Icon className="size-4" />}
                    />
                    <MetricPill
                      label="Bill Type"
                      value={invoice.billType}
                      icon={<ReceiptTextIcon className="size-4" />}
                    />
                  </CardContent>
                </Card>
              </div>
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent value="charges" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <div className="space-y-4 p-4">
              <Card className="border border-border shadow-none">
                <CardHeader className="border-b">
                  <CardTitle>Charge Detail</CardTitle>
                  <CardDescription>
                    Final freight and accessorial charges retained as invoice lines.
                  </CardDescription>
                </CardHeader>
                <CardContent className="pt-4">
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
                          <tr key={line.id} className="border-t">
                            <td className="px-4 py-3 font-mono text-xs">{line.lineNumber}</td>
                            <td className="px-4 py-3">{line.description}</td>
                            <td className="px-4 py-3">
                              <span className="rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground">
                                {line.type}
                              </span>
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
                    </table>
                  </div>
                </CardContent>
              </Card>
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

function StatusMessage({
  status,
  invoiceDate,
  postedAt,
}: {
  status: "Draft" | "Posted";
  invoiceDate: number;
  postedAt: number | null | undefined;
}) {
  if (status === "Posted") {
    return (
      <Alert variant="success">
        <SendIcon className="size-4" />
        <AlertTitle>Posted Invoice</AlertTitle>
        <AlertDescription>
          This invoice was posted on {formatUnixDateTime(postedAt)} and now represents a historical
          receivable record.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Alert variant="warning">
      <AlertTriangleIcon className="size-4" />
      <AlertTitle>Ready to Post</AlertTitle>
      <AlertDescription>
        This invoice was created from billing review on {formatUnixDate(invoiceDate)} and is still
        in draft status. Posting will finalize the invoice and move the source queue item into
        posted history.
      </AlertDescription>
    </Alert>
  );
}

function SummaryTile({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-xl border border-border bg-background/85 px-3 py-3">
      <div className="flex items-center gap-2 text-[11px] tracking-[0.16em] text-muted-foreground uppercase">
        {icon}
        <span>{label}</span>
      </div>
      <p className="mt-2 text-sm font-medium text-foreground">{value}</p>
    </div>
  );
}

function DataRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="grid grid-cols-[140px_1fr] gap-4 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className={cn(mono && "font-mono text-xs")}>{value}</span>
    </div>
  );
}

function AmountRow({
  label,
  value,
  emphasized = false,
}: {
  label: string;
  value: string;
  emphasized?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex items-center gap-2 text-muted-foreground">
        <CircleDollarSignIcon className="size-4" />
        <span>{label}</span>
      </div>
      <span className={cn("tabular-nums", emphasized && "font-semibold")}>{value}</span>
    </div>
  );
}

function LifecycleRow({
  label,
  active,
  timestamp,
}: {
  label: string;
  active: boolean;
  timestamp: string;
}) {
  return (
    <div className="flex items-start gap-3">
      <div
        className={cn(
          "mt-0.5 size-2.5 rounded-full",
          active ? "bg-success" : "bg-muted-foreground/30",
        )}
      />
      <div className="space-y-0.5">
        <p className={cn("text-sm", !active && "text-muted-foreground")}>{label}</p>
        <p className="text-xs text-muted-foreground">{timestamp}</p>
      </div>
    </div>
  );
}

function MetricPill({
  label,
  value,
  icon,
}: {
  label: string;
  value: string;
  icon: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between rounded-xl border border-border bg-muted/30 px-3 py-2">
      <div className="flex items-center gap-2 text-muted-foreground">
        {icon}
        <span className="text-sm">{label}</span>
      </div>
      <span className="text-sm font-medium">{value}</span>
    </div>
  );
}

function formatUnixDate(value: number | null | undefined) {
  if (!value) return "N/A";
  return new Date(value * 1000).toLocaleDateString();
}

function formatUnixDateTime(value: number | null | undefined) {
  if (!value) return "N/A";
  return new Date(value * 1000).toLocaleString();
}

function formatAddress(parts: Array<string | null | undefined>) {
  const values = parts.filter(Boolean);
  return values.length > 0 ? values.join(", ") : "N/A";
}
