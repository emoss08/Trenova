import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import type { FieldValues } from "react-hook-form";
import { MultiSelectAutocomplete } from "@/components/fields/multi-select-field";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api } from "@/lib/api";
import { billTypeChoices, billingQueueStatusChoices } from "@/lib/choices";
import { safeParse } from "@/lib/parse";
import { apiService } from "@/services/api";
import {
  billingQueueItemSchema,
  type BillingQueueFilterPreset,
  type BillingQueueItem,
} from "@/types/billing-queue";
import { createLimitOffsetResponse } from "@/types/server";
import type { User } from "@/types/user";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FilterIcon, InboxIcon, SaveIcon, SearchIcon, Trash2Icon, XIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, useDeferredValue, useEffect, useState } from "react";
import { toast } from "sonner";
import { queueSearchParamsParser } from "../use-billing-queue-state";
import { BillingQueueAssignDialog } from "./billing-queue-assign-dialog";
import { BillingQueueCancelDialog } from "./billing-queue-cancel-dialog";
import { BillingQueueItemCard } from "./billing-queue-item-card";
import { BillingQueueSavePresetDialog } from "./billing-queue-save-preset-dialog";

const billingQueueListSchema = createLimitOffsetResponse(billingQueueItemSchema);

function PresetSelector({
  selectedPresetId,
  onSelect,
}: {
  presets: BillingQueueFilterPreset[];
  selectedPresetId: string | null;
  onSelect: (value: string) => void;
}) {
  return (
    <div className="flex-1">
      <Autocomplete<BillingQueueFilterPreset, FieldValues>
        link="/billing-queue/filter-presets/"
        value={selectedPresetId}
        onChange={(val) => onSelect(val ?? "none")}
        getOptionValue={(preset) => preset.id}
        getDisplayValue={(preset) => preset.name}
        renderOption={(preset) => (
          <span className="text-xs">{preset.name}</span>
        )}
        placeholder="Select preset..."
        clearable
        triggerClassName="h-7 text-xs"
      />
    </div>
  );
}

export function BillingQueueSidebar({
  selectedItemId,
  onSelectItem,
}: {
  selectedItemId: string | null;
  onSelectItem: (id: string) => void;
}) {
  const [searchParams, setSearchParams] = useQueryStates(queueSearchParamsParser);
  const { status: statusFilter, query: search, billType: billTypeFilter, billers: billerFilter, preset: selectedPresetId } = searchParams;
  const deferredSearch = useDeferredValue(search);

  const [assignItemId, setAssignItemId] = useState<string | null>(null);
  const [cancelItemId, setCancelItemId] = useState<string | null>(null);
  const [savePresetOpen, setSavePresetOpen] = useState(false);
  const queryClient = useQueryClient();

  const activeFilterCount =
    (statusFilter ? 1 : 0) + (billerFilter.length > 0 ? 1 : 0) + (billTypeFilter ? 1 : 0);
  const hasActiveFilters = activeFilterCount > 0 || !!search;

  const { data: presetsData } = useQuery({
    queryKey: ["billing-queue-filter-presets"],
    queryFn: () => apiService.billingQueueService.listFilterPresets(),
    staleTime: 5 * 60 * 1000,
  });

  const { mutate: deletePreset } = useMutation({
    mutationFn: (id: string) => apiService.billingQueueService.deleteFilterPreset(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-filter-presets"] });
      void setSearchParams({ preset: null });
      toast.success("Filter preset deleted");
    },
    onError: () => {
      toast.error("Failed to delete filter preset");
    },
  });

  const presets: BillingQueueFilterPreset[] = presetsData ?? [];

  const applyPreset = useCallback(
    (preset: BillingQueueFilterPreset) => {
      const f = preset.filters;
      void setSearchParams({
        status: (f.status as string) ?? null,
        billers: Array.isArray(f.assignedBillerIds) ? f.assignedBillerIds : [],
        billType: (f.billType as string) ?? null,
        query: (f.search as string) ?? "",
        preset: preset.id,
      });
    },
    [setSearchParams],
  );

  const handlePresetChange = useCallback(
    (value: string | null) => {
      if (!value || value === "none") {
        void setSearchParams({ preset: null });
        return;
      }
      const preset = presets.find((p) => p.id === value);
      if (preset) {
        applyPreset(preset);
      }
    },
    [presets, applyPreset, setSearchParams],
  );

  const { data, isLoading } = useQuery({
    queryKey: ["billing-queue-list", statusFilter, billerFilter, billTypeFilter, deferredSearch],
    queryFn: async () => {
      const params = new URLSearchParams({ limit: "100" });
      const filters: Array<{ field: string; operator: string; value: string | string[] }> = [];
      if (statusFilter) {
        filters.push({ field: "status", operator: "eq", value: statusFilter });
      }
      if (billerFilter.length === 1) {
        filters.push({ field: "assignedBillerId", operator: "eq", value: billerFilter[0] });
      } else if (billerFilter.length > 1) {
        filters.push({ field: "assignedBillerId", operator: "in", value: billerFilter });
      }
      if (billTypeFilter) {
        filters.push({ field: "billType", operator: "eq", value: billTypeFilter });
      }
      if (deferredSearch.trim()) {
        params.set("query", deferredSearch.trim());
      }
      if (filters.length > 0) {
        params.set("fieldFilters", JSON.stringify(filters));
      }
      const response = await api.get(`/billing-queue/?${params.toString()}`);
      return safeParse(billingQueueListSchema, response, "BillingQueueList");
    },
  });

  const { mutate: updateStatus } = useMutation({
    mutationFn: ({ itemId, status }: { itemId: string; status: string }) =>
      apiService.billingQueueService.updateStatus(itemId, { status: status as any }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
    },
    onError: () => {
      toast.error("Failed to update status");
    },
  });

  const items = data?.results ?? [];

  const navigateItems = useCallback(
    (direction: "next" | "prev") => {
      if (items.length === 0) return;
      if (!selectedItemId) {
        onSelectItem(items[0].id);
        return;
      }
      const currentIndex = items.findIndex((i) => i.id === selectedItemId);
      const nextIndex =
        direction === "next"
          ? Math.min(currentIndex + 1, items.length - 1)
          : Math.max(currentIndex - 1, 0);
      onSelectItem(items[nextIndex].id);
    },
    [items, selectedItemId, onSelectItem],
  );

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
      if (e.key === "j") navigateItems("next");
      if (e.key === "k") navigateItems("prev");
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [navigateItems]);

  const clearFilters = () => {
    void setSearchParams({
      status: null,
      billers: [],
      billType: null,
      query: "",
      preset: null,
    });
  };

  const currentFilters: Record<string, string | string[] | null> = {
    status: statusFilter,
    assignedBillerIds: billerFilter.length > 0 ? billerFilter : null,
    billType: billTypeFilter,
    search: search || null,
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-col gap-1.5 border-b p-2">
        <Popover>
          <div className="flex items-center gap-1">
            <Input
              placeholder="Search PRO, BOL..."
              leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
              value={search}
              onChange={(e) => void setSearchParams({ query: e.target.value })}
              className="h-7 flex-1 text-xs"
            />
            <PopoverTrigger
              render={
                <Button size="xs" variant="outline" className="relative h-7 shrink-0 gap-1 px-2">
                  <FilterIcon className="size-3" />
                  <span className="text-xs">Filters</span>
                  {activeFilterCount > 0 && (
                    <span className="flex size-4 items-center justify-center rounded-full bg-primary text-[10px] font-medium text-primary-foreground">
                      {activeFilterCount}
                    </span>
                  )}
                </Button>
              }
            />
          </div>
          <PopoverContent sideOffset={4} className="w-[400px] p-3">
            <div className="flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium">Filters</span>
                {hasActiveFilters && (
                  <Button size="xs" variant="ghost" onClick={clearFilters} className="h-5 px-1">
                    <XIcon className="mr-0.5 size-3" />
                    <span className="text-xs">Clear all</span>
                  </Button>
                )}
              </div>
              <div className="flex flex-col gap-1">
                <div className="flex flex-row gap-2">
                  <div className="flex flex-col gap-1">
                    <p className="text-[11px] text-muted-foreground">Status</p>
                    <Select
                      value={statusFilter ?? "all"}
                      items={billingQueueStatusChoices}
                      onValueChange={(v) => void setSearchParams({ status: v === "all" ? null : v })}
                    >
                      <SelectTrigger className="h-7 text-xs w-[150px]">
                        <SelectValue placeholder="All statuses" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">All Statuses</SelectItem>
                        {billingQueueStatusChoices.map((choice) => (
                          <SelectItem key={choice.value} value={choice.value}>
                            {choice.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="flex flex-col gap-1">
                    <p className="text-[11px] text-muted-foreground">Bill Type</p>
                    <Select
                      value={billTypeFilter ?? "all"}
                      items={billTypeChoices}
                      onValueChange={(v) => void setSearchParams({ billType: v === "all" ? null : v })}
                    >
                      <SelectTrigger className="h-7 text-xs w-[150px]">
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
                <div className="flex flex-col gap-1">
                  <p className="text-[11px] text-muted-foreground">Assigned Billers</p>
                  <MultiSelectAutocomplete<User>
                    link="/users/select-options/"
                    label="Billers"
                    className="text-foreground"
                    placeholder="All billers"
                    values={billerFilter}
                    onChange={(values) =>
                      void setSearchParams({
                        billers: values
                          .map((v) => (typeof v === "string" ? v : (v.id ?? "")))
                          .filter(Boolean),
                      })
                    }
                    getOptionValue={(user) => user.id ?? ""}
                    getDisplayValue={(user) => user.name}
                    renderOption={(user) => <span className="text-xs">{user.name}</span>}
                    triggerClassName="h-7 text-xs"
                    maxCount={2}
                  />
                </div>
              </div>

              <div className="flex items-center gap-1 border-t pt-2">
                <PresetSelector
                  presets={presets}
                  selectedPresetId={selectedPresetId}
                  onSelect={handlePresetChange}
                />
                {hasActiveFilters && (
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => setSavePresetOpen(true)}
                    title="Save current filters as preset"
                  >
                    <SaveIcon className="size-3" />
                  </Button>
                )}
                {selectedPresetId && (
                  <Button
                    size="xs"
                    variant="ghost"
                    onClick={() => deletePreset(selectedPresetId)}
                    title="Delete selected preset"
                  >
                    <Trash2Icon className="size-3" />
                  </Button>
                )}
              </div>
            </div>
          </PopoverContent>
        </Popover>
      </div>
      <ScrollArea className="flex-1">
        <div className="flex flex-col gap-1 p-2">
          {isLoading && (
            <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
              Loading...
            </div>
          )}
          {!isLoading && items.length === 0 && (
            <div className="flex flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
              <InboxIcon className="size-8" />
              <p className="text-sm">No items in queue</p>
            </div>
          )}
          {items.map((item: BillingQueueItem) => (
            <BillingQueueItemCard
              key={item.id}
              item={item}
              isSelected={item.id === selectedItemId}
              onClick={() => onSelectItem(item.id)}
              onAssignBiller={() => setAssignItemId(item.id)}
              onHold={() => updateStatus({ itemId: item.id, status: "OnHold" })}
              onCancel={() => setCancelItemId(item.id)}
            />
          ))}
        </div>
      </ScrollArea>
      {assignItemId && (
        <BillingQueueAssignDialog
          open={!!assignItemId}
          onOpenChange={(open) => !open && setAssignItemId(null)}
          itemId={assignItemId}
        />
      )}
      {cancelItemId && (
        <BillingQueueCancelDialog
          open={!!cancelItemId}
          onOpenChange={(open) => !open && setCancelItemId(null)}
          itemId={cancelItemId}
        />
      )}
      <BillingQueueSavePresetDialog
        open={savePresetOpen}
        onOpenChange={setSavePresetOpen}
        filters={currentFilters}
      />
    </div>
  );
}
