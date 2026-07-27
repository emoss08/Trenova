import type { HomeWidget } from "@/lib/graphql/home-layout";
import type { ComponentType } from "react";
import type { HomeData } from "./use-home-data";
import {
  AttentionWidget,
  BillingQueueWidget,
  DetentionWatchWidget,
  EDIAttentionWidget,
  ExceptionsWidget,
  ExpiringCredentialsWidget,
  MyApprovalsWidget,
  ServiceFailuresWidget,
  TomorrowsPickupsWidget,
  UnassignedWidget,
} from "./widgets/work-widgets";
import {
  ARSnapshotWidget,
  FleetStatusWidget,
  KPIRowWidget,
  KPIWidget,
  OnTimeGoalWidget,
  RevenueTrendWidget,
} from "./widgets/pulse-widgets";
import {
  CustomerMixWidget,
  DashboardLinkWidget,
  LaneHeatmapWidget,
  ReportWidget,
} from "./widgets/insight-widgets";
import {
  ActivityWidget,
  FavoritesWidget,
  JumpBackInWidget,
  NotificationsWidget,
  QuickActionsWidget,
  SavedViewsWidget,
} from "./widgets/orientation-widgets";
import { AnnouncementWidget, MapWidget } from "./widgets/comms-widgets";

export type WidgetProps = {
  widget: HomeWidget;
  data: HomeData;
};

/**
 * Maps a widget key to the component that draws it. The keys are the same
 * strings the Go catalog defines; a test asserts every catalog key resolves
 * here, so shipping a widget the server offers but the client cannot draw fails
 * the build rather than leaving a blank tile on someone's home screen.
 */
export const WIDGET_COMPONENTS: Record<string, ComponentType<WidgetProps>> = {
  // Work
  attention: AttentionWidget,
  unassigned: UnassignedWidget,
  exceptions: ExceptionsWidget,
  "detention-watch": DetentionWatchWidget,
  "tomorrows-pickups": TomorrowsPickupsWidget,
  "my-approvals": MyApprovalsWidget,
  "billing-queue": BillingQueueWidget,
  "service-failures": ServiceFailuresWidget,
  "edi-attention": EDIAttentionWidget,
  "expiring-credentials": ExpiringCredentialsWidget,

  // Pulse
  kpi: KPIWidget,
  "kpi-row": KPIRowWidget,
  "ar-snapshot": ARSnapshotWidget,
  "revenue-trend": RevenueTrendWidget,
  "fleet-status": FleetStatusWidget,
  "on-time-goal": OnTimeGoalWidget,

  // Insight
  report: ReportWidget,
  "dashboard-link": DashboardLinkWidget,
  "lane-heatmap": LaneHeatmapWidget,
  "customer-mix": CustomerMixWidget,

  // Orientation
  "quick-actions": QuickActionsWidget,
  favorites: FavoritesWidget,
  "jump-back-in": JumpBackInWidget,
  "saved-views": SavedViewsWidget,
  activity: ActivityWidget,
  notifications: NotificationsWidget,

  // Comms
  announcement: AnnouncementWidget,
  map: MapWidget,
};

export function widgetComponentFor(key: string): ComponentType<WidgetProps> | null {
  return WIDGET_COMPONENTS[key] ?? null;
}
