import { cn } from "@trenova/shared/lib/utils";
import type { LoadingWarning } from "@/types/loading-optimization";
import { AlertCircleIcon, AlertTriangleIcon, InfoIcon } from "lucide-react";

const severityConfig = {
  error: { icon: AlertCircleIcon, color: "text-destructive" },
  warning: { icon: AlertTriangleIcon, color: "text-warning" },
  info: { icon: InfoIcon, color: "text-muted-foreground" },
} as const;

export function LoadingWarnings({ warnings }: { warnings: LoadingWarning[] }) {
  if (warnings.length === 0) return null;

  return (
    <div className="space-y-1">
      {warnings.map((warning, idx) => {
        const config = severityConfig[warning.severity];
        const Icon = config.icon;
        return (
          <div
            key={idx}
            className="border-border bg-card flex items-start gap-2 rounded-md border px-3 py-2"
          >
            <Icon className={cn("mt-0.5 size-3.5 shrink-0", config.color)} />
            <span className="text-muted-foreground text-xs leading-relaxed">{warning.message}</span>
          </div>
        );
      })}
    </div>
  );
}
