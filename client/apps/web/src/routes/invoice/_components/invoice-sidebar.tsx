import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { BillingListEmpty } from "@/components/billing/billing-empty";
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
import { usePostInvoice } from "@/hooks/use-post-invoice";
import { billTypeChoices, invoiceScopeChoices, invoiceStatusChoices } from "@/lib/choices";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import { useInfiniteQuery } from "@tanstack/react-query";
import { SearchIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useDeferredValue, useEffect, useMemo, useRef } from "react";
import { invoiceSidebarSearchParamsParser } from "../use-invoice-state";
import { InvoiceItemCard } from "./invoice-item-card";

const PAGE_SIZE = 20;

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

  const [searchParams, setSearchParams] = useQueryStates(invoiceSidebarSearchParamsParser);
  const { status, query, billType, scope, dispute } = searchParams;
  const deferredSearch = useDeferredValue(query);
  const hasActiveFilters = Boolean(status || billType || scope || query || dispute);
  const clearFilters = () =>
    void setSearchParams({ status: null, billType: null, scope: null, query: "", dispute: null });
  const observerTarget = useRef<HTMLDivElement>(null);
  const { mutate: postInvoice } = usePostInvoice();

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
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
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
              onPost={() => postInvoice(invoice.id)}
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
