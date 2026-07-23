import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { EDIMappingEntityType, EDIMappingResolution, EDITransfer } from "@trenova/shared/types/edi";

type LoadTenderStop = EDITransfer["tenderPayload"]["moves"][number]["stops"][number];
type LoadTenderCommodity = EDITransfer["tenderPayload"]["commodities"][number];
type LoadTenderCharge = EDITransfer["tenderPayload"]["additionalCharges"][number];

export type TargetLookupSelection = {
  targetId: string;
  targetLabel: string;
};

export function mappingKey(entityType: string, sourceId: string) {
  return `${entityType}:${sourceId}`;
}

export function findMapping(
  mappingRows: EDIMappingResolution[],
  entityType: EDIMappingEntityType,
  sourceId: string,
) {
  return mappingRows.find((row) => row.entityType === entityType && row.sourceId === sourceId);
}

export function formatStopName(stop: LoadTenderStop | undefined, mapping?: EDIMappingResolution) {
  if (!stop) return "Unknown stop";
  return firstDisplayValue(
    stop.locationLabel,
    joinDisplayParts(stop.locationCode, stop.locationName, " - "),
    stop.addressLine,
    mapping?.sourceLabel,
    "Unknown stop",
  );
}

export function formatStopAddress(stop: LoadTenderStop) {
  return firstDisplayValue(
    joinDisplayParts(
      stop.locationAddressLine1,
      stop.locationAddressLine2,
      joinDisplayParts(stop.locationCity, stop.locationStateCode, ", "),
      stop.locationPostalCode,
      ", ",
    ),
    stop.addressLine,
  );
}

export function formatCommodityName(
  commodity: LoadTenderCommodity,
  mapping?: EDIMappingResolution,
) {
  return firstDisplayValue(
    commodity.commodityLabel,
    commodity.commodityName,
    commodity.commodityDescription,
    mapping?.sourceLabel,
    "Unlabeled commodity",
  );
}

export function formatAccessorialName(charge: LoadTenderCharge, mapping?: EDIMappingResolution) {
  return firstDisplayValue(
    charge.accessorialLabel,
    joinDisplayParts(charge.accessorialCode, charge.accessorialDescription, " - "),
    charge.accessorialCode,
    charge.accessorialDescription,
    mapping?.sourceLabel,
    "Unlabeled charge",
  );
}

export function formatMappingDetail(mapping?: EDIMappingResolution) {
  if (mapping?.targetLabel) {
    return `Local record: ${mapping.targetLabel}`;
  }
  if (mapping?.resolved) {
    return "Mapped local record";
  }
  return "No mapping saved";
}

export function sourceValueLabel(label: string | null | undefined, id: string | null | undefined) {
  return firstDisplayValue(label, id ? "Unlabeled source value" : undefined, "-");
}

export function firstDisplayValue(...values: Array<string | null | undefined>) {
  return values.find((value) => value?.trim())?.trim() ?? "";
}

export function getTargetOptionValue(option: unknown) {
  if (!option || typeof option !== "object") {
    return "";
  }
  const record = option as Record<string, unknown>;
  return getStringValue(record.id) || getStringValue(record.value);
}

export function getTargetOptionDescription(option: unknown) {
  if (!option || typeof option !== "object") {
    return "";
  }
  const record = option as Record<string, unknown>;
  return getStringValue(record.description);
}

export function getTargetOptionLabel(option: unknown) {
  if (!option || typeof option !== "object") {
    return "";
  }
  const record = option as Record<string, unknown>;
  const codeName = joinDisplayParts(
    getStringValue(record.code),
    getStringValue(record.name),
    " - ",
  );
  return (
    codeName ||
    getStringValue(record.label) ||
    getStringValue(record.name) ||
    getStringValue(record.description) ||
    getTargetOptionValue(option)
  );
}

export function joinDisplayParts(...args: Array<string | null | undefined>) {
  const separator = args.pop() || " ";
  return args
    .map((part) => part?.trim())
    .filter(Boolean)
    .join(separator);
}

export function getStringValue(value: unknown) {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return value.toString();
  return "";
}

export function formatDecimalLike(value: unknown) {
  if (value === null || value === undefined) return "-";
  if (typeof value === "number") return value.toLocaleString();
  if (typeof value === "string") return value;
  if (typeof value === "object") {
    const record = value as Record<string, unknown>;
    return getStringValue(record.value) || getStringValue(record.String) || JSON.stringify(value);
  }
  return getStringValue(value) || "-";
}

export function formatNumber(value: number | null | undefined) {
  return typeof value === "number" ? value.toLocaleString() : "-";
}

export function formatWeight(value: number | null | undefined) {
  return typeof value === "number" ? `${value.toLocaleString()} lb` : "-";
}

export function formatUnix(value: number | null | undefined) {
  if (!value) return "-";
  return formatToUserTimezone(value);
}

export function formatWindow(start: number | null | undefined, end: number | null | undefined) {
  if (!start && !end) return "No appointment";
  if (!end) return formatUnix(start);
  return `${formatUnix(start)} - ${formatUnix(end)}`;
}
