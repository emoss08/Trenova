import type { DeltaTone } from "@/components/kpi/tone";
import type { ShipmentAnalyticsData } from "@/lib/shipment-analytics";
import { formatCompactCurrency } from "@trenova/shared/lib/utils";
import type { HomeData } from "./use-home-data";

/**
 * How a metric wants to be drawn. A percentage against a target reads best as a
 * ring; a percentage we want to push down reads best as a goal bar; a currency
 * total with a trend reads best as a hero. The shape is a property of the
 * measure, not of the tile, so a KPI widget never has to be told.
 */
export type MetricShape = "hero" | "ring" | "goal" | "stat";

export type MetricValue = {
  key: string;
  label: string;
  shape: MetricShape;
  /** Preformatted for display; the raw number is kept for gauges. */
  display: string;
  raw: number;
  unit?: string;
  target?: number;
  delta?: number;
  deltaLabel?: string;
  deltaTone?: DeltaTone;
  sub?: string;
  sparkline?: number[];
  href: string;
};

const SHIPMENTS_HREF = "/shipment-management/shipments";
const RECEIVABLES_HREF = "/accounting/ar/aging";

function minorToMajor(minor: number | null | undefined): number {
  return (minor ?? 0) / 100;
}

function shipmentMetric(key: string, analytics: ShipmentAnalyticsData): MetricValue | null {
  switch (key) {
    case "activeShipments":
      return {
        key,
        label: "Active Shipments",
        shape: "hero",
        display: String(analytics.activeShipments.count),
        raw: analytics.activeShipments.count,
        delta: analytics.activeShipments.changeFromYesterday,
        deltaTone: "success",
        sparkline: analytics.activeShipments.sparkline.map((point) => point.value),
        href: SHIPMENTS_HREF,
      };
    case "revenueToday":
      return {
        key,
        label: "Revenue Today",
        shape: "hero",
        display: formatCompactCurrency(analytics.revenueToday.total),
        raw: analytics.revenueToday.total,
        delta: analytics.revenueToday.deltaPct,
        deltaLabel: "%",
        deltaTone: "success",
        sub: `RPM $${analytics.revenueToday.rpm.toFixed(2)}`,
        sparkline: analytics.revenueToday.sparkline.map((point) => point.value),
        href: SHIPMENTS_HREF,
      };
    case "onTimePercent":
      return {
        key,
        label: "On-Time",
        shape: "ring",
        display: analytics.onTimePercent.percent.toFixed(1),
        raw: analytics.onTimePercent.percent,
        unit: "%",
        target: analytics.onTimePercent.target,
        delta: analytics.onTimePercent.deltaPp,
        deltaLabel: "pp",
        deltaTone: analytics.onTimePercent.deltaPp >= 0 ? "success" : "danger",
        sub: `Target ${analytics.onTimePercent.target}% · 7-day ${analytics.onTimePercent.sevenDayPercent.toFixed(1)}%`,
        href: SHIPMENTS_HREF,
      };
    case "emptyMilePercent":
      // Lower is better, so the delta tone inverts: a fall is the good news.
      return {
        key,
        label: "Empty Mile",
        shape: "goal",
        display: analytics.emptyMilePercent.percent.toFixed(1),
        raw: analytics.emptyMilePercent.percent,
        unit: "%",
        target: analytics.emptyMilePercent.target,
        delta: analytics.emptyMilePercent.deltaPp,
        deltaLabel: "pp",
        deltaTone: analytics.emptyMilePercent.deltaPp <= 0 ? "success" : "danger",
        sub: `${analytics.emptyMilePercent.emptyMiles.toLocaleString()} deadhead miles`,
        href: SHIPMENTS_HREF,
      };
    case "atRisk":
      return {
        key,
        label: "At Risk",
        shape: "stat",
        display: String(analytics.atRisk.count),
        raw: analytics.atRisk.count,
        delta: analytics.atRisk.delta,
        deltaTone: "danger",
        sub: `${analytics.atRisk.etaSlip} ETA slip · ${analytics.atRisk.weather} weather`,
        href: SHIPMENTS_HREF,
      };
    case "unassigned":
      return {
        key,
        label: "Unassigned",
        shape: "stat",
        display: String(analytics.unassigned.count),
        raw: analytics.unassigned.count,
        delta: analytics.unassigned.delta,
        deltaTone: "warning",
        sub: `${formatCompactCurrency(analytics.unassigned.revenueWaiting)} waiting`,
        href: SHIPMENTS_HREF,
      };
    case "readyToDispatch":
      return {
        key,
        label: "Ready to Dispatch",
        shape: "stat",
        display: String(analytics.readyToDispatch.count),
        raw: analytics.readyToDispatch.count,
        delta: analytics.readyToDispatch.delta,
        deltaTone: "brand",
        sub: `${analytics.readyToDispatch.driverReady} driver-ready`,
        href: SHIPMENTS_HREF,
      };
    case "marginPercent":
      return {
        key,
        label: "Margin",
        shape: "stat",
        display: `${analytics.profitability.avgMarginPct.toFixed(1)}%`,
        raw: analytics.profitability.avgMarginPct,
        sub: `${analytics.profitability.unprofitableCount} unprofitable loads`,
        href: SHIPMENTS_HREF,
      };
    case "revenuePerMile":
      return {
        key,
        label: "Revenue / Mile",
        shape: "stat",
        display: `$${analytics.revenueToday.rpm.toFixed(2)}`,
        raw: analytics.revenueToday.rpm,
        href: SHIPMENTS_HREF,
      };
    case "costPerMile":
      return {
        key,
        label: "Cost / Mile",
        shape: "stat",
        display: `$${analytics.profitability.avgCpm.toFixed(2)}`,
        raw: analytics.profitability.avgCpm,
        sub: `${analytics.profitability.totalMiles.toLocaleString()} miles`,
        href: SHIPMENTS_HREF,
      };
    default:
      return null;
  }
}

function receivablesMetric(key: string, receivables: HomeData["receivables"]): MetricValue | null {
  if (!receivables) return null;
  const overview = receivables.overview;

  switch (key) {
    case "arTotalOpen":
      return {
        key,
        label: "Open Receivables",
        shape: "hero",
        display: formatCompactCurrency(minorToMajor(overview?.totalOpenMinor)),
        raw: minorToMajor(overview?.totalOpenMinor),
        sub: `${overview?.openInvoiceCount ?? 0} invoices`,
        href: RECEIVABLES_HREF,
      };
    case "arOverdue":
      return {
        key,
        label: "Overdue",
        shape: "stat",
        display: formatCompactCurrency(minorToMajor(overview?.overdueMinor)),
        raw: minorToMajor(overview?.overdueMinor),
        deltaTone: "danger",
        sub: `${overview?.overdueInvoiceCount ?? 0} invoices · avg ${Math.round(overview?.avgDaysPastDue ?? 0)}d late`,
        href: RECEIVABLES_HREF,
      };
    case "arDaysSalesOutstanding":
      return {
        key,
        label: "Days Sales Outstanding",
        shape: "stat",
        display: `${(receivables.currentDsoDays ?? 0).toFixed(1)}`,
        raw: receivables.currentDsoDays ?? 0,
        unit: "d",
        delta: receivables.dsoDeltaDays ?? undefined,
        deltaLabel: "d",
        // Collecting sooner is better, so a falling DSO is the good direction.
        deltaTone: (receivables.dsoDeltaDays ?? 0) <= 0 ? "success" : "danger",
        href: RECEIVABLES_HREF,
      };
    default:
      return null;
  }
}

/**
 * Resolves a metric key against whichever payload owns it. Returns null when
 * the measure is unknown or its data has not arrived, so a tile can show a
 * skeleton instead of a zero it would have to correct.
 */
export function resolveMetric(key: string, data: HomeData): MetricValue | null {
  const fromShipments = shipmentMetric(key, data.shipmentAnalytics);
  if (fromShipments) {
    return data.shipmentAnalyticsLoading ? null : fromShipments;
  }
  return receivablesMetric(key, data.receivables);
}
