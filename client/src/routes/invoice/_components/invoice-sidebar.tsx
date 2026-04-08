import { EmptyState } from "@/components/empty-state";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { billTypeChoices } from "@/lib/choices";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { InvoiceStatus } from "@/types/invoice";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FileTextIcon, ReceiptTextIcon, SearchIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useDeferredValue } from "react";
import { toast } from "sonner";
import { invoiceSearchParamsParser } from "../use-invoice-state";
import { InvoiceItemCard } from "./invoice-item-card";

const invoiceStatusChoices: Array<{ label: string; value: InvoiceStatus }> = [
  { label: "Draft", value: "Draft" },
  { label: "Posted", value: "Posted" },
];

export function InvoiceSidebar({
  selectedInvoiceId,
  onSelectInvoice,
}: {
  selectedInvoiceId: string | null;
  onSelectInvoice: (id: string) => void;
}) {
  const [searchParams, setSearchParams] = useQueryStates(invoiceSearchParamsParser);
  const { status, query, billType } = searchParams;
  const deferredSearch = useDeferredValue(query);
  const queryClient = useQueryClient();

  const { mutate: postInvoice } = useMutation({
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

  const { data, isLoading } = useQuery({
    queryKey: ["invoice-list", status, billType, deferredSearch],
    queryFn: async () => {
      const params = new URLSearchParams({ limit: "100" });
      const filters: Array<{ field: string; operator: string; value: string }> = [];

      if (status) {
        filters.push({ field: "status", operator: "eq", value: status });
      }
      if (billType) {
        filters.push({ field: "billType", operator: "eq", value: billType });
      }
      if (deferredSearch.trim()) {
        params.set("query", deferredSearch.trim());
      }
      if (filters.length > 0) {
        params.set("fieldFilters", JSON.stringify(filters));
      }

      return apiService.invoiceService.list(Object.fromEntries(params.entries()));
    },
  });

  const invoices = data?.results ?? [];

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
        <div className={cn("flex flex-col gap-2 p-2", invoices.length === 0 && "h-full p-0")}>
          {!isLoading && invoices.length === 0 ? (
            <div className="flex h-full items-center justify-center">
              <EmptyState
                title="No invoices found"
                description="Adjust the search or filters to find draft and posted invoices."
                icons={[ReceiptTextIcon, FileTextIcon, ReceiptTextIcon]}
                className="flex flex-col h-full max-w-none items-center justify-center rounded-none border-none p-6 shadow-none"
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
        </div>
      </ScrollArea>
    </div>
  );
}
