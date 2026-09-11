import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { XIcon } from "lucide-react";
import {
  FileChartColumnIcon,
  ReceiptTextIcon,
  ShieldCheckIcon,
  TruckIcon,
  WrenchIcon,
  type LucideIcon,
} from "lucide-react";
import { m } from "motion/react";
import type { ReactNode } from "react";

type CategoryChrome = {
  icon: LucideIcon;
  tile: string;
};

const CATEGORY_CHROME: Record<string, CategoryChrome> = {
  operations: { icon: TruckIcon, tile: "bg-blue-500/10 text-blue-600 dark:text-blue-400" },
  billing: {
    icon: ReceiptTextIcon,
    tile: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  },
  compliance: { icon: ShieldCheckIcon, tile: "bg-amber-500/10 text-amber-600 dark:text-amber-400" },
  fleet: { icon: WrenchIcon, tile: "bg-violet-500/10 text-violet-600 dark:text-violet-400" },
};

const DEFAULT_CHROME: CategoryChrome = {
  icon: FileChartColumnIcon,
  tile: "bg-muted text-muted-foreground",
};

export function categoryChrome(category: string): CategoryChrome {
  return CATEGORY_CHROME[category.toLowerCase()] ?? DEFAULT_CHROME;
}

export function CategoryTile({ category, className }: { category: string; className?: string }) {
  const chrome = categoryChrome(category);
  const Icon = chrome.icon;
  return (
    <div
      className={cn(
        "flex size-8 shrink-0 items-center justify-center rounded-md",
        chrome.tile,
        className,
      )}
    >
      <Icon className="size-4" strokeWidth={1.75} />
    </div>
  );
}

export function CategoryGroupHeader({
  label,
  count,
  noun,
}: {
  label: string;
  count: number;
  noun: string;
}) {
  return (
    <div className="flex items-center gap-2">
      <h2 className="text-muted-foreground text-xs font-semibold tracking-wide uppercase">
        {label}
      </h2>
      <span className="text-2xs text-muted-foreground/70 tabular-nums">
        {count} {count === 1 ? noun : `${noun}s`}
      </span>
    </div>
  );
}

export function ReportCard({
  children,
  index,
  onClick,
  className,
}: {
  children: ReactNode;
  index: number;
  onClick?: () => void;
  className?: string;
}) {
  return (
    <m.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, delay: Math.min(index, 12) * 0.03, ease: "easeOut" }}
      onClick={onClick}
      className={cn(
        "group border-border bg-card relative flex flex-col rounded-lg border p-4",
        "transition-[border-color,box-shadow,background-color] duration-200",
        "hover:border-brand hover:bg-muted hover:ring-brand/25 hover:ring-2",
        onClick && "cursor-pointer",
        className,
      )}
    >
      {children}
    </m.div>
  );
}

export function ReportGridEmptyState({
  icon: Icon,
  title,
  description,
  action,
}: {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <m.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
      className="border-border col-span-full flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed py-16"
    >
      <div className="bg-muted flex size-10 items-center justify-center rounded-lg">
        <Icon className="text-muted-foreground size-5" strokeWidth={1.75} />
      </div>
      <div className="text-center">
        <p className="text-sm font-medium">{title}</p>
        <p className="text-muted-foreground text-xs">{description}</p>
      </div>
      {action}
    </m.div>
  );
}

/**
 * A report is a question with an answer drawn under it: each ghost card is a
 * document with a title, a preview of the figure it produces, a bar chart or
 * a few table rows, and the run control at the foot.
 */
const GHOST_REPORTS: readonly { name: string; preview: "bars" | "table" | "line" }[] = [
  { name: "w-3/5", preview: "bars" },
  { name: "w-2/5", preview: "table" },
  { name: "w-1/2", preview: "line" },
];
const GHOST_BARS = [40, 70, 55, 85, 60] as const;
const GHOST_TABLE_ROWS = ["w-4/5", "w-3/5", "w-full"] as const;
const GHOST_LINE = "0,26 15,20 30,22 45,14 60,16 75,9 90,12 105,5";

/**
 * A dashboard is one canvas: a filter bar across the top and widgets of
 * different sizes beneath it, each with the shape of what it shows.
 */
const GHOST_WIDGETS: readonly { span: string; kind: "number" | "bars" | "line" | "table" }[] = [
  { span: "col-span-1", kind: "number" },
  { span: "col-span-1", kind: "number" },
  { span: "col-span-2", kind: "line" },
  { span: "col-span-2", kind: "bars" },
  { span: "col-span-2", kind: "table" },
];

function GhostPill({ className }: { className?: string }) {
  return <span className={cn("border-border/70 block h-4 rounded-md border", className)} />;
}

function GhostPreview({ kind }: { kind: "number" | "bars" | "line" | "table" }) {
  switch (kind) {
    case "bars":
      return (
        <span className="flex h-12 items-end gap-1">
          {GHOST_BARS.map((height, index) => (
            <span
              key={index}
              className="bg-brand/25 block flex-1 rounded-t-sm"
              style={{ height: `${height}%` }}
            />
          ))}
        </span>
      );
    case "line":
      return (
        <svg viewBox="0 0 105 30" className="h-12 w-full" preserveAspectRatio="none">
          <polyline
            points={GHOST_LINE}
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            className="text-brand/40"
            vectorEffect="non-scaling-stroke"
          />
        </svg>
      );
    case "table":
      return (
        <span className="divide-border/60 flex h-12 flex-col justify-center divide-y divide-dashed">
          {GHOST_TABLE_ROWS.map((width, index) => (
            <span key={index} className="flex items-center justify-between gap-2 py-1">
              <GhostLine className={width} />
              <GhostLine className="w-6 shrink-0" />
            </span>
          ))}
        </span>
      );
    default:
      return (
        <span className="flex h-12 flex-col justify-center gap-1.5">
          <GhostLine className="h-4 w-14" />
          <GhostLine className="w-10" />
        </span>
      );
  }
}

function GhostReportDocuments() {
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 px-1">
        <GhostLine className="w-20" />
        <GhostLine className="w-4" />
      </div>
      <div className="grid grid-cols-3 gap-3">
        {GHOST_REPORTS.map((report, index) => (
          <div
            key={index}
            className="border-border/70 bg-card flex flex-col gap-3 rounded-lg border p-3"
          >
            <span className="flex items-center gap-2">
              <span className="bg-muted size-5 shrink-0 rounded-md" />
              <GhostLine className={`h-2 ${report.name}`} />
            </span>
            <span className="bg-muted/30 rounded-md px-2 py-1.5">
              <GhostPreview kind={report.preview} />
            </span>
            <span className="flex items-center justify-between gap-2">
              <GhostPill className="w-12" />
              <span className="bg-brand/30 block h-5 w-10 rounded-md" />
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function GhostDashboardCanvas() {
  return (
    <div className="border-border/70 bg-card flex flex-col gap-3 rounded-lg border p-3">
      <div className="flex items-center gap-2 border-b pb-3">
        <GhostLine className="h-2 w-24" />
        <span className="flex-1" />
        <GhostPill className="w-16" />
        <GhostPill className="w-12" />
        <GhostPill className="w-14" />
      </div>
      <div className="grid grid-cols-4 gap-2">
        {GHOST_WIDGETS.map((widget, index) => (
          <span
            key={index}
            className={cn(
              "border-border/60 flex flex-col gap-1.5 rounded-md border border-dashed p-2",
              widget.span,
            )}
          >
            <GhostLine className="w-1/2" />
            <GhostPreview kind={widget.kind} />
          </span>
        ))}
      </div>
    </div>
  );
}

/**
 * A report grid drawn as what it will become: report documents under a
 * category heading, each with a preview of its figure, or one dashboard
 * canvas with a filter bar and its widgets. The words underneath say what
 * fills it; the action is the way in, or the way back from a filter.
 */
export function ReportGridEmpty({
  variant,
  title,
  description,
  onClearFilters,
  action,
}: {
  variant: "reports" | "dashboards";
  title: string;
  description: string;
  onClearFilters?: () => void;
  action?: ReactNode;
}) {
  const t = useT();

  return (
    <EmptySheet
      className="col-span-full"
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onClearFilters ? (
          <Button variant="outline" size="sm" onClick={onClearFilters}>
            <XIcon className="size-3.5" />
            {t("Clear filters")}
          </Button>
        ) : (
          action
        )
      }
      sketch={
        <div className="text-left">
          {variant === "reports" ? <GhostReportDocuments /> : <GhostDashboardCanvas />}
        </div>
      }
    />
  );
}
