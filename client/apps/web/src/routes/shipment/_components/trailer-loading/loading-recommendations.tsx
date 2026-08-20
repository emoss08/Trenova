import { cn } from "@trenova/shared/lib/utils";
import type { LoadingRecommendation } from "@/types/loading-optimization";
import { LightbulbIcon, ShieldAlertIcon, TrendingUpIcon } from "lucide-react";

const priorityConfig = {
  critical: {
    icon: ShieldAlertIcon,
    badge: "bg-destructive/15 text-destructive",
    iconColor: "text-destructive",
    label: "Critical",
  },
  suggested: {
    icon: LightbulbIcon,
    badge: "bg-warning/15 text-warning",
    iconColor: "text-warning",
    label: "Suggested",
  },
  optimization: {
    icon: TrendingUpIcon,
    badge: "bg-primary/15 text-primary",
    iconColor: "text-primary",
    label: "Tip",
  },
} as const;

export function LoadingRecommendations({
  recommendations,
}: {
  recommendations: LoadingRecommendation[];
}) {
  if (recommendations.length === 0) return null;

  return (
    <div className="space-y-1.5">
      {recommendations.map((rec, idx) => {
        const config = priorityConfig[rec.priority];
        const Icon = config.icon;
        return (
          <div
            key={idx}
            className="border-border bg-card flex gap-2.5 rounded-md border px-3 py-2"
          >
            <Icon
              className={cn("mt-0.5 size-3.5 shrink-0", config.iconColor)}
            />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="text-foreground text-xs font-semibold">
                  {rec.title}
                </span>
                <span
                  className={cn(
                    "rounded-full px-1.5 py-px text-[9px] font-medium",
                    config.badge,
                  )}
                >
                  {config.label}
                </span>
              </div>
              <p className="text-muted-foreground mt-0.5 text-xs leading-relaxed">
                {rec.description}
              </p>
              {rec.impact && (
                <p className="text-2xs text-foreground/60 mt-0.5 font-medium">
                  {rec.impact}
                </p>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
