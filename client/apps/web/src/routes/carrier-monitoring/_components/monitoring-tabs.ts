import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { parseAsInteger, parseAsStringLiteral } from "nuqs";
import { enrollmentViewParser } from "./enrollment-filter";
import { eventInboxSearchParams } from "./event-inbox-filter";

export const MONITORING_TABS = ["inbox", "review", "enrollments", "usage"] as const;

export type MonitoringTab = (typeof MONITORING_TABS)[number];

export const monitoringTabParser = parseAsStringLiteral(MONITORING_TABS).withDefault("inbox");

export const monitoringPageSearchParams = {
  ...searchParamsParser,
  ...eventInboxSearchParams,
  tab: monitoringTabParser,
  enrollmentView: enrollmentViewParser,
  month: parseAsInteger,
};

export const CLEARED_TAB_STATE = {
  pageIndex: null,
  query: null,
  fieldFilters: null,
  filterGroups: null,
  sort: null,
  panelType: null,
  panelEntityId: null,
  entityId: null,
  modalType: null,
  scope: null,
  severity: null,
  category: null,
  carrier: null,
  enrollmentView: null,
  month: null,
} as const;
