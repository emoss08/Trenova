import { EmptyState } from "@/components/empty-state";
import { Input } from "@trenova/shared/components/ui/input";
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
import { billTypeChoices, invoiceStatusChoices } from "@/lib/choices";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import { useInfiniteQuery } from "@tanstack/react-query";
import { FileTextIcon, ReceiptTextIcon, SearchIcon } from "lucide-react";
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
  const [searchParams, setSearchParams] = useQueryStates(invoiceSidebarSearchParamsParser);
  const { status, query, billType } = searchParams;
  const deferredSearch = useDeferredValue(query);
  const observerTarget = useRef<HTMLDivElement>(null);
  const { mutate: postInvoice } = usePostInvoice();

  const queryKey = useMemo(
    () => ["invoice-list", status, billType, deferredSearch],
    [status, billType, deferredSearch],
  );

  const { data, isLoading, hasNextPage, isFetchingNextPage, fetchNextPage } = useInfiniteQuery({
    queryKey,
    queryFn: async ({ pageParam }) => {
      const fieldFilters: Array<{ field: string; operator: string; value: string }> = [];

      if (status) {
        fieldFilters.push({ field: "status", operator: "eq", value: status });
      }
      if (billType) {
        fieldFilters.push({ field: "billType", operator: "eq", value: billType });
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
          placeholder="Search invoice, PRO, bill-to..."
          leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
          value={query}
          onChange={(event) => void setSearchParams({ query: event.target.value })}
          className="h-7 text-xs"
        />
        <div className="flex gap-2">
          <Select
            value={status ?? "all"}
            items={invoiceStatusChoices}
            onValueChange={(value) =>
              void setSearchParams({ status: value === "all" ? null : value })
            }
          >
            <SelectTrigger className="h-7 text-xs">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Statuses</SelectItem>
              {invoiceStatusChoices.map((choice) => (
                <SelectItem key={choice.value} value={choice.value}>
                  {choice.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={billType ?? "all"}
            items={billTypeChoices}
            onValueChange={(value) =>
              void setSearchParams({ billType: value === "all" ? null : value })
            }
          >
            <SelectTrigger className="h-7 text-xs">
              <SelectValue placeholder="All bill types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Bill Types</SelectItem>
              {billTypeChoices.map((choice) => (
                <SelectItem key={choice.value} value={choice.value}>
                  {choice.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <ScrollArea className="flex-1">
        <div
          className={cn("flex flex-col gap-1.5 p-2", invoices.length === 0 && "h-full gap-0 p-0")}
        >
          {!isLoading && invoices.length === 0 ? (
            <div className="flex h-full items-center justify-center">
              <EmptyState
                title="No invoices found"
                description="Adjust the search or filters to find draft and posted invoices."
                icons={[ReceiptTextIcon, FileTextIcon, ReceiptTextIcon]}
                className="flex h-full max-w-none flex-col items-center justify-center rounded-none border-none p-6 shadow-none"
              />
            </div>
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
                Loading more...
              </TextShimmer>
            </div>
          ) : null}
          <div ref={observerTarget} className="h-px" />
        </div>
      </ScrollArea>
    </div>
  );
}
