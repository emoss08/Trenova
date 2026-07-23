import { REPORT_CATEGORY_CHOICES, REPORT_DEFINITION_STATUS_LABELS } from "@/types/report";
import { parseAsString, parseAsStringLiteral } from "nuqs";

export const reportTabs = ["library", "gallery"] as const;
export type ReportTab = (typeof reportTabs)[number];

export const reportSortOrders = ["name_asc", "name_desc", "last_run"] as const;
export type ReportSortOrder = (typeof reportSortOrders)[number];

export const reportStatusFilters = [
  "all",
  "draft",
  "active",
  "archived",
  "needs_attention",
] as const;
export type ReportStatusFilter = (typeof reportStatusFilters)[number];

export const reportsPageSearchParamsParser = {
  tab: parseAsStringLiteral(reportTabs).withDefault("library"),
  query: parseAsString.withDefault(""),
  sortBy: parseAsStringLiteral(reportSortOrders).withDefault("name_asc"),
  category: parseAsString.withDefault("all"),
  status: parseAsStringLiteral(reportStatusFilters).withDefault("all"),
};

export const LIBRARY_SORT_CHOICES: { value: ReportSortOrder; label: string }[] = [
  { value: "name_asc", label: "Name (A-Z)" },
  { value: "name_desc", label: "Name (Z-A)" },
  { value: "last_run", label: "Recently Run" },
];

export const GALLERY_SORT_CHOICES: { value: ReportSortOrder; label: string }[] = [
  { value: "name_asc", label: "Name (A-Z)" },
  { value: "name_desc", label: "Name (Z-A)" },
];

export const REPORT_CATEGORY_FILTER_CHOICES: { value: string; label: string }[] = [
  { value: "all", label: "All Categories" },
  ...REPORT_CATEGORY_CHOICES,
];

export const REPORT_STATUS_FILTER_CHOICES: { value: string; label: string }[] = [
  { value: "all", label: "All Statuses" },
  ...reportStatusFilters
    .filter((status) => status !== "all")
    .map((status) => ({
      value: status,
      label: REPORT_DEFINITION_STATUS_LABELS[status] ?? status,
    })),
];

const categoryOrder = new Map(
  REPORT_CATEGORY_CHOICES.map((choice, index) => [choice.value, index]),
);
const categoryLabels = new Map(
  REPORT_CATEGORY_CHOICES.map((choice) => [choice.value, choice.label]),
);

export function reportCategoryLabel(category: string): string {
  return categoryLabels.get(category) ?? (category || "Uncategorized");
}

export type ReportCategoryGroup<T> = {
  key: string;
  label: string;
  startIndex: number;
  items: T[];
};

export function groupByReportCategory<T extends { category: string }>(
  items: T[],
): ReportCategoryGroup<T>[] {
  const groups = new Map<string, ReportCategoryGroup<T>>();
  for (const item of items) {
    const key = item.category || "uncategorized";
    const group = groups.get(key);
    if (group) {
      group.items.push(item);
      continue;
    }
    groups.set(key, { key, label: reportCategoryLabel(key), startIndex: 0, items: [item] });
  }
  const ordered = Array.from(groups.values()).sort((left, right) => {
    const leftOrder = categoryOrder.get(left.key) ?? Number.MAX_SAFE_INTEGER;
    const rightOrder = categoryOrder.get(right.key) ?? Number.MAX_SAFE_INTEGER;
    return leftOrder === rightOrder
      ? left.label.localeCompare(right.label)
      : leftOrder - rightOrder;
  });
  let index = 0;
  for (const group of ordered) {
    group.startIndex = index;
    index += group.items.length;
  }
  return ordered;
}

export function compareReportsBySort<T extends { name: string; lastRunAt?: number | null }>(
  sortBy: ReportSortOrder,
): (left: T, right: T) => number {
  return (left, right) => {
    if (sortBy === "last_run") {
      const byLastRun = (right.lastRunAt ?? 0) - (left.lastRunAt ?? 0);
      if (byLastRun !== 0) return byLastRun;
      return left.name.localeCompare(right.name);
    }
    const comparison = left.name.localeCompare(right.name);
    return sortBy === "name_asc" ? comparison : -comparison;
  };
}
