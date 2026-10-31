import { useT } from "@trenova/shared/i18n/use-t";
import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import { MultiSelectAutocomplete } from "@/components/fields/multi-select-field";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { parseDateWindow, serializeDateWindow } from "@/lib/billing-queue-date-window";
import { billTypeChoices, billingQueueStatusChoices } from "@/lib/choices";
import type {
  GraphQLSelectOptionsConfig,
  SelectOption as GraphQLSelectOption,
} from "@/lib/graphql/select-options";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { BillingQueueFilterPreset } from "@trenova/shared/types/billing-queue";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  FilterFunnel01Icon,
  Save01Icon,
  Trash01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { type inferParserType, useQueryStates } from "nuqs";
import { useState } from "react";
import type { FieldValues } from "react-hook-form";
import {
  BILLING_QUEUE_FILTER_PRESETS_KEY,
  billingQueueFilterPresetsQuery,
} from "../billing-queue-queries";
import { queueSidebarSearchParamsParser } from "../use-billing-queue-state";
import { BillingQueueDateWindowField } from "./billing-queue-date-window-field";
import { BillingQueueSavePresetDialog } from "./billing-queue-save-preset-dialog";

/** The customers the queue can be narrowed to, by name or code. */
const CUSTOMER_SELECT_OPTIONS = { resource: "CUSTOMER" } satisfies GraphQLSelectOptionsConfig;

/** The people a queue item can be assigned to bill. */
const BILLER_SELECT_OPTIONS = { resource: "USER" } satisfies GraphQLSelectOptionsConfig;

const ALL = "all";

export type QueueFilterValues = inferParserType<typeof queueSidebarSearchParamsParser>;

export const CLEARED_QUEUE_FILTERS: QueueFilterValues = {
  status: null,
  billers: [],
  billType: null,
  payer: null,
  query: "",
  includePosted: false,
  preset: null,
  lanes: [],
  minAmount: null,
  maxAmount: null,
  createdBefore: null,
  shipmentCustomer: null,
  delivered: null,
  pickedUp: null,
  queued: null,
};

/** How many filters narrow the list, not counting the search text. */
export function activeQueueFilterCount(params: QueueFilterValues): number {
  return (
    (params.status ? 1 : 0) +
    (params.billers.length > 0 ? 1 : 0) +
    (params.billType ? 1 : 0) +
    (params.payer ? 1 : 0) +
    (params.includePosted ? 1 : 0) +
    (params.lanes.length > 0 ? 1 : 0) +
    (params.minAmount || params.maxAmount ? 1 : 0) +
    (params.createdBefore ? 1 : 0) +
    (params.shipmentCustomer ? 1 : 0) +
    (params.delivered ? 1 : 0) +
    (params.pickedUp ? 1 : 0) +
    (params.queued ? 1 : 0)
  );
}

/**
 * A customer to narrow the queue to: the shipment's own customer, or the payer a bill
 * goes to, which for a split shipment is not the same one.
 */
function CustomerFilter({
  value,
  onChange,
  placeholder,
}: {
  value: string | null;
  onChange: (value: string | null) => void;
  placeholder: string;
}) {
  const t = useT();

  return (
    <Autocomplete<GraphQLSelectOption, FieldValues>
      graphql={CUSTOMER_SELECT_OPTIONS}
      value={value}
      onChange={(val) => onChange(val ? String(val) : null)}
      getOptionValue={(option) => option.id}
      getDisplayValue={(option) => {
        const code = selectOptionMetaString(option, "code");
        return code ? t("{0} - {1}", code, option.label) : option.label;
      }}
      renderOption={(option) => <span className="text-xs">{option.label}</span>}
      placeholder={placeholder}
      clearable
      triggerClassName="h-7 text-xs"
    />
  );
}

export function BillingQueueFilters() {
  const t = useT();
  const queryClient = useQueryClient();
  const [params, setParams] = useQueryStates(queueSidebarSearchParamsParser);
  const [savePresetOpen, setSavePresetOpen] = useState(false);
  const activeCount = activeQueueFilterCount(params);
  const hasActiveFilters = activeCount > 0 || Boolean(params.query);

  const { data: presets = [] } = useQuery(billingQueueFilterPresetsQuery());
  // The select names what it holds from these, so "all" reads as words, not its value.
  const statusItems = [
    { value: ALL, label: t("All statuses") },
    ...billingQueueStatusChoices.map((choice) => ({ value: choice.value, label: t(choice.label) })),
  ];
  const billTypeItems = [
    { value: ALL, label: t("All bill types") },
    ...billTypeChoices.map((choice) => ({ value: choice.value, label: t(choice.label) })),
  ];

  const deletePreset = useApiMutation({
    mutationFn: (id: string) => apiService.billingQueueService.deleteFilterPreset(id),
    resourceName: "BillingQueueFilterPreset",
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [BILLING_QUEUE_FILTER_PRESETS_KEY] });
      void setParams({ preset: null });
    },
  });

  const applyPreset = (preset: BillingQueueFilterPreset) => {
    const f = preset.filters;
    void setParams({
      ...CLEARED_QUEUE_FILTERS,
      status: (f.status as string) ?? null,
      billers: Array.isArray(f.assignedBillerIds) ? f.assignedBillerIds : [],
      billType: (f.billType as string) ?? null,
      payer: (f.billToCustomerId as string) ?? null,
      shipmentCustomer: (f.shipmentCustomerId as string) ?? null,
      delivered: parseDateWindow(f.delivered as string | undefined),
      pickedUp: parseDateWindow(f.pickedUp as string | undefined),
      queued: parseDateWindow(f.queued as string | undefined),
      query: (f.search as string) ?? "",
      includePosted: f.includePosted === true,
      preset: preset.id,
    });
  };

  const currentFilters: Record<string, string | string[] | null> = {
    status: params.status,
    assignedBillerIds: params.billers.length > 0 ? params.billers : null,
    billType: params.billType,
    billToCustomerId: params.payer,
    shipmentCustomerId: params.shipmentCustomer,
    delivered: params.delivered ? serializeDateWindow(params.delivered) : null,
    pickedUp: params.pickedUp ? serializeDateWindow(params.pickedUp) : null,
    queued: params.queued ? serializeDateWindow(params.queued) : null,
    search: params.query || null,
    includePosted: params.includePosted ? "true" : null,
  };

  return (
    <>
      <Popover>
        <PopoverTrigger
          render={
            <Button size="sm" variant="outline" className="shrink-0 gap-1.5">
              <FilterFunnel01Icon className="size-3.5" />
              {t("Filter")}
              {activeCount > 0 && (
                <span className="bg-primary text-primary-foreground rounded-badge flex size-4 items-center justify-center text-2xs font-medium">
                  {activeCount}
                </span>
              )}
            </Button>
          }
        />
        <PopoverContent sideOffset={4} className="w-100 p-3">
          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium">{t("Filters")}</span>
              {hasActiveFilters && (
                <Button
                  size="xs"
                  variant="ghost"
                  onClick={() => void setParams(CLEARED_QUEUE_FILTERS)}
                  className="h-5 px-1"
                >
                  <XCloseIcon className="mr-0.5 size-3" />
                  <span className="text-xs">{t("Clear all")}</span>
                </Button>
              )}
            </div>
            <div className="flex flex-col gap-3">
              <div className="grid grid-cols-2 gap-3">
                <div className="flex min-w-0 flex-col gap-1.5">
                  <p className="text-muted-foreground text-xs">{t("Status")}</p>
                  <Select
                    value={params.status ?? ALL}
                    items={statusItems}
                    onValueChange={(v) => void setParams({ status: v === ALL ? null : v })}
                  >
                    <SelectTrigger aria-label={t("Status")} className="h-7 w-full text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {statusItems.map((item) => (
                        <SelectItem key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex min-w-0 flex-col gap-1.5">
                  <p className="text-muted-foreground text-xs">{t("Bill type")}</p>
                  <Select
                    value={params.billType ?? ALL}
                    items={billTypeItems}
                    onValueChange={(v) => void setParams({ billType: v === ALL ? null : v })}
                  >
                    <SelectTrigger aria-label={t("Bill type")} className="h-7 w-full text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {billTypeItems.map((item) => (
                        <SelectItem key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="flex min-w-0 flex-col gap-1.5">
                  <p className="text-muted-foreground text-xs">{t("Customer")}</p>
                  <CustomerFilter
                    value={params.shipmentCustomer}
                    onChange={(val) => void setParams({ shipmentCustomer: val })}
                    placeholder={t("All customers")}
                  />
                </div>
                <div className="flex min-w-0 flex-col gap-1.5">
                  <p className="text-muted-foreground text-xs">{t("Payer")}</p>
                  <CustomerFilter
                    value={params.payer}
                    onChange={(val) => void setParams({ payer: val })}
                    placeholder={t("All payers")}
                  />
                </div>
              </div>
              <div className="flex flex-col gap-2 rounded-md border p-2.5">
                <p className="text-xs font-medium">{t("Dates")}</p>
                <BillingQueueDateWindowField
                  label={t("Delivered")}
                  value={params.delivered}
                  onChange={(value) => void setParams({ delivered: value })}
                />
                <BillingQueueDateWindowField
                  label={t("Picked up")}
                  value={params.pickedUp}
                  onChange={(value) => void setParams({ pickedUp: value })}
                />
                <BillingQueueDateWindowField
                  label={t("Queued")}
                  value={params.queued}
                  onChange={(value) => void setParams({ queued: value })}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <p className="text-muted-foreground text-xs">{t("Assigned billers")}</p>
                <MultiSelectAutocomplete<GraphQLSelectOption>
                  link="/users/select-options/"
                  graphql={BILLER_SELECT_OPTIONS}
                  label={t("Billers")}
                  className="text-foreground"
                  placeholder={t("All billers")}
                  values={params.billers}
                  onChange={(values) =>
                    void setParams({
                      billers: values
                        .map((v) => (typeof v === "string" ? v : (v.id ?? "")))
                        .filter(Boolean),
                    })
                  }
                  getOptionValue={(option) => option.id}
                  getDisplayValue={(option) => option.label}
                  renderOption={(option) => <span className="text-xs">{option.label}</span>}
                  triggerClassName="h-7 text-xs"
                  maxCount={2}
                />
              </div>
              <label className="flex items-start gap-2 rounded-md border p-2.5">
                <Checkbox
                  checked={params.includePosted}
                  onCheckedChange={(checked) => void setParams({ includePosted: checked === true })}
                />
                <div className="space-y-0.5">
                  <p className="text-xs font-medium">{t("Include posted items")}</p>
                  <p className="text-muted-foreground text-xs">
                    {t("Show historical queue records that already produced a posted invoice.")}
                  </p>
                </div>
              </label>
            </div>

            <div className="flex items-center gap-1 border-t pt-3">
              <div className="flex-1">
                <Autocomplete<BillingQueueFilterPreset, FieldValues>
                  link="/billing-queue/filter-presets/"
                  value={params.preset}
                  onChange={(value) => {
                    const preset = presets.find((p) => p.id === value);
                    if (preset) {
                      applyPreset(preset);
                    } else {
                      void setParams({ preset: null });
                    }
                  }}
                  getOptionValue={(preset) => preset.id}
                  getDisplayValue={(preset) => preset.name}
                  renderOption={(preset) => <span className="text-xs">{preset.name}</span>}
                  placeholder={t("Select preset...")}
                  clearable
                  triggerClassName="h-7 text-xs"
                />
              </div>
              {hasActiveFilters && (
                <Button
                  size="xs"
                  variant="ghost"
                  onClick={() => setSavePresetOpen(true)}
                  aria-label={t("Save current filters as preset")}
                >
                  <Save01Icon className="size-3" />
                </Button>
              )}
              {params.preset && (
                <Button
                  size="xs"
                  variant="ghost"
                  onClick={() => params.preset && deletePreset.mutate(params.preset)}
                  aria-label={t("Delete selected preset")}
                >
                  <Trash01Icon className="size-3" />
                </Button>
              )}
            </div>
          </div>
        </PopoverContent>
      </Popover>
      <BillingQueueSavePresetDialog
        open={savePresetOpen}
        onOpenChange={setSavePresetOpen}
        filters={currentFilters}
      />
    </>
  );
}
