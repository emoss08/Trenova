import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { parseAsInteger, parseAsStringLiteral } from "nuqs";
import { CLEARED_INBOX_STATE, inboxSearchParams, type InboxScope } from "./inbox/inbox-filters";

export const MONITORING_TABS = ["inbox", "review", "enrollments", "usage"] as const;

export type MonitoringTab = (typeof MONITORING_TABS)[number];

export const monitoringTabParser = parseAsStringLiteral(MONITORING_TABS).withDefault("inbox");

export const monitoringPageSearchParams = {
  ...searchParamsParser,
  ...inboxSearchParams,
  tab: monitoringTabParser,
  month: parseAsInteger,
};

export const CLEARED_TAB_STATE = {
  ...CLEARED_INBOX_STATE,
  pageIndex: null,
  query: null,
  fieldFilters: null,
  filterGroups: null,
  sort: null,
  panelType: null,
  panelEntityId: null,
  entityId: null,
  modalType: null,
  month: null,
} as const;

export type MonitoringNavigation = {
  tab: MonitoringTab;
  scope?: InboxScope;
};

export function isMonitoringTab(value: unknown): value is MonitoringTab {
  return typeof value === "string" && (MONITORING_TABS as readonly string[]).includes(value);
}
