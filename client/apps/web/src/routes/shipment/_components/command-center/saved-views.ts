import type { FieldFilter } from "@trenova/shared/types/data-table";
import { shipmentStatusSchema, type ShipmentStatus } from "@trenova/shared/types/shipment";

export type SavedViewId = "all" | "transit" | "at-risk" | "unassigned" | "delivering-today";

export type ChipFilterId = "at-risk" | "reefer" | "today";

export type SavedView = {
  id: SavedViewId;
  label: string;
};

export const DEFAULT_VIEW_ID: SavedViewId = "all";

export const SAVED_VIEWS: readonly SavedView[] = [
  { id: "all", label: "All shipments" },
  { id: "transit", label: "In transit" },
  { id: "at-risk", label: "At risk" },
  { id: "unassigned", label: "Unassigned" },
  { id: "delivering-today", label: "Delivering today" },
] as const;

export const CHIP_FILTERS: readonly { id: ChipFilterId; label: string }[] = [
  { id: "at-risk", label: "At risk" },
  { id: "reefer", label: "Reefer" },
  { id: "today", label: "Today" },
] as const;

const STATUS_FIELD = "status";
const STATUS_FIELD_IN: FieldFilter["operator"] = "in";
const SHIPMENT_TYPE_CODE_FIELD = "shipmentType.code";
const DELIVERY_APPOINTMENT_FIELD = "deliveryAppointment.scheduledWindowStart";

const startOfTodaySeconds = () => Math.floor(new Date().setHours(0, 0, 0, 0) / 1000);

const statusInFilter = (statuses: ShipmentStatus[]): FieldFilter => ({
  field: STATUS_FIELD,
  operator: STATUS_FIELD_IN,
  value: statuses,
});

const todayDeliveryFilter = (): FieldFilter => ({
  field: DELIVERY_APPOINTMENT_FIELD,
  operator: "daterange",
  value: { from: startOfTodaySeconds(), to: startOfTodaySeconds() },
});

const reeferFilter = (): FieldFilter => ({
  field: SHIPMENT_TYPE_CODE_FIELD,
  operator: "ilike",
  value: "reefer",
});

export function getFiltersForView(view: SavedViewId): FieldFilter[] {
  switch (view) {
    case "all":
      return [];
    case "transit":
      return [statusInFilter([shipmentStatusSchema.enum.InTransit])];
    case "at-risk":
      return [statusInFilter([shipmentStatusSchema.enum.Delayed])];
    case "unassigned":
      return [
        statusInFilter([
          shipmentStatusSchema.enum.New,
          shipmentStatusSchema.enum.PartiallyAssigned,
        ]),
      ];
    case "delivering-today":
      return [todayDeliveryFilter()];
  }
}

export function getFiltersForChips(chips: ChipFilterId[]): FieldFilter[] {
  const filters: FieldFilter[] = [];
  for (const chip of chips) {
    switch (chip) {
      case "at-risk":
        filters.push(statusInFilter([shipmentStatusSchema.enum.Delayed]));
        break;
      case "reefer":
        filters.push(reeferFilter());
        break;
      case "today":
        filters.push(todayDeliveryFilter());
        break;
    }
  }
  return filters;
}

export function getMandatoryFieldFilters(view: SavedViewId, chips: ChipFilterId[]): FieldFilter[] {
  const merged = [...getFiltersForView(view), ...getFiltersForChips(chips)];
  // Dedupe field+operator+value triples to keep the URL/payload tidy when a
  // chip overlaps with the saved view (e.g., both add status=Delayed).
  const seen = new Set<string>();
  const result: FieldFilter[] = [];
  for (const f of merged) {
    const key = `${f.field}|${f.operator}|${JSON.stringify(f.value)}`;
    if (seen.has(key)) continue;
    seen.add(key);
    result.push(f);
  }
  return result;
}
