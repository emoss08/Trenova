import { cn } from "@trenova/shared/lib/utils";
import { ArrowDownIcon, ArrowUpIcon } from "lucide-react";
import { createContext, useContext } from "react";
import { Link } from "react-router";
import type React from "react";
import type { Tone } from "./tone";

const KpiStripContext = createContext(false);

export function useInKpiStrip(): boolean {
  return useContext(KpiStripContext);
}

const TONE_DOT: Record<Tone, string> = {
  success: "bg-success",
  danger: "bg-danger",
  warning: "bg-warning",
  brand: "bg-brand",
  info: "bg-info",
  muted: "bg-foreground-subtle",
};

const TONE_TEXT: Record<Tone, string> = {
  success: "text-success-foreground",
  danger: "text-danger-foreground",
  warning: "text-warning-foreground",
  brand: "text-brand",
  info: "text-info-foreground",
  muted: "text-foreground-muted",
};

type KpiStripProps = {
  children: React.ReactNode;
  minItemWidth?: string;
  className?: string;
  "aria-label"?: string;
};

export function KpiStrip({
  children,
  minItemWidth = "10rem",
  className,
  "aria-label": ariaLabel,
}: KpiStripProps) {
  return (
    <KpiStripContext.Provider value>
      <div
        data-slot="kpi-strip"
        role="group"
        aria-label={ariaLabel}
        className={cn("border-border bg-card overflow-hidden rounded-lg border", className)}
      >
        <div
          className="-mr-px -mb-px grid"
          style={{
            gridTemplateColumns: `repeat(auto-fit, minmax(min(${minItemWidth}, 100%), 1fr))`,
          }}
        >
          {children}
        </div>
      </div>
    </KpiStripContext.Provider>
  );
}

export const KPI_STRIP_CELL_CLASS = "border-border min-w-0 border-r border-b px-3 py-2.5";

export const KPI_VALUE_CLASS = "text-xl leading-none font-semibold tabular-nums";

export const KPI_VALUE_LG_CLASS = "text-2xl leading-none font-semibold tabular-nums";

const KPI_INTERACTIVE_CLASS =
  "ui-inset-focus-ring hover:bg-surface-hover cursor-pointer text-left transition-colors";

const KPI_LONE_CELL_CLASS = "border-border bg-card min-w-0 rounded-lg border px-3 py-2.5";

type KpiStripItemProps = {
  label: React.ReactNode;
  value: React.ReactNode;
  sub?: React.ReactNode;
  info?: React.ReactNode;
  tone?: Tone;
  size?: "md" | "lg";
  delta?: number | null;
  deltaLabel?: string;
  deltaTone?: Tone;
  hint?: string;
  active?: boolean;
  onClick?: () => void;
  to?: string;
  className?: string;
};

export function KpiStripItem({
  label,
  value,
  sub,
  info,
  tone,
  size = "md",
  delta,
  deltaLabel,
  deltaTone,
  hint,
  active = false,
  onClick,
  to,
  className,
}: KpiStripItemProps) {
  const cellClass = useInKpiStrip() ? KPI_STRIP_CELL_CLASS : KPI_LONE_CELL_CLASS;
  const body = (
    <>
      <div className="text-foreground-subtle flex min-w-0 items-center gap-1.5 text-xs font-medium">
        {tone ? (
          <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", TONE_DOT[tone])} />
        ) : null}
        <span className="truncate">{label}</span>
        {info}
      </div>
      <div
        className={cn(
          "text-foreground mt-1.5 truncate",
          size === "lg" ? KPI_VALUE_LG_CLASS : KPI_VALUE_CLASS,
        )}
      >
        {value}
      </div>
      {sub != null || delta != null ? (
        <div className="text-foreground-muted mt-0.5 flex min-w-0 items-center gap-1.5 text-xs">
          <KpiDelta delta={delta} label={deltaLabel} tone={deltaTone} />
          {sub != null ? <span className="truncate">{sub}</span> : null}
        </div>
      ) : null}
    </>
  );

  if (to) {
    return (
      <Link
        to={to}
        title={hint}
        className={cn(cellClass, KPI_INTERACTIVE_CLASS, "block", className)}
      >
        {body}
      </Link>
    );
  }

  if (onClick) {
    return (
      <button
        type="button"
        title={hint}
        aria-pressed={active}
        onClick={onClick}
        className={cn(
          cellClass,
          KPI_INTERACTIVE_CLASS,
          active && "bg-surface-selected hover:bg-surface-selected",
          className,
        )}
      >
        {body}
      </button>
    );
  }

  return (
    <div title={hint} className={cn(cellClass, className)}>
      {body}
    </div>
  );
}

type KpiDeltaProps = {
  delta: number | null | undefined;
  label?: string;
  tone?: Tone;
};

export function KpiDelta({ delta, label, tone }: KpiDeltaProps) {
  if (delta === undefined || delta === null) return null;
  const positive = delta >= 0;
  const resolved: Tone = tone ?? (positive ? "success" : "danger");
  const Icon = positive ? ArrowUpIcon : ArrowDownIcon;

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-0.5 font-medium tabular-nums",
        TONE_TEXT[resolved],
      )}
    >
      <Icon aria-hidden className="size-3" strokeWidth={2.25} />
      {Math.abs(delta)}
      {label ?? ""}
    </span>
  );
}
