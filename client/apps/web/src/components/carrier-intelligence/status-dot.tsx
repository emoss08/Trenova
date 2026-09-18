import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

export type StatusTone = "critical" | "high" | "medium" | "low" | "info" | "success" | "neutral";

const TONE_CLASSES: Record<StatusTone, string> = {
  critical: "bg-danger",
  high: "bg-warning",
  medium: "bg-warning",
  low: "bg-accent-sky",
  info: "bg-muted-foreground/60",
  success: "bg-success",
  neutral: "bg-muted-foreground/40",
};

export type StatusDotProps = {
  tone: StatusTone;
  pulse?: boolean;
  className?: string;
};

export function StatusDot({ tone, pulse = false, className }: StatusDotProps) {
  return (
    <span aria-hidden="true" className={cn("relative inline-flex size-2 shrink-0", className)}>
      {pulse && (
        <span
          className={cn(
            "absolute inline-flex size-full animate-ping rounded-full opacity-60",
            TONE_CLASSES[tone],
          )}
        />
      )}
      <span className={cn("relative inline-flex size-2 rounded-full", TONE_CLASSES[tone])} />
    </span>
  );
}

export function severityTone(severity: string | null | undefined): StatusTone {
  switch ((severity ?? "").toLowerCase()) {
    case "critical":
    case "blocker":
      return "critical";
    case "high":
      return "high";
    case "medium":
    case "warning":
      return "medium";
    case "low":
      return "low";
    case "info":
    case "advisory":
      return "info";
    default:
      return "neutral";
  }
}

export function riskTone(level: string | null | undefined): StatusTone {
  switch ((level ?? "").toLowerCase()) {
    case "veryhigh":
    case "critical":
      return "critical";
    case "high":
      return "high";
    case "elevated":
    case "moderate":
    case "medium":
      return "medium";
    case "low":
      return "success";
    default:
      return "neutral";
  }
}

export type SeverityLabelProps = {
  severity: string | null | undefined;
  label?: string;
  className?: string;
};

export function SeverityLabel({ severity, label, className }: SeverityLabelProps) {
  const t = useT();
  const fallback: Record<StatusTone, string> = {
    critical: t("Critical"),
    high: t("High"),
    medium: t("Medium"),
    low: t("Low"),
    info: t("Info"),
    success: t("Low"),
    neutral: t("Unknown"),
  };
  const tone = severityTone(severity);
  return (
    <span className={cn("inline-flex items-center gap-2 text-xs", className)}>
      <StatusDot tone={tone} />
      <span className="text-muted-foreground">{label ?? fallback[tone]}</span>
    </span>
  );
}

export type RiskLabelProps = {
  level: string | null | undefined;
  showDot?: boolean;
  className?: string;
};

export function RiskLabel({ level, showDot = true, className }: RiskLabelProps) {
  const t = useT();
  const labels: Record<string, string> = {
    low: t("Low risk"),
    moderate: t("Moderate risk"),
    elevated: t("Elevated risk"),
    high: t("High risk"),
    veryhigh: t("Very high risk"),
  };
  const text = level ? (labels[level.toLowerCase()] ?? t("Risk unknown")) : t("Not vetted");
  return (
    <span className={cn("inline-flex items-center gap-2 text-xs", className)}>
      {showDot && <StatusDot tone={riskTone(level)} />}
      <span className="text-muted-foreground">{text}</span>
    </span>
  );
}
