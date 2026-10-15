import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { BillingListEmpty } from "@/components/billing/billing-empty";
import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import {
  InvoiceTableDocument,
  type DataTablePageInfoFieldsFragment,
  type InvoiceTableQueryVariables,
  type InvoiceTableRowFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { SelectionDock } from "@/components/selection-dock";
import { useListSelection } from "@/hooks/use-list-selection";
import { BULK_CONCURRENCY, notifyBulkOutcome, settleAll } from "@/lib/bulk-outcome";
import { apiService } from "@/services/api";
import type { DockAction } from "@trenova/shared/types/data-table";
import { billTypeChoices, invoiceScopeChoices, invoiceStatusChoices } from "@/lib/choices";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { Mail01Icon, SearchLgIcon, Send01Icon } from "@trenova/shared/components/icons";
import { useQueryStates } from "nuqs";
import { useCallback, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import { invoiceSidebarSearchParamsParser } from "../use-invoice-state";
import { InvoiceBulkConfirm, type InvoiceBulkKind } from "./invoice-bulk-confirm";
import { invoiceBulkEligibility } from "./invoice-bulk";
import { InvoiceItemCard } from "./invoice-item-card";
import { InvoicePostDialog } from "./invoice-post-dialog";

const PAGE_SIZE = 20;

type InvoiceRow = InvoiceTableRowFieldsFragment;

const invoiceId = (invoice: InvoiceRow) => invoice.id;
const invoiceLabel = (invoice: InvoiceRow) => invoice.number;

type InvoiceTablePage = {
  invoices: {
    edges: Array<{ node: InvoiceTableRowFieldsFragment }>;
    totalCount: number | null;
    pageInfo: DataTablePageInfoFieldsFragment;
  };
};

export function InvoiceSidebar({
  selectedInvoiceId,
  onSelectInvoice,
}: {
  selectedInvoiceId: string | null;
  onSelectInvoice: (id: string) => void;
}) {
  const t = useT();
  const { allowed: canPost } = usePermission(Resource.Invoice, Operation.Post);

  const [searchParams, setSearchParams] = useQueryStates(invoiceSidebarSearchParamsParser);
  const { status, query, billType, scope, dispute } = searchParams;
  const deferredSearch = useDeferredValue(query);
  const hasActiveFilters = Boolean(status || billType || scope || query || dispute);
  const clearFilters = () =>
    void setSearchParams({ status: null, billType: null, scope: null, query: "", dispute: null });
  const observerTarget = useRef<HTMLDivElement>(null);
  const [postingInvoiceId, setPostingInvoiceId] = useState<string | null>(null);

  const statusOptions = useMemo(
    () => withAllOption(t("All statuses"), invoiceStatusChoices, t),
    [t],
  );
  const billTypeOptions = useMemo(
    () => withAllOption(t("All bill types"), billTypeChoices, t),
    [t],
  );
  const scopeOptions = useMemo(() => withAllOption(t("All scopes"), invoiceScopeChoices, t), [t]);

  const queryKey = useMemo(
    () => ["invoice-list", status, billType, scope, dispute, deferredSearch],
    [status, billType, scope, dispute, deferredSearch],
  );

  const { data, isLoading, hasNextPage, isFetchingNextPage, fetchNextPage } = useInfiniteQuery({
    queryKey,
    queryFn: async ({ pageParam, signal }) => {
      const fieldFilters: Array<{ field: string; operator: string; value: string }> = [];

      if (status) {
        fieldFilters.push({ field: "status", operator: "eq", value: status });
      }
      if (billType) {
        fieldFilters.push({ field: "billType", operator: "eq", value: billType });
      }
      if (scope) {
        fieldFilters.push({ field: "scope", operator: "eq", value: scope });
      }
      if (dispute) {
        fieldFilters.push({ field: "disputeStatus", operator: "eq", value: "Disputed" });
      }

      return requestGraphQL<InvoiceTablePage, InvoiceTableQueryVariables>({
        document: InvoiceTableDocument,
        operationName: "InvoiceTable",
        variables: {
          input: {
            first: PAGE_SIZE,
            after: pageParam ?? undefined,
            query: deferredSearch.trim() || undefined,
            fieldFilters,
          },
        },
        signal,
      });
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => {
      const { pageInfo } = lastPage.invoices;
      return pageInfo.hasNextPage ? pageInfo.endCursor : undefined;
    },
  });

  const invoices = useMemo(
    () => data?.pages.flatMap((page) => page.invoices.edges.map((edge) => edge.node)) ?? [],
    [data?.pages],
  );

  const queryClient = useQueryClient();
  const selection = useListSelection(invoices, invoiceId);
  const clearSelection = selection.clear;
  const eligible = useMemo(() => invoiceBulkEligibility(selection.selected), [selection.selected]);
  const [confirm, setConfirm] = useState<{ kind: InvoiceBulkKind; invoices: InvoiceRow[] } | null>(
    null,
  );
  const [running, setRunning] = useState(false);

  const runBulk = useCallback(
    async (kind: InvoiceBulkKind, targets: InvoiceRow[]) => {
      setRunning(true);
      try {
        const outcome = await settleAll(
          targets,
          (invoice) =>
            kind === "post"
              ? apiService.invoiceService.post(invoice.id)
              : apiService.invoiceService.send(invoice.id),
          { concurrency: BULK_CONCURRENCY, label: invoiceLabel },
        );
        void queryClient.invalidateQueries({ queryKey: ["invoice"] });
        void queryClient.invalidateQueries({ queryKey: ["invoice-list"] });
        void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
        void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
        notifyBulkOutcome(
          outcome,
          kind === "post"
            ? {
                succeeded: (count) =>
                  t("{0, plural, one {Posted # invoice} other {Posted # invoices}}", count),
                partial: (succeeded, failed) =>
                  t("Posted {0}; {1} could not be posted", succeeded, failed),
                allFailed: (failed) =>
                  t("{0, plural, one {# invoice} other {# invoices}} could not be posted", failed),
              }
            : {
                succeeded: (count) =>
                  t("{0, plural, one {Sending # invoice} other {Sending # invoices}}", count),
                partial: (succeeded, failed) =>
                  t("Sending {0}; {1} could not be sent", succeeded, failed),
                allFailed: (failed) =>
                  t("{0, plural, one {# invoice} other {# invoices}} could not be sent", failed),
              },
        );
        setConfirm(null);
        if (outcome.failed.length === 0) {
          clearSelection();
        }
      } finally {
        setRunning(false);
      }
    },
    [queryClient, clearSelection, t],
  );

  const dockActions: DockAction<InvoiceRow>[] = [];
  if (canPost && eligible.postable.length > 0) {
    dockActions.push({
      id: "post",
      label: t("Post {0}", eligible.postable.length),
      icon: Send01Icon,
      onClick: () => setConfirm({ kind: "post", invoices: eligible.postable }),
    });
  }
  if (eligible.sendable.length > 0) {
    dockActions.push({
      id: "send",
      label: t("Send {0}", eligible.sendable.length),
      icon: Mail01Icon,
      onClick: () => setConfirm({ kind: "send", invoices: eligible.sendable }),
    });
  }

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
    <div className="flex h-full flex-col">
      <div className="flex flex-col gap-1.5 border-b p-2">
        <Input
          placeholder={t("Search invoice, PRO, bill-to...")}
          leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
          value={query}
          onChange={(event) => void setSearchParams({ query: event.target.value })}
          className="h-7 text-xs"
        />
        <div className="grid grid-cols-2 gap-2">
          <FilterSelect
            label={t("Invoice status")}
            value={status}
            options={statusOptions}
            onChange={(value) => void setSearchParams({ status: value })}
          />
          <FilterSelect
            label={t("Bill type")}
            value={billType}
            options={billTypeOptions}
            onChange={(value) => void setSearchParams({ billType: value })}
          />
          <FilterSelect
            label={t("Invoice scope")}
            value={scope}
            options={scopeOptions}
            onChange={(value) => void setSearchParams({ scope: value })}
          />
          <div className="flex h-7 items-center gap-2 px-1 text-xs">
            <Checkbox
              id="invoice-filter-disputed"
              checked={dispute}
              onCheckedChange={(checked) =>
                void setSearchParams({ dispute: checked === true ? true : null })
              }
              aria-label={t("Disputed only")}
            />
            <Label htmlFor="invoice-filter-disputed" className="text-xs font-normal">
              {t("Disputed only")}
            </Label>
          </div>
        </div>
      </div>

      {invoices.length > 0 && (
        <label className="text-muted-foreground flex items-center gap-2 border-b px-3 py-1.5 text-xs">
          <Checkbox
            checked={selection.allState === "all"}
            indeterminate={selection.allState === "some"}
            onCheckedChange={(checked) => selection.setAll(checked === true)}
            aria-label={t("Select all loaded invoices")}
          />
          {selection.allState === "none"
            ? t("{0, plural, one {# invoice loaded} other {# invoices loaded}}", invoices.length)
            : t("{0} of {1} selected", selection.selected.length, invoices.length)}
        </label>
      )}
      <ScrollArea className="flex-1">
        <div
          className={cn("flex flex-col gap-1.5 p-2", invoices.length === 0 && "h-full gap-0 p-0")}
        >
          {!isLoading && invoices.length === 0 ? (
            <BillingListEmpty
              title={hasActiveFilters ? "Nothing matches" : "No invoices yet"}
              description={
                hasActiveFilters
                  ? "No invoice fits the search and filters. Widen them, or clear them to see every draft and posted invoice."
                  : "An invoice is drafted from the billing queue as each item there is approved. Until one is, there is nothing to review here."
              }
              onClearFilters={hasActiveFilters ? clearFilters : undefined}
            />
          ) : null}
          {invoices.map((invoice) => (
            <InvoiceItemCard
              key={invoice.id}
              invoice={invoice}
              isSelected={selectedInvoiceId === invoice.id}
              onClick={() => onSelectInvoice(invoice.id)}
              onPost={() => setPostingInvoiceId(invoice.id)}
              checked={selection.isSelected(invoice.id)}
              onCheckedChange={(checked) => selection.setSelected(invoice.id, checked)}
            />
          ))}
          {isFetchingNextPage ? (
            <div className="flex items-center justify-center py-4">
              <TextShimmer className="font-mono text-sm" duration={1}>
                {t("Loading more...")}
              </TextShimmer>
            </div>
          ) : null}
          <div ref={observerTarget} className="h-px" />
        </div>
      </ScrollArea>
      {postingInvoiceId && (
        <InvoicePostDialog
          invoiceId={postingInvoiceId}
          open
          onOpenChange={(open) => !open && setPostingInvoiceId(null)}
        />
      )}
      {confirm && (
        <InvoiceBulkConfirm
          kind={confirm.kind}
          count={confirm.invoices.length}
          skipped={selection.selected.length - confirm.invoices.length}
          isPending={running}
          onOpenChange={(open) => !open && !running && setConfirm(null)}
          onConfirm={() => void runBulk(confirm.kind, confirm.invoices)}
        />
      )}
      <SelectionDock
        selectedCount={selection.selected.length}
        actions={dockActions}
        getSelected={() => selection.selected}
        onClearSelection={clearSelection}
      />
    </div>
  );
}

const ALL_FILTER_VALUE = "all";

type FilterOption = { value: string; label: string };

function withAllOption(
  allLabel: string,
  choices: ReadonlyArray<{ value: string; label: string }>,
  t: TranslateFn,
): FilterOption[] {
  return [
    { value: ALL_FILTER_VALUE, label: allLabel },
    ...choices.map((choice) => ({ value: choice.value, label: t(choice.label) })),
  ];
}

function FilterSelect({
  label,
  value,
  options,
  onChange,
  className,
}: {
  label: string;
  value: string | null;
  options: FilterOption[];
  onChange: (value: string | null) => void;
  className?: string;
}) {
  return (
    <Select
      value={value ?? ALL_FILTER_VALUE}
      items={options}
      onValueChange={(next) => onChange(!next || next === ALL_FILTER_VALUE ? null : next)}
    >
      <SelectTrigger className={cn("h-7 w-full text-xs", className)} aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
