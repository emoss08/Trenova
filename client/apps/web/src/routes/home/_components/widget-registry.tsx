import type { HomeWidget } from "@/lib/graphql/home-layout";
import { lazy, type ComponentType, type LazyExoticComponent } from "react";
import type { HomeData } from "./use-home-data";

export type WidgetProps = {
  widget: HomeWidget;
  data: HomeData;
};

type WidgetComponent = ComponentType<WidgetProps>;

const work = () => import("./widgets/work-widgets");
const pulse = () => import("./widgets/pulse-widgets");
const insight = () => import("./widgets/insight-widgets");
const orientation = () => import("./widgets/orientation-widgets");
const comms = () => import("./widgets/comms-widgets");
const workerAttention = () => import("./widgets/worker-attention-widget");
const aiInsights = () => import("./widgets/ai-insights-widget");

/**
 * Maps a widget key to the code that draws it. The keys are the same strings the
 * Go catalog defines; a test asserts every catalog key resolves here, so shipping
 * a widget the server offers but the client cannot draw fails the build rather
 * than leaving a blank tile on someone's home screen.
 *
 * Each entry is a loader, not the component: a home screen reads the code of the
 * widgets on it and nothing else, so the chart library arrives with the first
 * chart and the report renderer with the first report tile. Widgets that share a
 * module share its one download.
 */
export const WIDGET_LOADERS: Record<string, () => Promise<WidgetComponent>> = {
  // Work
  attention: () => work().then((module) => module.AttentionWidget),
  unassigned: () => work().then((module) => module.UnassignedWidget),
  exceptions: () => work().then((module) => module.ExceptionsWidget),
  "detention-watch": () => work().then((module) => module.DetentionWatchWidget),
  "tomorrows-pickups": () => work().then((module) => module.TomorrowsPickupsWidget),
  "my-approvals": () => work().then((module) => module.MyApprovalsWidget),
  "billing-queue": () => work().then((module) => module.BillingQueueWidget),
  "service-failures": () => work().then((module) => module.ServiceFailuresWidget),
  "edi-attention": () => work().then((module) => module.EDIAttentionWidget),
  "expiring-credentials": () => work().then((module) => module.ExpiringCredentialsWidget),
  "worker-attention": () => workerAttention().then((module) => module.WorkerAttentionWidget),

  // Pulse
  kpi: () => pulse().then((module) => module.KPIWidget),
  "kpi-row": () => pulse().then((module) => module.KPIRowWidget),
  "ar-snapshot": () => pulse().then((module) => module.ARSnapshotWidget),
  "revenue-trend": () => pulse().then((module) => module.RevenueTrendWidget),
  "fleet-status": () => pulse().then((module) => module.FleetStatusWidget),
  "on-time-goal": () => pulse().then((module) => module.OnTimeGoalWidget),

  // Insight
  "ai-insights": () => aiInsights().then((module) => module.AIInsightsWidget),
  report: () => insight().then((module) => module.ReportWidget),
  "dashboard-link": () => insight().then((module) => module.DashboardLinkWidget),
  "lane-heatmap": () => insight().then((module) => module.LaneHeatmapWidget),
  "customer-mix": () => insight().then((module) => module.CustomerMixWidget),

  // Orientation
  "quick-actions": () => orientation().then((module) => module.QuickActionsWidget),
  favorites: () => orientation().then((module) => module.FavoritesWidget),
  "jump-back-in": () => orientation().then((module) => module.JumpBackInWidget),
  "saved-views": () => orientation().then((module) => module.SavedViewsWidget),
  activity: () => orientation().then((module) => module.ActivityWidget),
  notifications: () => orientation().then((module) => module.NotificationsWidget),

  // Comms
  announcement: () => comms().then((module) => module.AnnouncementWidget),
  map: () => comms().then((module) => module.MapWidget),
};

/**
 * The component for each key, read when it is first drawn. Built once at module
 * scope: a lazy component made during a render is a new component every render,
 * and would reload its widget each time the canvas drew.
 */
export const WIDGET_COMPONENTS: Record<
  string,
  LazyExoticComponent<WidgetComponent>
> = Object.fromEntries(
  Object.entries(WIDGET_LOADERS).map(([key, load]) => [
    key,
    lazy(() => load().then((component) => ({ default: component }))),
  ]),
);

export function widgetComponentFor(key: string): LazyExoticComponent<WidgetComponent> | null {
  return WIDGET_COMPONENTS[key] ?? null;
}
