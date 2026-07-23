import { cn } from "@trenova/shared/lib/utils";
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
      <h2 className="text-xs font-semibold tracking-wide text-muted-foreground uppercase">
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
        "group relative flex flex-col rounded-lg border border-border bg-card p-4",
        "transition-[border-color,box-shadow,background-color] duration-200",
        "hover:border-brand hover:bg-muted hover:ring-2 hover:ring-brand/25",
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
      className="col-span-full flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border py-16"
    >
      <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
        <Icon className="size-5 text-muted-foreground" strokeWidth={1.75} />
      </div>
      <div className="text-center">
        <p className="text-sm font-medium">{title}</p>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      {action}
    </m.div>
  );
}
