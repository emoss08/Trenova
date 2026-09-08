import type { PTOStatus, PTOType } from "../types/worker";

export type PTOTypeMeta = {
  label: string;
  badgeVariant: "purple" | "inactive" | "info" | "active" | "pink" | "teal" | "secondary";
  barClass: string;
  dotClass: string;
  accentClass: string;
};

export const PTO_TYPE_META: Record<PTOType, PTOTypeMeta> = {
  Vacation: {
    label: "Vacation",
    badgeVariant: "purple",
    barClass: "bg-purple-600/80 text-white",
    dotClass: "bg-purple-600",
    accentClass: "from-purple-600 to-purple-600/5",
  },
  Sick: {
    label: "Sick",
    badgeVariant: "inactive",
    barClass: "bg-red-600/80 text-white",
    dotClass: "bg-red-600",
    accentClass: "from-red-600 to-red-600/5",
  },
  Holiday: {
    label: "Holiday",
    badgeVariant: "info",
    barClass: "bg-blue-600/80 text-white",
    dotClass: "bg-blue-600",
    accentClass: "from-blue-600 to-blue-600/5",
  },
  Bereavement: {
    label: "Bereavement",
    badgeVariant: "active",
    barClass: "bg-green-600/80 text-white",
    dotClass: "bg-green-600",
    accentClass: "from-green-600 to-green-600/5",
  },
  Maternity: {
    label: "Maternity",
    badgeVariant: "pink",
    barClass: "bg-pink-600/80 text-white",
    dotClass: "bg-pink-600",
    accentClass: "from-pink-600 to-pink-600/5",
  },
  Paternity: {
    label: "Paternity",
    badgeVariant: "teal",
    barClass: "bg-teal-600/80 text-white",
    dotClass: "bg-teal-600",
    accentClass: "from-teal-600 to-teal-600/5",
  },
  Personal: {
    label: "Personal",
    badgeVariant: "secondary",
    barClass: "bg-slate-500/80 text-white",
    dotClass: "bg-slate-500",
    accentClass: "from-slate-500 to-slate-500/5",
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
