import { analytics } from "@/lib/queries/analytics";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  BoltIcon,
  CheckIcon,
  ClockIcon,
  DollarSignIcon,
  FlagIcon,
  RouteIcon,
  ShieldIcon,
  TruckIcon,
} from "lucide-react";
import {
  type DeepPartial,
  type ShipmentAnalyticsData,
  mergeShipmentAnalyticsWithDefaults,
} from "../mock-data";
import { KpiGoalBar } from "./kpi-goal-bar";
import { KpiHero } from "./kpi-hero";
import { KpiInfoPopover } from "./kpi-info-popover";
import { KpiRing } from "./kpi-ring";
import { KpiStat } from "./kpi-stat";
import { KpiWatchlist } from "./kpi-watchlist";

const ICON_PROPS = { className: "size-[11px]" } as const;
const KPI_INFO = {
  revenueToday: {
    title: "Revenue today",
    description: "Revenue activity for the selected business unit in the selected timezone.",
    rows: [
      {
        label: "Total",
        value: "Sum of rated shipment revenue created from local midnight to now.",
      },
      { label: "Delta", value: "Compares today so far with the same elapsed window yesterday." },
      { label: "RPM", value: "Revenue divided by loaded miles for shipments in today's window." },
      {
        label: "CPM",
        value:
          "Fleet cost per mile resolved from your cost profile — benchmarks, overrides, GL actuals, and live fuel.",
      },
      {
        label: "Margin",
        value:
          "Trailing 30-day revenue minus estimated cost, over revenue. Includes unprofitable-load count in the fleet summary.",
      },
    ],
  },
  activeShipments: {
    title: "Active shipments",
    description: "A current operational-status snapshot, not a total shipment count.",
    rows: [
      {
        label: "Included",
        value:
          "New, Partially Assigned, Assigned, In Transit, Delayed, and Partially Completed shipments.",
      },
      {
        label: "Excluded",
        value: "Completed, Ready to Invoice, Invoiced, and Canceled shipments.",
      },
      {
        label: "Trend",
        value: "Delta and sparkline use shipments created today versus the same window yesterday.",
      },
    ],
  },
  onTimePercent: {
    title: "On-time",
    description: "Service performance for completed shipment activity.",
    rows: [
      {
        label: "Formula",
        value: "On-time completed shipments divided by total completed shipments.",
      },
      { label: "Delta", value: "Percentage-point change from the same elapsed window yesterday." },
      { label: "7-day", value: "Trailing seven full local days." },
    ],
  },
  emptyMilePercent: {
    title: "Empty mile %",
    description: "Deadhead exposure for shipment mileage in the local-day window.",
    rows: [
      { label: "Formula", value: "Empty miles divided by total miles." },
      { label: "Delta", value: "Percentage-point change from the same elapsed window yesterday." },
      { label: "Goal", value: "Lower is better." },
    ],
  },
  tenderAccept: {
    title: "Tender accept",
    description: "Carrier tender response rate.",
    rows: [
      { label: "Formula", value: "Accepted tenders divided by accepted plus declined tenders." },
      { label: "Delta", value: "Percentage-point change against the comparison window." },
      { label: "Source", value: "Uses frontend defaults until tender analytics are connected." },
    ],
  },
  atRisk: {
    title: "At-risk",
    description: "Active shipments with signals that may require dispatch attention.",
    rows: [
      { label: "ETA slip", value: "Active delayed or late shipments." },
      { label: "Weather", value: "Active shipments intersecting active weather alerts." },
      { label: "Reefer", value: "Temperature-controlled active shipments in an at-risk state." },
    ],
  },
  unassigned: {
    title: "Unassigned",
    description: "Active shipments without an active, non-canceled assignment.",
    rows: [
      { label: "Count", value: "Active shipments with no usable assignment." },
      { label: "Revenue", value: "Rated revenue attached to those waiting shipments." },
      {
        label: "Delta",
        value: "Today-created unassigned shipments minus yesterday's matching window.",
      },
    ],
  },
  readyToDispatch: {
    title: "Ready to dispatch",
    description: "Assigned shipments that are ready for dispatch execution.",
    rows: [
      { label: "Count", value: "Assigned shipments with dispatch-ready assignment context." },
      { label: "Unassigned", value: "Ready shipments still missing assignment coverage." },
      { label: "Driver-ready", value: "Ready shipments with driver assignment in place." },
    ],
  },
  hosNearLimit: {
    title: "HOS near limit",
    description: "Drivers nearing hours-of-service constraints.",
    rows: [
      { label: "Warning", value: "Driver has limited hours remaining." },
      { label: "Danger", value: "Driver is critically close to the limit." },
      { label: "Source", value: "Uses frontend defaults until HOS analytics are connected." },
    ],
  },
  detentionWatchlist: {
    title: "Detention dwell > 2h",
    description: "Stops currently dwelling beyond detention-watch thresholds.",
    rows: [
      { label: "Included", value: "Shipment stops with dwell time over two hours." },
      { label: "Ordering", value: "Longest dwell first." },
      { label: "Tone", value: "Warning over two hours; danger over four hours." },
    ],
  },
};

function profitabilitySub(merged: ShipmentAnalyticsData): string {
  const rpm = `RPM $${merged.revenueToday.rpm.toFixed(2)}`;
  const { avgCpm, avgMarginPct, hasMargin } = merged.profitability;
  if (!hasMargin) {
    return avgCpm > 0 ? `${rpm}  ·  CPM $${avgCpm.toFixed(2)}` : rpm;
  }
  return `${rpm}  ·  CPM $${avgCpm.toFixed(2)}  ·  Margin ${avgMarginPct.toFixed(1)}%`;
}

export default function KpiRail() {
  const { data } = useSuspenseQuery(analytics.get("shipment-management"));
  const merged = mergeShipmentAnalyticsWithDefaults(data as DeepPartial<ShipmentAnalyticsData>);

  const revenueSparkline = merged.revenueToday.sparkline.map((point) => point.value);
  const activeSparkline = merged.activeShipments.sparkline.map((point) => point.value);
  const breakdown = merged.activeShipments.breakdown;

  return (
    <div className="grid grid-cols-12 gap-2 pt-1">
      <KpiHero
        label="Revenue today"
        value={`$${formatCompact(merged.revenueToday.total)}`}
        delta={merged.revenueToday.deltaPct}
        deltaLabel="%"
        deltaTone="success"
        sub={profitabilitySub(merged)}
        sparkData={revenueSparkline}
        sparkColor="var(--success)"
        icon={<DollarSignIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.revenueToday} />}
        span={3}
      />
      <KpiHero
        label="Active shipments"
        value={String(merged.activeShipments.count)}
        delta={merged.activeShipments.changeFromYesterday}
        deltaTone="success"
        sparkData={activeSparkline}
        sparkColor="var(--brand)"
        icon={<TruckIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.activeShipments} />}
        breakdown={[
          { label: "In-transit", value: breakdown.inTransit, color: "var(--brand)" },
          { label: "At-risk", value: breakdown.atRisk, color: "var(--destructive)" },
          { label: "Loading", value: breakdown.loading, color: "var(--info)" },
          { label: "Done", value: breakdown.done, color: "var(--success)" },
        ]}
        span={3}
      />
      <KpiRing
        label="On-time"
        value={merged.onTimePercent.percent.toFixed(1)}
        unit="%"
        target={merged.onTimePercent.target}
        ringValue={merged.onTimePercent.percent}
        delta={merged.onTimePercent.deltaPp}
        deltaLabel="pp"
        deltaTone={merged.onTimePercent.deltaPp >= 0 ? "success" : "danger"}
        sub={`Target ${merged.onTimePercent.target}%  ·  7-day ${merged.onTimePercent.sevenDayPercent.toFixed(1)}%`}
        icon={<ClockIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.onTimePercent} />}
        span={2}
      />
      <KpiGoalBar
        label="Empty mile %"
        value={merged.emptyMilePercent.percent.toFixed(1)}
        unit="%"
        target={merged.emptyMilePercent.target}
        actual={merged.emptyMilePercent.percent}
        max={20}
        delta={merged.emptyMilePercent.deltaPp}
        deltaLabel="pp"
        deltaTone={merged.emptyMilePercent.deltaPp <= 0 ? "success" : "danger"}
        sub={`${merged.emptyMilePercent.emptyMiles.toLocaleString()} deadhead miles · goal <${merged.emptyMilePercent.target}%`}
        icon={<RouteIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.emptyMilePercent} />}
        span={2}
      />
      <KpiRing
        label="Tender accept"
        value={merged.tenderAccept.percent.toFixed(1)}
        unit="%"
        target={merged.tenderAccept.target}
        ringValue={merged.tenderAccept.percent}
        delta={merged.tenderAccept.deltaPp}
        deltaLabel="pp"
        deltaTone={merged.tenderAccept.deltaPp >= 0 ? "success" : "danger"}
        sub={`${merged.tenderAccept.accepted} accepted · ${merged.tenderAccept.declined} declined`}
        icon={<CheckIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.tenderAccept} />}
        span={2}
      />

      <KpiStat
        label="At-risk"
        value={String(merged.atRisk.count)}
        delta={merged.atRisk.delta}
        tone="danger"
        sub={`${merged.atRisk.etaSlip} ETA slip · ${merged.atRisk.weather} weather · ${merged.atRisk.reefer} reefer`}
        icon={<AlertTriangleIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.atRisk} />}
        span={2}
      />
      <KpiStat
        label="Unassigned"
        value={String(merged.unassigned.count)}
        delta={merged.unassigned.delta}
        tone="warning"
        sub={`$${merged.unassigned.revenueWaiting.toLocaleString()} revenue waiting`}
        icon={<FlagIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.unassigned} />}
        span={2}
      />
      <KpiStat
        label="Ready to dispatch"
        value={String(merged.readyToDispatch.count)}
        delta={merged.readyToDispatch.delta}
        tone="brand"
        sub={`${merged.readyToDispatch.unassigned} unassigned · ${merged.readyToDispatch.driverReady} driver-ready`}
        icon={<BoltIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.readyToDispatch} />}
        span={2}
      />
      <KpiWatchlist
        label="HOS near limit"
        items={merged.hosNearLimit.items.map((item) => ({
          id: item.driverId,
          who: `${item.driverId} ${item.name}`,
          meta: item.hoursLeftLabel,
          tone: item.tone,
        }))}
        icon={<ShieldIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.hosNearLimit} />}
        span={3}
      />
      <KpiWatchlist
        label="Detention dwell > 2h"
        items={merged.detentionWatchlist.items.map((item) => ({
          id: item.shipmentId,
          who: `${item.shipmentId} ${item.customer}`,
          meta: item.dwellLabel,
          tone: item.tone,
        }))}
        icon={<ClockIcon {...ICON_PROPS} />}
        info={<KpiInfoPopover {...KPI_INFO.detentionWatchlist} />}
        span={3}
      />
    </div>
  );
}

function formatCompact(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return value.toLocaleString();
}
