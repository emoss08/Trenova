import type { PTOStatus, PTOType } from "../types/worker";

export type PTOTypeMeta = {
  label: string;
  badgeVariant: "info" | "danger" | "success" | "neutral";
  barClass: string;
  dotClass: string;
  accentClass: string;
};

export const PTO_TYPE_META: Record<PTOType, PTOTypeMeta> = {
  Vacation: {
    label: "Vacation",
    badgeVariant: "info",
    barClass: "bg-accent-violet/80 text-white",
    dotClass: "bg-accent-violet",
    accentClass: "from-accent-violet to-accent-violet/5",
  },
  Sick: {
    label: "Sick",
    badgeVariant: "danger",
    barClass: "bg-danger/80 text-white",
    dotClass: "bg-danger",
    accentClass: "from-danger to-danger/5",
  },
  Holiday: {
    label: "Holiday",
    badgeVariant: "info",
    barClass: "bg-info/80 text-white",
    dotClass: "bg-info",
    accentClass: "from-info to-info/5",
  },
  Bereavement: {
    label: "Bereavement",
    badgeVariant: "success",
    barClass: "bg-success/80 text-white",
    dotClass: "bg-success",
    accentClass: "from-success to-success/5",
  },
  Maternity: {
    label: "Maternity",
    badgeVariant: "info",
    barClass: "bg-accent-rose/80 text-white",
    dotClass: "bg-accent-rose",
    accentClass: "from-accent-rose to-accent-rose/5",
  },
  Paternity: {
    label: "Paternity",
    badgeVariant: "info",
    barClass: "bg-accent-teal/80 text-white",
    dotClass: "bg-accent-teal",
    accentClass: "from-accent-teal to-accent-teal/5",
  },
  Personal: {
    label: "Personal",
    badgeVariant: "neutral",
    barClass: "bg-accent-slate/80 text-foreground-on-solid",
    dotClass: "bg-accent-slate",
    accentClass: "from-accent-slate to-accent-slate/5",
  },
};

export function ptoTypeMeta(type: PTOType | string): PTOTypeMeta {
  return (
    PTO_TYPE_META[type as PTOType] ?? {
      label: String(type),
      badgeVariant: "secondary",
      barClass: "bg-muted-foreground/60 text-white",
      dotClass: "bg-muted-foreground",
      accentClass: "from-muted-foreground/30 to-transparent",
    }
  );
}

export const PTO_STATUS_BAR_CLASS: Record<PTOStatus, string> = {
  Requested: "border-dashed",
  Approved: "",
  Rejected: "line-through opacity-40",
  Cancelled: "line-through opacity-40",
};

/**
 * Renders a signed day figure at ledger precision. PTO decimals arrive as
 * strings so no balance loses a hundredth on the way through JSON.
 */
export function formatPtoDays(value: string, fractionDigits = 2): string {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed.toFixed(fractionDigits) : value;
}

/**
 * Renders a day figure as a headline: whole days stay whole and a fraction is
 * kept to one place. A tile reporting "312.00 days" reads like an invoice line
 * rather than a total.
 */
export function formatPtoDayTotal(value: string): string {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return value;
  return parsed.toLocaleString("en-US", { maximumFractionDigits: 1 });
}
