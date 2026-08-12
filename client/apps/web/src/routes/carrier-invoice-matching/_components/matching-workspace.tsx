import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { EmptyState } from "@/components/empty-state";
import { CarrierInvoiceMatchStatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  acceptCarrierInvoiceMatch,
  acceptCarrierInvoiceMatchWithVariance,
  createCarrierInvoiceMatch,
  fetchCarrierInvoiceMatches,
  fetchCarrierSettlementControl,
  fetchEdiCarrierInvoices,
  rejectCarrierInvoiceMatch,
  suggestCarrierForEdiInvoice,
  type CarrierInvoiceMatchRow,
  type EdiCarrierInvoiceRow,
} from "@/lib/graphql/carrier-settlement";
import { carrierInvoiceMatchStatusChoices } from "@/lib/choices";
import type { CarrierInvoiceMatchVia } from "@trenova/shared/types/carrier-settlement";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { formatSettlementDate } from "@trenova/shared/lib/date";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Building2Icon,
  CheckCheckIcon,
  FileTextIcon,
  LinkIcon,
  ReceiptTextIcon,
  ScaleIcon,
  SparklesIcon,
  XIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { LinkCarrierDialog } from "./link-carrier-dialog";

type QueueTab = "invoices" | "matches";

type InvoiceFilter = "attention" | "all";

type MatchFilter = "open" | "Suggested" | "Matched" | "Variance" | "Resolved" | "Rejected";

type MatchViaFilter = "all" | CarrierInvoiceMatchVia;

const OPEN_MATCH_STATUSES = new Set(["Suggested", "Matched", "Variance"]);

const matchFilterChips: ReadonlyArray<{ value: MatchFilter; label: string }> = [
  { value: "open", label: "Open" },
  ...carrierInvoiceMatchStatusChoices.map((choice) => ({
    value: choice.value as MatchFilter,
    label: choice.label,
  })),
];

const matchViaFilterChips: ReadonlyArray<{ value: MatchViaFilter; label: string }> = [
  { value: "all", label: "All" },
  { value: "Auto", label: "Auto" },
  { value: "Manual", label: "Manual" },
];

/**
 * A resolution with a timestamp but no actor can only have come from the
 * auto-accept sweep: every dispatcher-driven resolution passes through the
 * service's actor guard, which refuses a nil user, while the sweep resolves
 * with no user at all. The resolution note is deliberately not consulted —
 * free text is not a provenance signal.
 */
function wasAutoAccepted(match: CarrierInvoiceMatchRow): boolean {
  return match.resolvedAt != null && !match.resolvedById;
}

function invoiceNeedsAttention(invoice: EdiCarrierInvoiceRow): boolean {
  return (
    invoice.reconciliationStatus === "Unmatched" ||
    invoice.reconciliationStatus === "MappingRequired" ||
    invoice.reconciliationStatus === "Variance"
  );
}

function invalidateMatchingQueries(queryClient: ReturnType<typeof useQueryClient>) {
  for (const key of [
    "carrier-invoice-matching-invoices",
    "carrier-invoice-matching-matches",
    "carrier-settlement-invoice-matches",
    "carrier-cost-event-list",
  ]) {
    void queryClient.invalidateQueries({ queryKey: [key] });
  }
}

export default function MatchingWorkspace() {
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<QueueTab>("invoices");
  const [invoiceFilter, setInvoiceFilter] = useState<InvoiceFilter>("attention");
  const [matchFilter, setMatchFilter] = useState<MatchFilter>("open");
  const [matchViaFilter, setMatchViaFilter] = useState<MatchViaFilter>("all");
  const [search, setSearch] = useState("");
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<string | null>(null);
  const [selectedMatchId, setSelectedMatchId] = useState<string | null>(null);

  const invoicesQuery = useQuery({
    queryKey: ["carrier-invoice-matching-invoices"],
    queryFn: () => fetchEdiCarrierInvoices({ limit: 200 }),
  });

  const matchesQuery = useQuery({
    queryKey: ["carrier-invoice-matching-matches"],
    queryFn: () => fetchCarrierInvoiceMatches({ limit: 200 }),
  });

  const controlQuery = useQuery({
    queryKey: ["carrier-settlement-control"],
    queryFn: fetchCarrierSettlementControl,
  });

  const refresh = () => invalidateMatchingQueries(queryClient);

  const invoices = useMemo(() => {
    const term = search.trim().toLowerCase();
    return (invoicesQuery.data?.items ?? []).filter((invoice) => {
      if (invoiceFilter === "attention" && !invoiceNeedsAttention(invoice)) return false;
      if (!term) return true;
      return (
        invoice.invoiceNumber.toLowerCase().includes(term) ||
        invoice.proNumber.toLowerCase().includes(term) ||
        invoice.billToName.toLowerCase().includes(term) ||
        invoice.shipmentReference.toLowerCase().includes(term)
      );
    });
  }, [invoicesQuery.data, invoiceFilter, search]);

  const matches = useMemo(() => {
    const term = search.trim().toLowerCase();
    return (matchesQuery.data?.items ?? []).filter((match) => {
      if (matchFilter === "open" && !OPEN_MATCH_STATUSES.has(match.status)) return false;
      if (matchFilter !== "open" && match.status !== matchFilter) return false;
      if (matchViaFilter !== "all" && match.matchedVia !== matchViaFilter) return false;
      if (!term) return true;
      return (
        match.invoiceNumber.toLowerCase().includes(term) ||
        (match.carrier?.name ?? "").toLowerCase().includes(term) ||
        (match.carrierAssignment?.proNumber ?? "").toLowerCase().includes(term)
      );
    });
  }, [matchesQuery.data, matchFilter, matchViaFilter, search]);

  const selectedInvoice =
    invoices.find((invoice) => invoice.id === selectedInvoiceId) ?? invoices[0] ?? null;
  const selectedMatch = matches.find((match) => match.id === selectedMatchId) ?? matches[0] ?? null;

  const allInvoices = invoicesQuery.data?.items ?? [];
  const allMatches = matchesQuery.data?.items ?? [];
  const attentionInvoiceCount = allInvoices.filter(invoiceNeedsAttention).length;
  const varianceCount = allMatches.filter((match) => match.status === "Variance").length;
  const suggestedCount = allMatches.filter((match) => match.status === "Suggested").length;
  const resolvedCount = allMatches.filter((match) => match.status === "Resolved").length;

  const settlementControl = controlQuery.data ?? null;

  return (
    <div className="flex flex-col gap-3">
      {settlementControl && (
        <p
          data-testid="carrier-match-automation-status"
          className="flex flex-wrap items-center gap-x-1.5 text-[11px] text-muted-foreground"
        >
          <span>Auto-match: {settlementControl.autoMatchInboundInvoices ? "On" : "Off"}</span>
          <span aria-hidden>·</span>
          <span>
            Auto-accept within tolerance:{" "}
            {settlementControl.autoAcceptWithinTolerance ? "On" : "Off"}
          </span>
          <Link
            to="/admin/carrier-settlement-control"
            className="font-medium text-foreground underline underline-offset-2"
          >
            Automation settings
          </Link>
        </p>
      )}
      <div className="grid gap-3 md:grid-cols-4">
        <SummaryCard label="Invoices Needing Attention" value={String(attentionInvoiceCount)} />
        <SummaryCard label="Variance Matches" value={String(varianceCount)} />
        <SummaryCard label="Suggested Matches" value={String(suggestedCount)} />
        <SummaryCard label="Resolved" value={String(resolvedCount)} />
      </div>
      <div className="grid h-[calc(100vh-260px)] min-h-120 gap-0 overflow-hidden rounded-lg border md:grid-cols-[340px_1fr]">
        <div className="flex h-full min-h-0 flex-col overflow-hidden border-r">
          <div className="flex flex-col gap-2 border-b p-2">
            <div className="flex gap-1">
              <TabChip
                active={tab === "invoices"}
                label={`Carrier Invoices (${invoices.length})`}
                onClick={() => setTab("invoices")}
              />
              <TabChip
                active={tab === "matches"}
                label={`Matches (${matches.length})`}
                onClick={() => setTab("matches")}
              />
            </div>
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search invoice, pro number, carrier..."
              className="h-8 text-xs"
            />
            {tab === "invoices" ? (
              <div className="flex flex-wrap gap-1">
                {(
                  [
                    { value: "attention", label: "Needs Attention" },
                    { value: "all", label: "All" },
                  ] as Array<{ value: InvoiceFilter; label: string }>
                ).map((chip) => (
                  <FilterChip
                    key={chip.value}
                    active={invoiceFilter === chip.value}
                    label={chip.label}
                    onClick={() => setInvoiceFilter(chip.value)}
                  />
                ))}
              </div>
            ) : (
              <div className="flex flex-col gap-1">
                <div className="flex flex-wrap gap-1">
                  {matchFilterChips.map((chip) => (
                    <FilterChip
                      key={chip.value}
                      active={matchFilter === chip.value}
                      label={chip.label}
                      onClick={() => setMatchFilter(chip.value)}
                    />
                  ))}
                </div>
                <div className="flex flex-wrap items-center gap-1">
                  <span className="text-[10px] tracking-wide text-muted-foreground uppercase">
                    Created
                  </span>
                  {matchViaFilterChips.map((chip) => (
                    <FilterChip
                      key={chip.value}
                      active={matchViaFilter === chip.value}
                      label={chip.label}
                      onClick={() => setMatchViaFilter(chip.value)}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>
          <ScrollArea className="min-h-0 flex-1" viewportClassName="min-h-0">
            {tab === "invoices" ? (
              <InvoiceList
                invoices={invoices}
                loading={invoicesQuery.isLoading}
                selectedId={selectedInvoice?.id ?? null}
                onSelect={setSelectedInvoiceId}
              />
            ) : (
              <MatchList
                matches={matches}
                loading={matchesQuery.isLoading}
                selectedId={selectedMatch?.id ?? null}
                onSelect={setSelectedMatchId}
              />
            )}
          </ScrollArea>
        </div>
        <ScrollArea className="h-full min-h-0" viewportClassName="min-h-0">
          {tab === "invoices" ? (
            selectedInvoice ? (
              <InvoiceDetail
                key={selectedInvoice.id}
                invoice={selectedInvoice}
                onChanged={refresh}
              />
            ) : (
              <DetailEmptyState
                title="No invoice selected"
                description="Select a carrier freight invoice to link it to a carrier and create a match."
              />
            )
          ) : selectedMatch ? (
            <MatchDetail
              match={selectedMatch}
              varianceToleranceMinor={controlQuery.data?.varianceToleranceMinor ?? null}
              onChanged={refresh}
            />
          ) : (
            <DetailEmptyState
              title="No match selected"
              description="Select a match to compare the invoice against the negotiated buy rate."
            />
          )}
        </ScrollArea>
      </div>
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <p className="text-[11px] tracking-[0.16em] text-muted-foreground uppercase">{label}</p>
      <p className="mt-1 text-2xl font-semibold tabular-nums">{value}</p>
    </div>
  );
}

function TabChip({
  active,
  label,
  onClick,
}: {
  active: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex-1 rounded-md border px-2 py-1 text-[11px] font-medium transition-colors",
        active ? "border-primary bg-primary text-primary-foreground" : "hover:bg-muted",
      )}
    >
      {label}
    </button>
  );
}

function FilterChip({
  active,
  label,
  onClick,
}: {
  active: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors",
        active
          ? "border-primary bg-primary text-primary-foreground"
          : "text-muted-foreground hover:bg-muted",
      )}
    >
      {label}
    </button>
  );
}

function DetailEmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <EmptyState
        title={title}
        description={description}
        icons={[ReceiptTextIcon, ScaleIcon, Building2Icon]}
        className="border-none shadow-none"
      />
    </div>
  );
}

function InvoiceList({
  invoices,
  loading,
  selectedId,
  onSelect,
}: {
  invoices: EdiCarrierInvoiceRow[];
  loading: boolean;
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  if (loading) {
    return (
      <div className="flex flex-col gap-1.5 p-2">
        {Array.from({ length: 5 }, (_, index) => (
          <Skeleton key={index} className="h-20 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (invoices.length === 0) {
    return (
      <p className="p-4 text-center text-xs text-muted-foreground">
        No carrier invoices match this view.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-1.5 p-2">
      {invoices.map((invoice) => (
        <button
          key={invoice.id}
          type="button"
          onClick={() => onSelect(invoice.id)}
          className={cn(
            "rounded-lg border p-2.5 text-left transition-colors",
            invoice.id === selectedId ? "border-primary bg-primary/5" : "hover:bg-muted/40",
          )}
        >
          <div className="flex items-center justify-between gap-2">
            <span className="font-mono text-xs font-medium">
              {invoice.invoiceNumber || "No invoice #"}
            </span>
            <span className="rounded-full border px-2 py-0.5 text-[10px] uppercase">
              {invoice.reconciliationStatus}
            </span>
          </div>
          <p className="mt-1 text-[11px] text-muted-foreground">
            {invoice.proNumber ? `PRO ${invoice.proNumber} · ` : ""}
            {invoice.billToName || "Unknown bill-to"}
          </p>
          <div className="mt-1 flex items-center justify-between text-[11px]">
            <span
              className={cn(
                "inline-flex items-center gap-1",
                invoice.carrierId
                  ? "text-green-700 dark:text-green-400"
                  : "text-amber-700 dark:text-amber-400",
              )}
            >
              <Building2Icon className="size-3" aria-hidden />
              {invoice.carrierId ? "Carrier linked" : "No carrier link"}
            </span>
            <span className="font-semibold tabular-nums">
              {invoice.totalAmount != null
                ? formatCurrency(Number(invoice.totalAmount), invoice.currencyCode || "USD")
                : "—"}
            </span>
          </div>
        </button>
      ))}
    </div>
  );
}

function MatchList({
  matches,
  loading,
  selectedId,
  onSelect,
}: {
  matches: CarrierInvoiceMatchRow[];
  loading: boolean;
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  if (loading) {
    return (
      <div className="flex flex-col gap-1.5 p-2">
        {Array.from({ length: 5 }, (_, index) => (
          <Skeleton key={index} className="h-20 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (matches.length === 0) {
    return (
      <p className="p-4 text-center text-xs text-muted-foreground">
        No matches in this view. Create one from an unmatched carrier invoice.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-1.5 p-2" data-testid="carrier-match-list">
      {matches.map((match) => (
        <button
          key={match.id}
          type="button"
          onClick={() => onSelect(match.id)}
          className={cn(
            "rounded-lg border p-2.5 text-left transition-colors",
            match.id === selectedId ? "border-primary bg-primary/5" : "hover:bg-muted/40",
          )}
        >
          <div className="flex items-center justify-between gap-2">
            <span className="font-mono text-xs font-medium">
              {match.invoiceNumber || "No invoice #"}
            </span>
            <div className="flex shrink-0 items-center gap-1">
              {match.matchedVia === "Auto" && (
                <Badge
                  variant="info"
                  className="h-4 px-1 text-[9px]"
                  title="Created by the inbound EDI 210 auto-match sweep"
                >
                  Auto-matched
                </Badge>
              )}
              {wasAutoAccepted(match) && (
                <Badge
                  variant="active"
                  className="h-4 px-1 text-[9px]"
                  title="Resolved automatically because the variance was within tolerance"
                >
                  Auto-accepted
                </Badge>
              )}
              <CarrierInvoiceMatchStatusBadge status={match.status} />
            </div>
          </div>
          <p className="mt-1 truncate text-[11px] text-muted-foreground">
            {match.carrier?.name ?? "Unknown carrier"}
            {match.carrierAssignment?.proNumber
              ? ` · PRO ${match.carrierAssignment.proNumber}`
              : ""}
          </p>
          <div className="mt-1 flex items-center justify-between text-[11px]">
            <span className="text-muted-foreground">
              billed <AmountDisplay value={match.invoiceTotalMinor} currency={match.currencyCode} />
            </span>
            {match.varianceMinor !== 0 && (
              <span className="font-medium text-amber-700 tabular-nums dark:text-amber-400">
                Δ <AmountDisplay value={match.varianceMinor} currency={match.currencyCode} />
              </span>
            )}
          </div>
        </button>
      ))}
    </div>
  );
}

function InvoiceDetail({
  invoice,
  onChanged,
}: {
  invoice: EdiCarrierInvoiceRow;
  onChanged: () => void;
}) {
  const [linkOpen, setLinkOpen] = useState(false);
  const [suggestedCarrierId, setSuggestedCarrierId] = useState<string | null>(null);

  const suggestMutation = useMutation({
    mutationFn: () => suggestCarrierForEdiInvoice(invoice.id),
    onSuccess: (carrier) => {
      if (!carrier) {
        toast.info("No carrier in the master matches this invoice's SCAC or DOT number.");
        return;
      }
      setSuggestedCarrierId(carrier.id);
      toast.success(
        `Suggested carrier: ${carrier.name}${carrier.scac ? ` (${carrier.scac})` : ""} — confirm the link to proceed`,
      );
      setLinkOpen(true);
    },
    onError: (error: Error) => toast.error(error.message || "Failed to suggest a carrier"),
  });

  const createMatchMutation = useMutation({
    mutationFn: () => createCarrierInvoiceMatch({ ediCarrierInvoiceId: invoice.id }),
    onSuccess: (match) => {
      toast.success(
        match.status === "Variance"
          ? "Match created with a variance — resolve it from the Matches tab"
          : "Match created",
      );
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to create match"),
  });

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="rounded-lg border">
        <div className="border-b px-4 py-3">
          <h3 className="text-sm font-semibold">Invoice {invoice.invoiceNumber || invoice.id}</h3>
          <p className="text-xs text-muted-foreground">
            EDI 210 carrier freight invoice · received {formatSettlementDate(invoice.createdAt)}
          </p>
        </div>
        <div className="grid gap-3 p-4 md:grid-cols-2">
          <Metric label="Reconciliation Status" value={invoice.reconciliationStatus} />
          <Metric
            label="Invoice Total"
            value={
              invoice.totalAmount != null
                ? formatCurrency(Number(invoice.totalAmount), invoice.currencyCode || "USD")
                : "—"
            }
          />
          <Metric label="Invoice Date" value={formatSettlementDate(invoice.invoiceDate)} />
          <Metric label="Delivery Date" value={formatSettlementDate(invoice.deliveryDate)} />
          <Metric label="Pro Number" value={invoice.proNumber || "—"} />
          <Metric label="BOL" value={invoice.bol || "—"} />
          <Metric label="Shipment Reference" value={invoice.shipmentReference || "—"} />
          <Metric label="Bill To" value={invoice.billToName || "—"} />
          {invoice.expectedAmount != null && (
            <Metric
              label="Expected Amount"
              value={formatCurrency(Number(invoice.expectedAmount), invoice.currencyCode || "USD")}
            />
          )}
          {invoice.varianceAmount != null && (
            <Metric
              label="Variance"
              value={formatCurrency(Number(invoice.varianceAmount), invoice.currencyCode || "USD")}
            />
          )}
        </div>
      </div>

      <div className="rounded-lg border p-4">
        <h4 className="mb-1 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
          Carrier Link
        </h4>
        <p className="mb-2 text-[11px] text-muted-foreground">
          {invoice.carrierId
            ? "This invoice is linked to a carrier in the master and can be matched against its assignments."
            : "Link the invoice to a carrier in the master before creating a match — suggest looks it up by SCAC and DOT number."}
        </p>
        <div className="flex flex-wrap gap-2">
          {!invoice.carrierId && (
            <>
              <Button
                size="sm"
                variant="outline"
                disabled={suggestMutation.isPending}
                onClick={() => suggestMutation.mutate()}
              >
                <SparklesIcon className="size-3.5" />
                Suggest Carrier
              </Button>
              <Button size="sm" variant="outline" onClick={() => setLinkOpen(true)}>
                <LinkIcon className="size-3.5" />
                Link Carrier
              </Button>
            </>
          )}
          <Button
            size="sm"
            disabled={!invoice.carrierId || createMatchMutation.isPending}
            title={
              invoice.carrierId
                ? "Pair this invoice with the carrier assignment resolved by pro number or shipment reference"
                : "Link a carrier first"
            }
            onClick={() => createMatchMutation.mutate()}
          >
            <FileTextIcon className="size-3.5" />
            Create Match
          </Button>
        </div>
      </div>

      <LinkCarrierDialog
        open={linkOpen}
        invoiceId={invoice.id}
        invoiceNumber={invoice.invoiceNumber}
        suggestedCarrierId={suggestedCarrierId}
        onOpenChange={(open) => {
          setLinkOpen(open);
          if (!open) setSuggestedCarrierId(null);
        }}
        onLinked={onChanged}
      />
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-background px-3 py-2">
      <p className="text-[10px] tracking-[0.16em] text-muted-foreground uppercase">{label}</p>
      <p className="mt-1 text-sm font-medium">{value}</p>
    </div>
  );
}

function MatchDetail({
  match,
  varianceToleranceMinor,
  onChanged,
}: {
  match: CarrierInvoiceMatchRow;
  /** Null when the settlement control could not be loaded — tolerance is then unknown, not $0. */
  varianceToleranceMinor: number | null;
  onChanged: () => void;
}) {
  const [rejectOpen, setRejectOpen] = useState(false);
  const assignment = match.carrierAssignment;
  const currency = match.currencyCode || "USD";
  const withinTolerance =
    varianceToleranceMinor != null && Math.abs(match.varianceMinor) <= varianceToleranceMinor;
  const isOpen = OPEN_MATCH_STATUSES.has(match.status);

  const acceptMutation = useMutation({
    mutationFn: () => acceptCarrierInvoiceMatch({ matchId: match.id }),
    onSuccess: () => {
      toast.success("Match accepted — the invoice is reconciled");
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to accept match"),
  });

  const acceptWithVarianceMutation = useMutation({
    mutationFn: () => acceptCarrierInvoiceMatchWithVariance({ matchId: match.id }),
    onSuccess: () => {
      toast.success(
        "Match accepted with variance — an adjustment cost event was accrued for the difference",
      );
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to accept with variance"),
  });

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="flex flex-wrap items-center gap-2">
        <CarrierInvoiceMatchStatusBadge status={match.status} />
        <span className="text-xs font-medium">
          {match.carrier?.name ?? "Unknown carrier"}
          {match.carrier?.scac ? ` (${match.carrier.scac})` : ""}
        </span>
        <span className="ml-auto text-xs text-muted-foreground">
          created {formatSettlementDate(match.createdAt)}
        </span>
      </div>

      <div className="grid gap-3 lg:grid-cols-2">
        <div className="rounded-lg border">
          <div className="border-b bg-muted/30 px-4 py-2">
            <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
              Carrier Invoice
            </h4>
          </div>
          <div className="flex flex-col gap-2 p-4 text-xs">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Invoice number</span>
              <span className="font-mono font-medium">{match.invoiceNumber || "—"}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Source</span>
              <span className="font-medium">
                {match.ediCarrierInvoiceId ? "EDI 210" : "Document AI"}
              </span>
            </div>
            <div className="mt-2 flex justify-between border-t pt-2 text-sm">
              <span className="font-semibold">Invoice total</span>
              <span className="font-semibold">
                <AmountDisplay value={match.invoiceTotalMinor} currency={currency} />
              </span>
            </div>
          </div>
        </div>

        <div className="rounded-lg border">
          <div className="border-b bg-muted/30 px-4 py-2">
            <h4 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
              Negotiated Buy Rate
            </h4>
          </div>
          <div className="flex flex-col gap-2 p-4 text-xs">
            {assignment ? (
              <>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">
                    Base ({assignment.rateMethod === "PerMile" ? "per mile" : "flat"})
                  </span>
                  <span className="font-medium tabular-nums">
                    {formatCurrency(Number(assignment.baseAmount ?? 0), currency)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Fuel surcharge</span>
                  <span className="font-medium tabular-nums">
                    {formatCurrency(Number(assignment.fuelSurcharge ?? 0), currency)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Accessorials</span>
                  <span className="font-medium tabular-nums">
                    {formatCurrency(Number(assignment.accessorialTotal ?? 0), currency)}
                  </span>
                </div>
                {(assignment.accessorials ?? []).map((accessorial) => (
                  <div key={accessorial.id} className="flex justify-between pl-3">
                    <span className="truncate text-muted-foreground">
                      {accessorial.description}
                    </span>
                    <span className="tabular-nums">
                      {formatCurrency(Number(accessorial.amount ?? 0), currency)}
                    </span>
                  </div>
                ))}
                <div className="mt-2 flex justify-between border-t pt-2 text-sm">
                  <span className="font-semibold">Expected total</span>
                  <span className="font-semibold">
                    <AmountDisplay value={match.expectedTotalMinor} currency={currency} />
                  </span>
                </div>
              </>
            ) : (
              <p className="text-muted-foreground">
                The linked assignment is unavailable. Expected total:{" "}
                <AmountDisplay value={match.expectedTotalMinor} currency={currency} />
              </p>
            )}
          </div>
        </div>
      </div>

      <div
        className={cn(
          "rounded-lg border p-4",
          match.varianceMinor === 0 || withinTolerance
            ? "border-green-200 bg-green-50/50 dark:border-green-900 dark:bg-green-950/30"
            : "border-amber-200 bg-amber-50/60 dark:border-amber-900 dark:bg-amber-950/30",
        )}
      >
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <p className="text-xs font-semibold">
              Variance:{" "}
              <AmountDisplay value={match.varianceMinor} variant="auto" currency={currency} />
            </p>
            <p className="text-[11px] text-muted-foreground">
              {varianceToleranceMinor != null ? (
                <>
                  Invoice minus expected · tolerance{" "}
                  <AmountDisplay value={varianceToleranceMinor} currency={currency} /> —{" "}
                  {withinTolerance
                    ? "within tolerance"
                    : "beyond tolerance, accept with variance to accrue the difference"}
                </>
              ) : (
                "Invoice minus expected"
              )}
            </p>
          </div>
        </div>
      </div>

      {isOpen ? (
        <div className="flex flex-wrap items-center gap-2">
          {match.status !== "Variance" && (
            <Button
              size="sm"
              disabled={acceptMutation.isPending || acceptWithVarianceMutation.isPending}
              onClick={() => acceptMutation.mutate()}
              title="Reconciles the invoice against the buy rate without changing the accrued cost"
            >
              <CheckCheckIcon className="size-3.5" />
              Accept
            </Button>
          )}
          {match.status === "Variance" && (
            <Button
              size="sm"
              disabled={acceptMutation.isPending || acceptWithVarianceMutation.isPending}
              onClick={() => acceptWithVarianceMutation.mutate()}
              title="Accrues an adjustment cost event for the variance so the carrier is paid the billed amount"
            >
              <ScaleIcon className="size-3.5" />
              Accept with Variance (
              <AmountDisplay value={match.varianceMinor} variant="auto" currency={currency} />)
            </Button>
          )}
          <Button
            size="sm"
            variant="ghost"
            className="text-red-600 hover:text-red-700 dark:text-red-400"
            disabled={acceptMutation.isPending || acceptWithVarianceMutation.isPending}
            onClick={() => setRejectOpen(true)}
          >
            <XIcon className="size-3.5" />
            Reject
          </Button>
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">
          {match.status === "Resolved"
            ? `Resolved${match.resolvedAt ? ` ${formatSettlementDate(match.resolvedAt)}` : ""}${match.resolutionNote ? ` — ${match.resolutionNote}` : ""}`
            : `Rejected${match.resolutionNote ? ` — ${match.resolutionNote}` : ""}`}
        </p>
      )}

      <RejectMatchDialog
        open={rejectOpen}
        matchId={match.id}
        onOpenChange={setRejectOpen}
        onChanged={onChanged}
      />
    </div>
  );
}

function RejectMatchDialog({
  open,
  matchId,
  onOpenChange,
  onChanged,
}: {
  open: boolean;
  matchId: string;
  onOpenChange: (open: boolean) => void;
  onChanged: () => void;
}) {
  const [note, setNote] = useState("");

  const mutation = useMutation({
    mutationFn: () => rejectCarrierInvoiceMatch({ matchId, note: note.trim() }),
    onSuccess: () => {
      toast.success("Match rejected");
      setNote("");
      onOpenChange(false);
      onChanged();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to reject match"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Reject match</DialogTitle>
          <DialogDescription>
            Dismisses the pairing — the invoice stays open for a different assignment or a dispute
            with the carrier.
          </DialogDescription>
        </DialogHeader>
        <Textarea
          value={note}
          onChange={(event) => setNote(event.target.value)}
          placeholder="Reason (required) — e.g. Invoice bills a different load"
          rows={3}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={!note.trim() || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            Reject
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
