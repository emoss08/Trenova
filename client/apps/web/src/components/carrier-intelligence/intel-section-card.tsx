import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelSection } from "@trenova/graphql/generated/graphql";
import { isSectionCovered } from "@/lib/carrier-intelligence";
import { formatNumber } from "@trenova/shared/i18n/format";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { LucideIcon } from "lucide-react";
import { useId, type ReactNode } from "react";
import { SectionUnavailable } from "./section-unavailable";

export type IntelSectionPart = {
  section: CarrierIntelSection;
  hasData: boolean;
  content: ReactNode;
  heading?: string;
};

export type IntelSectionCardProps = {
  title: string;
  icon: LucideIcon;
  coverage: readonly CarrierIntelSection[];
  provider: string | null | undefined;
  parts: IntelSectionPart[];
  headerAction?: ReactNode;
  emphasis?: "none" | "warning" | "danger";
  className?: string;
};

const EMPHASIS_CLASSES = {
  none: "",
  warning: "border-yellow-600/40",
  danger: "border-red-600/50",
} as const;

export function IntelSectionCard({
  title,
  icon: Icon,
  coverage,
  provider,
  parts,
  headerAction,
  emphasis = "none",
  className,
}: IntelSectionCardProps) {
  const headingId = useId();

  return (
    <section
      aria-labelledby={headingId}
      className={cn(
        "bg-card flex flex-col overflow-hidden rounded-lg border",
        EMPHASIS_CLASSES[emphasis],
        className,
      )}
    >
      <header className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <Icon className="text-muted-foreground size-3.5" aria-hidden />
          <h3 id={headingId} className="text-sm font-medium">
            {title}
          </h3>
        </div>
        {headerAction}
      </header>
      <div className="flex flex-col gap-3 p-3">
        {parts.map((part) => (
          <div key={part.section} className="flex flex-col gap-2" data-section={part.section}>
            {part.heading ? (
              <h4 className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                {part.heading}
              </h4>
            ) : null}
            {!isSectionCovered(coverage, part.section) ? (
              <SectionUnavailable provider={provider} reason="not-covered" />
            ) : !part.hasData ? (
              <SectionUnavailable provider={provider} reason="empty" />
            ) : (
              part.content
            )}
          </div>
        ))}
      </div>
    </section>
  );
}

export function IntelFieldGrid({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <dl className={cn("grid grid-cols-1 gap-x-4 gap-y-2 sm:grid-cols-2", className)}>{children}</dl>
  );
}

export function IntelField({
  label,
  children,
  className,
}: {
  label: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-0.5", className)}>
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd className="truncate text-sm">{children}</dd>
    </div>
  );
}

export function IntelValue({ value }: { value: string | null | undefined }) {
  if (value === null || value === undefined || value.trim() === "") {
    return <span className="text-muted-foreground">-</span>;
  }
  return <>{value}</>;
}

export function IntelNumber({
  value,
  options,
}: {
  value: number | null | undefined;
  options?: Intl.NumberFormatOptions;
}) {
  if (value === null || value === undefined) {
    return <span className="text-muted-foreground">-</span>;
  }
  return <span className="tabular-nums">{formatNumber(value, options)}</span>;
}

export function IntelDate({ value }: { value: number | null | undefined }) {
  if (!value) {
    return <span className="text-muted-foreground">-</span>;
  }
  return <>{formatUnixDateMedium(value)}</>;
}

export function IntelBoolean({
  value,
  trueTone = "neutral",
}: {
  value: boolean | null | undefined;
  trueTone?: "neutral" | "good" | "bad";
}) {
  const t = useT();

  if (value === null || value === undefined) {
    return <span className="text-muted-foreground">-</span>;
  }

  return (
    <span
      className={cn(
        value && trueTone === "good" && "text-green-700 dark:text-green-400",
        value && trueTone === "bad" && "font-medium text-red-700 dark:text-red-400",
      )}
    >
      {value ? t("Yes") : t("No")}
    </span>
  );
}
