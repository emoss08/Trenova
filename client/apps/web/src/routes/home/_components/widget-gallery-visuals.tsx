import { cn } from "@trenova/shared/lib/utils";
import {
  ActivityIcon,
  BellIcon,
  BookmarkIcon,
  CalendarClockIcon,
  ChartAreaIcon,
  ChartPieIcon,
  CheckCheckIcon,
  CircleAlertIcon,
  FileClockIcon,
  GaugeIcon,
  HistoryIcon,
  LayoutDashboardIcon,
  LayoutGridIcon,
  LayoutPanelTopIcon,
  MegaphoneIcon,
  MapPinnedIcon,
  NetworkIcon,
  PackageSearchIcon,
  ReceiptTextIcon,
  RouteIcon,
  ShieldAlertIcon,
  StarIcon,
  TableIcon,
  TargetIcon,
  TimerIcon,
  TriangleAlertIcon,
  TruckIcon,
  UsersIcon,
  WalletIcon,
  ZapIcon,
  type LucideIcon,
} from "lucide-react";

/**
 * The shape a widget draws on the canvas, reduced to the handful of silhouettes
 * a person actually recognises. The gallery draws the silhouette rather than a
 * screenshot: a screenshot goes stale the moment the widget changes and cannot
 * be rendered for a viewer whose data has not loaded, while the silhouette says
 * the one thing the decision turns on — is this a list, a number, or a picture.
 */
export type WidgetShape =
  | "grid"
  | "list"
  | "metric"
  | "metricStrip"
  | "chart"
  | "gauge"
  | "stats"
  | "table"
  | "heatmap"
  | "donut"
  | "map"
  | "text"
  | "link";

export type WidgetVisual = {
  icon: LucideIcon;
  shape: WidgetShape;
};

const FALLBACK: WidgetVisual = { icon: LayoutGridIcon, shape: "list" };

/**
 * Keyed by the widget keys the Go catalog defines. A key that reaches a client
 * too old to know it falls back to a neutral list card rather than rendering
 * nothing, because the gallery must stay usable across a version skew.
 */
const WIDGET_VISUALS: Record<string, WidgetVisual> = {
  // Work
  attention: { icon: TriangleAlertIcon, shape: "grid" },
  unassigned: { icon: PackageSearchIcon, shape: "list" },
  exceptions: { icon: ShieldAlertIcon, shape: "list" },
  "detention-watch": { icon: TimerIcon, shape: "list" },
  "tomorrows-pickups": { icon: CalendarClockIcon, shape: "list" },
  "my-approvals": { icon: CheckCheckIcon, shape: "list" },
  "billing-queue": { icon: ReceiptTextIcon, shape: "list" },
  "service-failures": { icon: CircleAlertIcon, shape: "list" },
  "edi-attention": { icon: NetworkIcon, shape: "list" },
  "expiring-credentials": { icon: FileClockIcon, shape: "list" },
  "worker-attention": { icon: UsersIcon, shape: "grid" },

  // Pulse
  kpi: { icon: GaugeIcon, shape: "metric" },
  "kpi-row": { icon: LayoutPanelTopIcon, shape: "metricStrip" },
  "ar-snapshot": { icon: WalletIcon, shape: "stats" },
  "revenue-trend": { icon: ChartAreaIcon, shape: "chart" },
  "fleet-status": { icon: TruckIcon, shape: "stats" },
  "on-time-goal": { icon: TargetIcon, shape: "gauge" },

  // Insight
  report: { icon: TableIcon, shape: "table" },
  "dashboard-link": { icon: LayoutDashboardIcon, shape: "link" },
  "lane-heatmap": { icon: RouteIcon, shape: "heatmap" },
  "customer-mix": { icon: ChartPieIcon, shape: "donut" },

  // Orientation
  "quick-actions": { icon: ZapIcon, shape: "grid" },
  favorites: { icon: StarIcon, shape: "list" },
  "jump-back-in": { icon: HistoryIcon, shape: "list" },
  "saved-views": { icon: BookmarkIcon, shape: "list" },
  activity: { icon: ActivityIcon, shape: "list" },
  notifications: { icon: BellIcon, shape: "list" },

  // Comms
  announcement: { icon: MegaphoneIcon, shape: "text" },
  map: { icon: MapPinnedIcon, shape: "map" },
};

export function widgetVisualFor(key: string): WidgetVisual {
  return WIDGET_VISUALS[key] ?? FALLBACK;
}

const GHOST = "bg-foreground/[0.09] dark:bg-foreground/[0.14]";
const GHOST_STRONG = "bg-foreground/20";
const GHOST_BRAND = "bg-brand/35";

function Line({ className }: { className?: string }) {
  return <span className={cn("block h-1 rounded-full", GHOST, className)} />;
}

function Cell({ className }: { className?: string }) {
  return (
    <span className={cn("border-foreground/10 flex flex-col gap-1 rounded border p-1", className)}>
      <span className={cn("block h-1 w-2/3 rounded-full", GHOST)} />
      <span className={cn("block h-1.5 w-1/2 rounded-full", GHOST_STRONG)} />
    </span>
  );
}

function Row({ share = 60 }: { share?: number }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className={cn("size-1.5 shrink-0 rounded-full", GHOST_STRONG)} />
      <span className={cn("block h-1 rounded-full", GHOST)} style={{ width: `${share}%` }} />
      <span className={cn("ml-auto block h-1 w-3 shrink-0 rounded-full", GHOST)} />
    </span>
  );
}

const BAR_HEIGHTS = [40, 62, 48, 78, 58, 92, 70];
const HEAT_OPACITIES = [0.08, 0.22, 0.4, 0.14, 0.55, 0.1, 0.3, 0.7, 0.18, 0.12, 0.45, 0.26];

/**
 * A faint drawing of what the widget puts on the canvas. Decoration for the eye
 * only — the label and description carry the meaning, so the whole thing is
 * hidden from assistive technology.
 */
export function WidgetSketch({ shape, className }: { shape: WidgetShape; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "bg-muted/40 group-hover/widget-card:bg-muted/70 block h-14 overflow-hidden rounded-md p-2 transition-colors",
        className,
      )}
    >
      <SketchBody shape={shape} />
    </span>
  );
}

function SketchBody({ shape }: { shape: WidgetShape }) {
  switch (shape) {
    case "grid":
      return (
        <span className="grid h-full grid-cols-3 grid-rows-2 gap-1">
          {Array.from({ length: 6 }, (_, index) => (
            <Cell key={index} />
          ))}
        </span>
      );

    case "list":
      return (
        <span className="flex h-full flex-col justify-between">
          <Row share={62} />
          <Row share={48} />
          <Row share={70} />
          <Row share={40} />
        </span>
      );

    case "metric":
      return (
        <span className="flex h-full flex-col justify-between">
          <Line className="w-1/3" />
          <span className={cn("block h-4 w-1/2 rounded", GHOST_STRONG)} />
          <span className="flex items-end gap-0.5">
            {BAR_HEIGHTS.map((height, index) => (
              <span
                key={index}
                className={cn("w-1 rounded-sm", GHOST_BRAND)}
                style={{ height: `${height / 10}px` }}
              />
            ))}
          </span>
        </span>
      );

    case "metricStrip":
      return (
        <span className="grid h-full grid-cols-4 gap-1">
          {Array.from({ length: 4 }, (_, index) => (
            <Cell key={index} className="justify-center" />
          ))}
        </span>
      );

    case "chart":
      return (
        <span className="flex h-full items-end gap-1">
          {BAR_HEIGHTS.map((height, index) => (
            <span
              key={index}
              className={cn("flex-1 rounded-t-sm", index % 2 === 0 ? GHOST_BRAND : GHOST)}
              style={{ height: `${height}%` }}
            />
          ))}
        </span>
      );

    case "gauge":
      return (
        <span className="flex h-full items-center justify-center">
          <svg viewBox="0 0 48 26" className="h-full">
            <path
              d="M4 24a20 20 0 0 1 40 0"
              className="stroke-foreground/12 fill-none"
              strokeWidth={5}
              strokeLinecap="round"
            />
            <path
              d="M4 24a20 20 0 0 1 34-14"
              className="stroke-brand/45 fill-none"
              strokeWidth={5}
              strokeLinecap="round"
            />
          </svg>
        </span>
      );

    case "stats":
      return (
        <span className="grid h-full grid-cols-3 items-center gap-2">
          {[70, 45, 60].map((width, index) => (
            <span key={index} className="flex flex-col gap-1">
              <span className={cn("block h-1 rounded-full", GHOST)} style={{ width: "60%" }} />
              <span
                className={cn("block h-2.5 rounded", GHOST_STRONG)}
                style={{ width: `${width}%` }}
              />
            </span>
          ))}
        </span>
      );

    case "table":
      return (
        <span className="flex h-full flex-col gap-1">
          <span className="grid grid-cols-4 gap-1">
            {Array.from({ length: 4 }, (_, index) => (
              <span key={index} className={cn("block h-1 rounded-full", GHOST_STRONG)} />
            ))}
          </span>
          {[0, 1, 2].map((row) => (
            <span key={row} className="grid grid-cols-4 gap-1">
              {Array.from({ length: 4 }, (_, index) => (
                <span key={index} className={cn("block h-1 rounded-full", GHOST)} />
              ))}
            </span>
          ))}
        </span>
      );

    case "heatmap":
      return (
        <span className="grid h-full grid-cols-6 grid-rows-2 gap-1">
          {HEAT_OPACITIES.map((opacity, index) => (
            <span
              key={index}
              className="bg-brand rounded-[2px]"
              style={{ opacity: opacity + 0.06 }}
            />
          ))}
        </span>
      );

    case "donut":
      return (
        <span className="flex h-full items-center gap-2">
          <svg viewBox="0 0 36 36" className="h-full">
            <circle
              cx={18}
              cy={18}
              r={13}
              className="stroke-foreground/12 fill-none"
              strokeWidth={7}
            />
            <circle
              cx={18}
              cy={18}
              r={13}
              className="stroke-brand/45 fill-none"
              strokeWidth={7}
              strokeDasharray="52 82"
              transform="rotate(-90 18 18)"
            />
          </svg>
          <span className="flex flex-1 flex-col gap-1.5">
            <Line className="w-full" />
            <Line className="w-3/4" />
            <Line className="w-1/2" />
          </span>
        </span>
      );

    case "map":
      return (
        <span className="relative block h-full overflow-hidden rounded">
          <svg viewBox="0 0 96 40" className="size-full" preserveAspectRatio="none">
            <path
              d="M0 28 Q16 12 34 20 T64 14 T96 24"
              className="stroke-foreground/12 fill-none"
              strokeWidth={2}
            />
            <path
              d="M0 36 Q24 26 46 32 T96 30"
              className="stroke-foreground/10 fill-none"
              strokeWidth={2}
            />
            <circle cx={22} cy={19} r={2.5} className="fill-brand/50" />
            <circle cx={54} cy={16} r={2.5} className="fill-brand/50" />
            <circle cx={78} cy={26} r={2.5} className="fill-brand/50" />
          </svg>
        </span>
      );

    case "text":
      return (
        <span className="flex h-full flex-col justify-center gap-1.5">
          <Line className="w-1/4" />
          <Line className="w-full" />
          <Line className="w-5/6" />
          <Line className="w-2/3" />
        </span>
      );

    case "link":
      return (
        <span className="border-foreground/10 flex h-full items-center gap-2 rounded border px-2">
          <span className={cn("size-5 shrink-0 rounded", GHOST_BRAND)} />
          <span className="flex flex-1 flex-col gap-1">
            <Line className="w-2/3" />
            <Line className="w-1/3" />
          </span>
          <span className={cn("size-2 shrink-0 rotate-45 rounded-[1px]", GHOST_STRONG)} />
        </span>
      );

    default:
      return null;
  }
}
